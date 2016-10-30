package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"html/template"
	"io"
	"math/rand"
	"net"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gnd.la/app"
	"gnd.la/config"
	"gnd.la/internal/runtimeutil"
	"gnd.la/log"
	"gnd.la/util/stringutil"

	"github.com/rainycape/browser"
	"github.com/rainycape/command"
)

const (
	devConfigName = "dev.conf"
)

var (
	sourceExtensions = []string{
		".go",
		".h",
		".c",
		".s",
		".cpp",
		".cxx",
	}
	noColorRegexp = regexp.MustCompile("\x1b\\[\\d+;\\d+m(.*?)\x1b\\[00m")
	panicRe       = regexp.MustCompile("\npanic: (.+)")
)

func uncolor(s string) string {
	return noColorRegexp.ReplaceAllString(s, "$1")
}

func isSource(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, v := range sourceExtensions {
		if ext == v {
			return true
		}
	}
	return false
}

func formatTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return strconv.FormatInt(t.Unix(), 10)
}

func exitStatus(p *os.ProcessState) int {
	ws := p.Sys().(syscall.WaitStatus)
	return ws.ExitStatus()
}

func cmdString(cmd *exec.Cmd) string {
	return strings.Join(cmd.Args, " ")
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
	if b.Filename == "" {
		return ""
	}
	return fmt.Sprintf("%s, line %d", b.Filename, b.Line)
}

func (b *BuildError) Code() template.HTML {
	if b.Filename == "" {
		return template.HTML("")
	}
	s, err := runtimeutil.FormatSourceHTML(b.Filename, b.Line, 5, true, true)
	if err != nil {
		log.Errorf("error formatting code from %s: %s", b.Filename, err)
	}
	return s
}

func NewProject(dir string, config string) *Project {
	p := &Project{
		dir:        dir,
		configPath: config,
	}
	a := app.New()
	a.Config().Port = 8888
	a.Logger = nil
	a.SetTemplatesFS(devAssets)
	a.Handle("/_gondola_dev_server_status", p.StatusHandler)
	a.Handle("/", p.Handler)
	p.App = a
	return p
}

type Project struct {
	sync.Mutex
	App          *app.App
	dir          string
	configPath   string
	tags         string
	goFlags      string
	noDebug      bool
	noCache      bool
	profile      bool
	port         int
	proxy        *httputil.ReverseProxy
	proxyChecked bool
	buildCmd     *exec.Cmd
	errors       []*BuildError
	cmd          *exec.Cmd
	watcher      *fsWatcher
	built        time.Time
	started      time.Time
	// runtime info
	out      bytes.Buffer
	runError error
	exitCode int
}

func (p *Project) Listen() {
	os.Setenv("GONDOLA_IS_DEV_SERVER", "1")
	p.App.MustListenAndServe()
}

func (p *Project) Name() string {
	return filepath.Base(p.dir)
}

func (p *Project) buildTags() []string {
	tags := p.tags
	if p.profile {
		if tags == "" {
			tags = "profile"
		} else {
			tags += " profile"
		}
	}
	if tags != "" {
		return []string{"-tags", tags}
	}
	return nil
}

func (p *Project) importPackage(imported map[string]bool, pkgs *[]*build.Package, path string) []error {
	if imported[path] {
		return nil
	}
	pkg, err := build.Import(path, p.dir, 0)
	if err != nil {
		return []error{err}
	}
	imported[path] = true
	*pkgs = append(*pkgs, pkg)
	var errs []error
	for _, imp := range pkg.Imports {
		if imp == "C" {
			continue
		}
		if errs2 := p.importPackage(imported, pkgs, imp); len(errs2) > 0 {
			errs = append(errs, errs2...)
		}
	}
	return errs
}

// Packages returns the packages imported by the Project, either
// directly or transitively.
func (p *Project) Packages() ([]*build.Package, error) {
	var pkgs []*build.Package
	imported := make(map[string]bool)
	var err error
	errs := p.importPackage(imported, &pkgs, ".")
	if len(errs) > 0 {
		var msgs []string
		for _, v := range errs {
			if v != nil {
				msgs = append(msgs, v.Error())
			}
		}
		if len(msgs) > 0 {
			err = errors.New(strings.Join(msgs, ", "))
		}
	}
	return pkgs, err
}

func (p *Project) StopMonitoring() {
	if p.watcher != nil {
		p.watcher.Close()
		p.watcher = nil
	}
}

