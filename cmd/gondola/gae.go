package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-fsnotify/fsnotify"

	"gnd.la/log"
)

func startServe(buildArgs []string, opts *gaeDevOptions) (*exec.Cmd, error) {
	args := append([]string{"serve"}, buildArgs...)
	if opts != nil {
		if opts.Host != "" {
			args = append(args, "-host", opts.Host)
		}
		if opts.Port > 0 {
			args = append(args, "-port", strconv.Itoa(opts.Port))
		}
		if opts.AdminPort > 0 {
			args = append(args, "-admin_port", strconv.Itoa(opts.AdminPort))
		}
	}
	cmd := exec.Command("goapp", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func runCmd(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runBuild(buildArgs []string) error {
	args := append([]string{"build"}, buildArgs...)
	return runCmd(exec.Command("goapp", args...))
}

func appPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	name := filepath.Base(wd)
	return filepath.Join(wd, name), nil
}

func makeAppAssets(buildArgs []string) ([]string, error) {
	log.Debugf("compiling app assets")
	if err := runBuild(buildArgs); err != nil {
		return nil, err
	}
	p, err := appPath()
	if err != nil {
		return nil, err
	}
	defer os.Remove(p)
	err = runCmd(exec.Command(p, "-log-debug=false", "make-assets"))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	cmd := exec.Command(p, "_print-resources")
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		return nil, err
	}
	values := make([]string, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values, nil
}

func watchAppResources(buildArgs []string, resources []string) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	for _, v := range resources {
		w.Add(v)
	}
	for {
		select {
		case e := <-w.Errors:
			return e
		case ev := <-w.Events:
			name := filepath.Base(ev.Name)
			op := ev.Op
			if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") || op == fsnotify.Remove || op == fsnotify.Rename {
				continue
			}
			makeAppAssets(buildArgs)
		}
	}
	return nil
}

type gaeDevOptions struct {
	Host      string `help:"Host name to which application modules should bind"`
	Port      int    `help:"Lowest port to which application modules should bind"`
	AdminPort int    `help:"Port to which the admin server should bind"`
}

func gaeDevCommand(opts *gaeDevOptions) error {
	log.Debugf("starting App Engine development server - press Control+C to stop")
	var buildArgs []string
	resources, err := makeAppAssets(buildArgs)
	if err != nil {
		return err
	}
	go watchAppResources(buildArgs, resources)
	serveCmd, err := startServe(buildArgs, opts)
	if err != nil {
		return err
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveCmd.Wait()
	}()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	select {
	case <-ch:
		log.Debugf("exiting")
		<-errCh
		log.Debugf("exited")
	case err := <-errCh:
		return err
	}
	return nil
}

type gaeTestOptions struct {
	Verbose bool `name:"v" help:"Enable verbose tests"`
}

func gaeTestCommand(opts *gaeTestOptions) error {
	log.Debugf("starting App Engine tests")
	var buildArgs []string
	serveCmd, err := startServe(buildArgs, nil)
	if err != nil {
		return err
	}
	serveCh := make(chan error, 1)
	go func() {
		serveCh <- serveCmd.Wait()
	}()
	args := append([]string{"test"}, buildArgs...)
	if opts.Verbose {
		args = append(args, "-v")
	}
	args = append(args, "-L")
	testCmd := exec.Command("goapp", args...)
	runCmd(testCmd)
	serveCmd.Process.Signal(os.Interrupt)
	<-serveCh
	return nil
}

type gaeDeployOptions struct {
	OAuth bool `name:"oauth" help:"Use oAuth 2 authentication rather than password"`
}

func gaeDeployCommand(opts *gaeDeployOptions) error {
	cmd := exec.Command("gondola", "rm-gen")
	if err := runCmd(cmd); err != nil {
		return err
	}
	makeAppAssets(nil)
	// Remove the app binary, otherwise it gets uploaded to GAE
	exec.Command("go", "clean").Run()
	args := []string{"deploy"}
	if opts.OAuth {
		args = append(args, "-oauth")
	}
	deployCmd := exec.Command("goapp", args...)
	if err := runCmd(deployCmd); err != nil {
		return err
	}
	return nil
}
