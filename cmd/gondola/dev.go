package main

import (
	"bufio"
	"bytes"
	"code.google.com/p/go.exp/fsnotify"
	"fmt"
	"go/build"
	"gondola/admin"
	"gondola/config"
	"gondola/log"
	"gondola/mux"
	"gondola/runtimeutil"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func exitStatus(p *os.ProcessState) int {
	ws := p.Sys().(syscall.WaitStatus)
	return ws.ExitStatus()
}

func cmdString(cmd *exec.Cmd) string {
	return strings.Join(cmd.Args, " ")
}

func supportsRace() bool {
	return runtime.GOARCH == "amd64" && (runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "windows")
}

func randomFreePort() int {
	for {
		mp := rand.Intn(65000)
		if mp < 10000 {
			continue
		}
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", mp))
		if err == nil {
			listener.Close()
			return mp
		}
	}
	panic("unreachable")
}

type BuildError struct {
	Package  string
	Filename string
	Line     int
	Error    string
}

func (b *BuildError) Location() string {
	return fmt.Sprintf("%s, line %d", b.Filename, b.Line)
}

func (b *BuildError) Code() template.HTML {
	s, err := runtimeutil.FormatSourceHTML(b.Filename, b.Line, 5, true, true)
	if err != nil {
		log.Errorf("Error formatting code from %s: %s", b.Filename, err)
	}
	return s
}

func NewProject(dir string, config string, tags string, race bool) *Project {
	p := &Project{
		dir:    dir,
		config: config,
		tags:   tags,
		race:   race,
	}
	m := mux.New()
	m.Logger = nil
	m.SetTemplatesLoader(assets)
	m.HandleFunc("/", p.Handler)
	m.SetPort(randomFreePort())
	go func() {
		m.MustListenAndServe()
	}()
	return p
}

type Project struct {
	sync.Mutex
	dir            string
	config         string
	configFilename string
	tags           string
	race           bool
	port           int
	muxPort        int
	building       bool
	errors         []*BuildError
	cmd            *exec.Cmd
	watcher        *fsnotify.Watcher
	proxied        map[net.Conn]struct{}
}

func (p *Project) Name() string {
	return filepath.Base(p.dir)
}

func (p *Project) ConfDir() string {
	return filepath.Join(p.dir, "conf")
}

func (p *Project) Conf() string {
	if p.configFilename == "" {
		p.configFilename = p.configFile()
	}
	return p.configFilename
}

func (p *Project) configFile() string {
	dir := p.ConfDir()
	// If a config name was provided, try to use it
	if p.config != "" {
		var filename string
		if filepath.IsAbs(p.config) {
			filename = p.config
		} else {
			filename = filepath.Join(dir, p.config)
		}
		if _, err := os.Stat(filename); err == nil {
			return filename
		} else {
			log.Warningf("could not find config file %q (error was %s), ignoring", p.config, err)
		}
	}
	// Otherwise, try development.conf
	filename := filepath.Join(dir, "development.conf")
	if _, err := os.Stat(filename); err == nil {
		return filename
	}
	// Grab the first .conf file in the directory
	if files, err := ioutil.ReadDir(dir); err == nil {
		for _, v := range files {
			if strings.ToLower(filepath.Ext(v.Name())) == ".conf" {
				filename := filepath.Join(dir, v.Name())
				if _, err := os.Stat(filename); err == nil {
					return filename
				}
			}
		}
	}
	return ""
}

func (p *Project) importPackage(imported map[string]bool, pkgs *[]*build.Package, path string) error {
	if imported[path] {
		return nil
	}
	pkg, err := build.Import(path, p.dir, 0)
	if err != nil {
		return err
	}
	imported[path] = true
	*pkgs = append(*pkgs, pkg)
	for _, imp := range pkg.Imports {
		if imp == "C" {
			continue
		}
		if err := p.importPackage(imported, pkgs, imp); err != nil {
			return err
		}
	}
	return nil
}

func (p *Project) Packages() ([]*build.Package, error) {
	var pkgs []*build.Package
	imported := make(map[string]bool)
	err := p.importPackage(imported, &pkgs, ".")
	return pkgs, err
}

func (p *Project) StopMonitoring() {
	if p.watcher != nil {
		p.watcher.Close()
		p.watcher = nil
	}
}

func (p *Project) StartMonitoring() error {
	pkgs, err := p.Packages()
	if err != nil {
		return err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	for _, v := range pkgs {
		if err := watcher.Watch(v.Dir); err != nil {
			return err
		}
	}
	watcher.Watch(p.Conf())
	p.watcher = watcher
	go func() {
		var t *time.Timer
	finished:
		for {
			select {
			case ev := <-watcher.Event:
				if ev == nil {
					// Closed
					break finished
				}
				ext := filepath.Ext(strings.ToLower(ev.Name))
				if ext != ".go" && ext != ".h" && ext != ".c" && ext != ".s" && ext != ".cpp" && ext != ".cxx" {
					if ext == ".conf" {
						if ev.IsModify() {
							log.Debugf("Config file %s changed, restarting...", p.Conf())
							if err := p.Stop(); err != nil {
								log.Errorf("Error stopping %s: %s", p.Name(), err)
								break
							}
							if err := p.Start(); err != nil {
								log.Panicf("Error starting %s: %s", p.Name(), err)
							}
						} else if ev.IsDelete() {
							// It seems the Watcher stops watching a file
							// if it receives a DELETE event for it. For some
							// reason, some editors generate a DELETE event
							// for a file when saving it, so we must watch the
							// file again. Since fsnotify is in exp/ and its
							// API might change, remove the watch first, just
							// in case.
							watcher.RemoveWatch(ev.Name)
							watcher.Watch(ev.Name)
						}
					}
					break
				}
				if t != nil {
					t.Stop()
				}
				if p.building {
					break
				}
				t = time.AfterFunc(50*time.Millisecond, func() {
					p.Compile()
				})
			case err := <-watcher.Error:
				if err == nil {
					// Closed
					break finished
				}
				log.Errorf("Error watching: %s", err)
			}
		}
	}()
	return nil
}

func (p *Project) ProjectCmd() *exec.Cmd {
	name := p.Name()
	if runtime.GOOS != "windows" {
		name = "./" + name
	}
	cmd := exec.Command(name, "-debug", "-config", p.Conf(), fmt.Sprintf("-port=%d", p.port))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = p.dir
	return cmd
}

func (p *Project) Start() error {
	p.port = randomFreePort()
	cmd := p.ProjectCmd()
	log.Debugf("Starting %s (%s)", p.Name(), cmdString(cmd))
	p.cmd = cmd
	err := cmd.Start()
	go func() {
		cmd.Wait()
		if cmd == p.cmd {
			// Othewise the process was intentionally killed
			if s := cmd.ProcessState; s != nil && !s.Success() {
				log.Warningf("%s exited with code %d. Restarting...", p.Name(), exitStatus(s))
				time.Sleep(500 * time.Millisecond)
				go p.Start()
			}
		}
	}()
	return err
}

func (p *Project) Stop() error {
	var err error
	if p.cmd != nil {
		cmd := p.cmd
		p.cmd = nil
		if cmd.Process != nil {
			err = cmd.Process.Kill()
		}
		cmd.Wait()
	}
	return err
}

func (p *Project) GoCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	cmd.Dir = p.dir
	return cmd
}

func (p *Project) CompilerCmd() *exec.Cmd {
	// -e reports all the errors
	args := []string{"build", "-gcflags", "-e"}
	if p.race && supportsRace() {
		args = append(args, "-race")
	}
	if p.tags != "" {
		args = append(args, []string{"-tags", p.tags}...)
	}
	lib := filepath.Join(p.dir, "lib")
	if st, err := os.Stat(lib); err == nil && st.IsDir() {
		// If there's a lib directory, add it to rpath
		args = append(args, []string{"-ldflags", "-r lib"}...)
	}
	return p.GoCmd(args...)
}

func (p *Project) Compile() {
	p.Lock()
	defer p.Unlock()
	// Browsers might keep connections open
	// after the request is served and they
	// might need to be rerouted (e.g. the
	// project had errors and it doesn't have
	// them anymore).
	p.DropConnections()
	p.StopMonitoring()
	p.building = true
	if err := p.Stop(); err != nil {
		log.Panic(err)
	}
	p.errors = nil
	cmd := p.CompilerCmd()
	log.Debugf("Building %s (%s)", p.Name(), cmdString(cmd))
	var buf bytes.Buffer
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			log.Panic(err)
		}
		if exitStatus(exitErr.ProcessState) != 2 {
			// gc returns 2 when there are compilation errors
			log.Panic(err)
		}
		r := bufio.NewReader(bytes.NewReader(buf.Bytes()))
		var pkg string
		for {
			eline, err := r.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Panic(err)
			}
			if strings.HasPrefix(eline, "#") {
				pkg = strings.TrimSpace(eline[1:])
			} else if strings.HasPrefix(eline, "\t") {
				// Info related to the previous error. Let it
				// crash if we don't have a previous error, just
				// in case there are any circumstances where a line
				// starting with \t means something else in the future.
				// This way the problem will be easier to catch.
				be := p.errors[len(p.errors)-1]
				be.Error += fmt.Sprintf(" (%s)", strings.TrimSpace(eline))
			} else {
				parts := strings.SplitN(eline, ":", 3)
				filename := filepath.Clean(filepath.Join(p.dir, parts[0]))
				line, err := strconv.Atoi(parts[1])
				if err != nil {
					log.Panic(err)
				}
				be := &BuildError{
					Package:  pkg,
					Filename: filename,
					Line:     line,
					Error:    strings.TrimSpace(parts[2]),
				}
				p.errors = append(p.errors, be)
			}
		}
	}
	if c := len(p.errors); c == 0 {
		// TODO: Report error when starting project via web
		if err := p.Start(); err != nil {
			log.Panic(err)
		}
	} else {
		log.Errorf("%d errors building %s", c, p.Name())
	}
	if err := p.StartMonitoring(); err != nil {
		log.Errorf("Error monitoring files for project %s: %s. Development server must be manually restarted.", p.Name(), err)
	}
	// Build dependencies, to speed up future builds
	go func() {
		args := []string{"test", "-i"}
		if p.race && supportsRace() {
			args = append(args, "-race")
		}
		p.GoCmd(args...).Run()
	}()
	p.building = false
}