func (p *Project) StartMonitoring() error {
	watcher, err := newFSWatcher()
	if err != nil {
		return err
	}
	pkgs, err := p.Packages()
	if err != nil && len(pkgs) == 0 {
		// TODO: Show the packages which failed to be monitored
		return err
	}
	watcher.IsValidFile = func(path string) bool {
		return path == p.configPath || isSource(path)
	}
	watcher.Changed = func(path string) {
		if path == p.configPath {
			log.Infof("Config file %s changed, restarting...", p.configPath)
			if err := p.Stop(); err != nil {
				log.Errorf("Error stopping %s: %s", p.Name(), err)
			}
			if err := p.Start(); err != nil {
				log.Panicf("Error starting %s: %s", p.Name(), err)
			}
		} else {
			p.Build()
		}
	}
	if err := watcher.AddPackages(pkgs); err != nil {
		return err
	}
	if err := watcher.Add(p.configPath); err != nil {
		return err
	}
	p.watcher = watcher
	return nil
}

func (p *Project) ProjectCmd() *exec.Cmd {
	name := p.Name()
	if runtime.GOOS != "windows" {
		name = "./" + name
	}
	args := []string{"-config", p.configPath, fmt.Sprintf("-port=%d", p.port)}
	if p.noDebug {
		args = append(args, "-debug=false", "-template-debug=false", "-log-debug=false")
	} else {
		if p.profile {
			args = append(args, "-debug=false", "-template-debug=false", "-log-debug")
		} else {
			args = append(args, "-debug", "-template-debug", "-log-debug")
		}
	}
	if p.noCache {
		args = append(args, "-cache=dummy://")
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = io.MultiWriter(os.Stdout, &p.out)
	cmd.Stderr = io.MultiWriter(os.Stderr, &p.out)
	cmd.Dir = p.dir
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "GONDOLA_DEV_SERVER=1")
	cmd.Env = append(cmd.Env, "GONDOLA_FORCE_TTY=1")
	if p.profile {
		cmd.Env = append(cmd.Env, "GONDOLA_NO_CACHE_LAYER=1")
	}
	return cmd
}

func (p *Project) Start() error {
	p.Lock()
	defer p.Unlock()
	return p.startLocked()
}

func (p *Project) startLocked() error {
	p.port = randomFreePort()
	cmd := p.ProjectCmd()
	log.Infof("Starting %s (%s)", p.Name(), cmdString(cmd))
	p.cmd = cmd
	p.out.Reset()
	p.runError = nil
	p.exitCode = 0
	err := cmd.Start()
	go func() {
		werr := cmd.Wait()
		if cmd == p.cmd {
			// Othewise the process was intentionally killed
			if s := cmd.ProcessState; s != nil {
				exitCode := exitStatus(s)
				p.Lock()
				defer p.Unlock()
				p.runError = werr
				p.exitCode = exitCode
				log.Warningf("%s exited with code %d", p.Name(), exitCode)
			}
		}
	}()
	time.AfterFunc(100*time.Millisecond, p.projectStarted)
	return err
}

func (p *Project) projectStarted() {
	p.Lock()
	defer p.Unlock()
	u, err := url.Parse(fmt.Sprintf("http://localhost:%d", p.port))
	if err != nil {
		panic(err)
	}
	p.proxyChecked = false
	p.proxy = httputil.NewSingleHostReverseProxy(u)
	p.started = time.Now().UTC()
}

