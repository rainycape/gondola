package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"code.google.com/p/go.exp/fsnotify"

	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/log"
)

func startServe(buildArgs []string) (*exec.Cmd, error) {
	args := append([]string{"serve"}, buildArgs...)
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
		w.Watch(v)
	}
	for {
		select {
		case e := <-w.Error:
			return e
		case ev := <-w.Event:
			fmt.Println(ev)
			name := filepath.Base(ev.Name)
			if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") || ev.IsAttrib() || ev.IsDelete() || ev.IsRename() {
				fmt.Println("IGNORED", ev)
				continue
			}
			makeAppAssets(buildArgs)
		}
	}
	return nil
}

func GaeDev(ctx *app.Context) {
	log.Debugf("starting App Engine development server - press Control+C to stop")
	var tags string
	ctx.ParseParamValue("tags", &tags)
	var buildArgs []string
	if tags != "" {
		buildArgs = append(buildArgs, "-tags", tags)
	}
	resources, err := makeAppAssets(buildArgs)
	if err != nil {
		panic(err)
	}
	go watchAppResources(buildArgs, resources)
	serveCmd, err := startServe(buildArgs)
	if err != nil {
		panic(err)
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
		panic(err)
	}
}

func GaeDeploy(ctx *app.Context) {
	cmd := exec.Command("gondola", "rm-gen")
	if err := runCmd(cmd); err != nil {
		panic(err)
	}
	makeAppAssets(nil)
	deployCmd := exec.Command("goapp", "deploy")
	if err := runCmd(deployCmd); err != nil {
		panic(err)
	}
}

func GaeTest(ctx *app.Context) {
	log.Debugf("starting App Engine tests")
	var tags string
	ctx.ParseParamValue("tags", &tags)
	var buildArgs []string
	if tags != "" {
		buildArgs = append(buildArgs, "-tags", tags)
	}
	serveCmd, err := startServe(buildArgs)
	if err != nil {
		panic(err)
	}
	serveCh := make(chan error, 1)
	go func() {
		serveCh <- serveCmd.Wait()
	}()
	args := append([]string{"test"}, buildArgs...)
	var verbose bool
	ctx.ParseParamValue("v", &verbose)
	if verbose {
		args = append(args, "-v")
	}
	args = append(args, "-L")
	testCmd := exec.Command("goapp", args...)
	runCmd(testCmd)
	serveCmd.Process.Signal(os.Interrupt)
	<-serveCh
}

func init() {
	admin.Register(GaeDev, &admin.Options{
		Help: "Start App Engine development server",
	})
	admin.Register(GaeDeploy, &admin.Options{
		Help: "Deploy your application to App Engine development",
	})
	admin.Register(GaeTest, &admin.Options{
		Help: "Start serving your app on localhost and run gnd.la/test/tester tests against it",
		Flags: admin.Flags(
			admin.BoolFlag("v", false, "Enable verbose tests"),
		),
	})
}