func (p *Project) Handler(ctx *mux.Context) {
	data := map[string]interface{}{
		"Project": p,
		"Errors":  p.errors,
		"Count":   len(p.errors),
	}
	ctx.MustExecute("errors.html", data)
}

func (p *Project) DropConnections() {
	for k := range p.proxied {
		k.Close()
	}
	p.proxied = nil
}

func (p *Project) HandleConnection(conn net.Conn) {
	p.Lock()
	p.Unlock()
	if p.proxied == nil {
		p.proxied = make(map[net.Conn]struct{})
	}
	p.proxied[conn] = struct{}{}
	if len(p.errors) > 0 {
		p.ProxyConnection(conn, p.muxPort)
	} else {
		p.ProxyConnection(conn, p.port)
	}
}

func (p *Project) ProxyConnection(conn net.Conn, port int) {
	r := fmt.Sprintf("localhost:%d", port)
	sock, err := net.Dial("tcp", r)
	if err != nil {
		if oerr, ok := err.(*net.OpError); ok {
			if oerr.Err == syscall.ECONNREFUSED {
				// Wait a bit for the server to start
				time.Sleep(time.Second)
				sock, err = net.Dial("tcp", r)
			}
		}
		if err != nil {
			log.Errorf("Error proxying connection to %s: %s", r, err)
			conn.Close()
			return
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		io.Copy(sock, conn)
		// conn was closed by client
		sock.Close()
		wg.Done()
	}()
	go func() {
		io.Copy(conn, sock)
		// sock was closed by server
		conn.Close()
		wg.Done()
	}()
	wg.Wait()
	conn.Close()
	sock.Close()
}

func Dev(ctx *mux.Context) {
	log.SetLevel(log.LDebug)
	var dir string
	var configName string
	var tags string
	var race bool
	ctx.ParseParamValue("dir", &dir)
	ctx.ParseParamValue("config", &configName)
	ctx.ParseParamValue("tags", &tags)
	ctx.ParseParamValue("race", &race)
	path, err := filepath.Abs(dir)
	if err != nil {
		log.Panic(err)
	}
	p := NewProject(path, configName, tags, race)
	if c := p.Conf(); c == "" {
		log.Panicf("can't find configuration for %s. Please, create a config file in the directory %s (its extension must be .conf)", p.Name(), p.ConfDir())
	} else {
		log.Debugf("Using config file %s", c)
	}
	var port int
	ctx.ParseParamValue("port", &port)
	if port == 0 {
		var conf config.Config
		if err := config.ParseFile(p.Conf(), &conf); err == nil {
			port = conf.Port
		}
		if port == 0 {
			port = 8888
		}
	}
	go p.Compile()
	eof := "C"
	if runtime.GOOS == "windows" {
		eof = "Z"
	}
	log.Debugf("Starting Gondola development server on port %d (press Control+%s to exit)", port, eof)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Warningf("Error accepting connection: %s", err)
			continue
		}
		go p.HandleConnection(conn)
	}
}

func init() {
	admin.Register(Dev, &admin.Options{
		Help: "Starts the development server",
		Flags: admin.Flags(
			admin.StringFlag("dir", ".", "Directory of the project"),
			admin.StringFlag("config", "", "Configuration name to use. If none is provided, development.conf is used."),
			admin.StringFlag("tags", "", "Go build tags to pass to the compiler"),
			admin.IntFlag("port", 0, "Port to listen on. If zero, the project configuration is parsed to look for the port. If none is found, 8888 is used."),
			admin.BoolFlag("race", false, "Enable -race when building. If the platform does not support -race, this option is ignored."),
		),
	})
}
