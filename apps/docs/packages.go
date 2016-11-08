package docs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gnd.la/apps/docs/doc"
	"gnd.la/log"
)

var (
	// Groups includes the packages to list in the package index.
	// See the Group struct for further information.
	ticker *time.Ticker
)

// StartUpdatingPackages starts regularly updating the packages listed
// in Groups at the given interval. Note that for updating packages, the
// system should have installed the client for the SCM systems used by
// them (e.g. git, hg, etc...). A working Go installation on the host
// is also required, since go get will be used to download them.
func StartUpdatingPackages(ctx *doc.Environment, interval time.Duration) {
	StopUpdatingPackages()
	go updatePackages(ctx)
	ticker = time.NewTicker(interval)
	go func() {
		for _ = range ticker.C {
			updatePackages(ctx)
		}
	}()
}

// StopUpdatingPackages stops updating packages. Note that an in-flight update
// won't be stopped, but no more updates will be scheduled.
func StopUpdatingPackages() {
	if ticker != nil {
		ticker.Stop()
		ticker = nil
	}
}

func updatePackages(e *doc.Environment) {
	for _, gr := range getDocsApp(e).Groups {
		for _, pkg := range gr.Packages {
			if err := updatePackage(e, pkg); err != nil {
				log.Errorf("error updating %s: %s", pkg, err)
			}
		}
	}
}

func updatePackage(e *doc.Environment, pkg string) error {
	if strings.HasSuffix(pkg, "/") {
		pkg += "..."
	}
	goBin := "go"
	env := make(map[string]string)
	for _, v := range os.Environ() {
		if eq := strings.Index(v, "="); eq >= 0 {
			env[v[:eq]] = v[eq+1:]
		}
	}
	if goRoot := e.Context.GOROOT; goRoot != "" {
		goBin = filepath.Join(goRoot, "bin", "go")
		env["GOROOT"] = goRoot
	}
	if goPath := e.Context.GOPATH; goPath != "" {
		env["GOPATH"] = goPath
	}
	cmd := exec.Command(goBin, "get", "-u", "-v", pkg)
	cmdEnv := make([]string, 0, len(env))
	for k, v := range env {
		cmdEnv = append(cmdEnv, k+"="+v)
	}
	cmd.Env = cmdEnv
	log.Debugf("Updating package %s", pkg)
	if log.Level() == log.LDebug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}