func (p *Project) Stop() error {
	p.Lock()
	defer p.Unlock()
	p.proxy = nil
	p.started = time.Time{}
	var err error
	cmd := p.cmd
	if cmd != nil {
		proc := cmd.Process
		if proc != nil {
			err = proc.Kill()
		}
		cmd.Wait()
		p.cmd = nil
	}
	if err != nil && strings.Contains(err.Error(), "already finished") {
		err = nil
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
	gcflags := []string{"-e"}
	args := []string{"build"}
	if p.goFlags != "" {
		fields, err := stringutil.SplitFields(p.goFlags, " ")
		if err != nil {
			panic(fmt.Errorf("error parsing -goflags: %v", err))
		}
		for ii := 0; ii < len(fields); ii++ {
			field := fields[ii]
			if field == "-gcflags" {
				subFields, err := stringutil.SplitFields(fields[ii+1], " ")
				if err != nil {
					panic(fmt.Errorf("error parsing -gcflags: %v", err))
				}
				gcflags = append(gcflags, subFields...)
				ii++
				continue
			}
			args = append(args, field)
		}
	}
	args = append(args, "-gcflags")
	args = append(args, strings.Join(gcflags, " "))
	args = append(args, p.buildTags()...)
	lib := filepath.Join(p.dir, "lib")
	if st, err := os.Stat(lib); err == nil && st.IsDir() {
		// If there's a lib directory, add it to rpath
		args = append(args, []string{"-ldflags", "-r lib"}...)
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = p.dir
	return cmd
}

// Build builds the project. If the project was already building, the build
// is restarted.
func (p *Project) Build() {
	cmd := p.CompilerCmd()
	var restarted bool
	p.Lock()
	if p.buildCmd != nil {
		proc := p.buildCmd.Process
		if proc != nil {
			proc.Signal(os.Interrupt)
			restarted = true
		}
	}
	p.buildCmd = cmd
	p.StopMonitoring()
	p.Unlock()
	if err := p.Stop(); err != nil {
		log.Panic(err)
	}
	p.errors = nil
	if !restarted {
		log.Infof("Building %s (%s)", p.Name(), cmdString(cmd))
	}
	var buf bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(&buf, os.Stderr)
	err := p.buildCmd.Run()
	p.Lock()
	defer p.Unlock()
	if p.buildCmd != cmd {
		// Canceled by another build
		return
	}
	p.buildCmd = nil
	p.built = time.Now().UTC()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			log.Panic(err)
		}
		if es := exitStatus(exitErr.ProcessState); es != 1 && es != 2 {
			// gc returns 1 when it can't find a package, 2 when there are compilation errors
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
			var be *BuildError
			switch {
			case strings.HasPrefix(eline, "package "):
				// package level error, like cyclic or non-allowed
				// (e.g. internal) imports. We need to create an error
				// now, since this line will usually be followed by
				// lines starting with \t
				pkg = strings.TrimSpace(eline[len("package"):])
				be = &BuildError{
					Error: strings.TrimSpace(eline),
				}
			case strings.HasPrefix(eline, "#"):
				// Package name before file level errors
				pkg = strings.TrimSpace(eline[1:])
			case strings.HasPrefix(eline, "\t"):
				// Info related to the previous error. Let it
				// crash if we don't have a previous error, just
				// in case there are any circumstances where a line
				// starting with \t means something else in the future.
				// This way the problem will be easier to catch.
				be := p.errors[len(p.errors)-1]
				be.Error += fmt.Sprintf(" (%s)", strings.TrimSpace(eline))
			default:
				parts := strings.SplitN(eline, ":", 3)
				if len(parts) == 3 {
					// file level error => filename:line:error
					filename := filepath.Clean(filepath.Join(p.dir, parts[0]))
					line, err := strconv.Atoi(parts[1])
					if err != nil {
						// Not a line number, show error message
						be = &BuildError{
							Error: strings.TrimSpace(eline),
						}
						break
					}
					be = &BuildError{
						Filename: filename,
						Line:     line,
						Error:    strings.TrimSpace(parts[2]),
					}
				} else {
					// Unknown error, just show the error message
					be = &BuildError{
						Error: strings.TrimSpace(eline),
					}
				}
			}
			if be != nil {
				be.Package = pkg
				p.errors = append(p.errors, be)
			}
		}
	}
	if c := len(p.errors); c == 0 {
		// TODO: Report error when starting project via web
		if err := p.startLocked(); err != nil {
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
		args = append(args, p.buildTags()...)
		p.GoCmd(args...).Run()
	}()
}

func (p *Project) Handler(ctx *app.Context) {
	if len(p.errors) > 0 {
		data := map[string]interface{}{
			"Project": p,
			"Errors":  p.errors,
			"Count":   len(p.errors),
		}
		ctx.MustExecute("errors.html", data)
		return
	}
	if p.runError != nil {
		// Exited at start
		s := p.out.String()
		var errorMessage string
		if m := panicRe.FindStringSubmatch(s); len(m) > 1 {
			errorMessage = m[1]
		}
		data := map[string]interface{}{
			"Project":  p,
			"Error":    errorMessage,
			"ExitCode": p.exitCode,
			"Output":   uncolor(s),
		}
		ctx.MustExecute("exited.html", data)
		return
	}
	if p.proxy == nil {
		// Building
		if ctx.R.Method != "POST" || ctx.R.Method != "PUT" {
			data := map[string]interface{}{
				"Project": p,
				"Name":    p.Name(),
			}
			ctx.MustExecute("building.html", data)
			return
		}
		// Wait until the app starts
		for {
			time.Sleep(10 * time.Millisecond)
			if p.proxy != nil {
				break
			}
		}
	}
	for !p.proxyChecked {
		// Check if we can connect to the app, to make
		// sure it has really started.
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", p.port))
		if err == nil {
			conn.Close()
			p.proxyChecked = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Proxy
	p.proxy.ServeHTTP(ctx, ctx.R)
}

func (p *Project) StatusHandler(ctx *app.Context) {
	built := formatTime(p.built)
	started := formatTime(p.started)
	if p.proxy == nil {
		// Building - this waits until the app is restarted for reloading
		// or compilation fails with an error
		if p.built.IsZero() {
			built = "1"
		}
		started = "1"
	}
	ctx.WriteJSON(map[string]interface{}{
		"built":   built,
		"started": started,
	})
}

func (p *Project) waitForBuild() {
	for {
		p.Lock()
		if p.buildCmd == nil {
			p.Unlock()
			break
		}
		p.Unlock()
	}
}

func (p *Project) isRunning() bool {
	return len(p.errors) == 0 && p.runError == nil
}

func autoConfigNames() []string {
	return []string{
		devConfigName,
		filepath.Base(config.DefaultFilename),
	}
}

func findConfig(dir string, name string) string {
	if name == "" {
		for _, v := range autoConfigNames() {
			if c := findConfig(dir, v); c != "" {
				return c
			}
		}
		return ""
	}
	configPath := filepath.Join(dir, name)
	for _, v := range []string{configPath, name} {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	return ""
}

type devOptions struct {
	Dir       string `help:"Project directory"`
	Port      int    `help:"Port to listen on"`
	Config    string `help:"Configuration file. If empty, dev.conf and app.conf are tried in that order"`
	Tags      string `help:"Build tags to pass to the Go compiler"`
	NoDebug   bool   `name:"no-debug" help:"Disable AppDebug, TemplateDebug and LogDebug - see gnd.la/config for details"`
	NoCache   bool   `name:"no-cache" help:"Disables the cache when running the project"`
	Profile   bool   `help:"Compiles and runs the project with profiling enabled"`
	NoBrowser bool   `name:"no-browser" help:"Don't open the default browser when starting the development server"`
	Verbose   bool   `name:"v" help:"Enable verbose output"`
	GoFlags   string `name:"goflags" help:"Extra flags to pass to the go command when building the app"`
}

func devCommand(args *command.Args, opts *devOptions) error {
	if !opts.Verbose {
		log.SetLevel(log.LInfo)
	}
	dir := opts.Dir
	if dir == "" {
		dir = "."
	}
	path, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	configPath := findConfig(dir, opts.Config)
	if configPath == "" {
		name := opts.Config
		if name == "" {
			name = fmt.Sprintf("(tried %s)", strings.Join(autoConfigNames(), ", "))
		}
		log.Panicf("can't find configuration file %s in %s", name, dir)
	}
	log.Infof("Using config file %s", configPath)
	p := NewProject(path, configPath)
	p.port = opts.Port
	p.tags = opts.Tags
	p.goFlags = opts.GoFlags
	p.noDebug = opts.NoDebug
	p.noCache = opts.NoCache
	p.profile = opts.Profile
	go p.Build()
	eof := "C"
	if runtime.GOOS == "windows" {
		eof = "Z"
	}
	log.Infof("Starting Gondola development server on port %d (press Control+%s to exit)", p.port, eof)
	if !opts.NoBrowser {
		time.AfterFunc(time.Second, func() {
			host := "localhost"
			if sshConn := os.Getenv("SSH_CONNECTION"); sshConn != "" {
				parts := strings.Split(sshConn, " ")
				// e.g. SSH_CONNECTION="10.211.55.2 56989 10.211.55.8 22"
				if len(parts) == 4 {
					if net.ParseIP(parts[2]) != nil {
						host = parts[2]
					}
				}
			}
			url := fmt.Sprintf("http://%s:%d", host, p.port)
			if err := browser.Open(url); err != nil {
				log.Errorf("error opening browser: open %s manually (error was %s)", url, err)
			}
		})
	}
	p.Listen()
	return nil
}
