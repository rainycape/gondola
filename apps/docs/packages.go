package docs

import (
	"gnd.la/log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	// Groups includes the packages to list in the package index.
	// See the Group struct for further information.
	Groups []*Group
	ticker *time.Ticker
)

// Group represents a group of packages to be displayed under the same
// title. Note that all subpackages of any included package will also
// be listed. Packages must be referred by their import path (e.g.
// example.com/pkg).
type Group struct {
	Title    string
	Packages []string
}

// StartUpdatingPackages starts regularly updating the packages listed
// in Groups at the given interval. Note that for updating packages, the
// system should have installed the client for the SCM systems used by
// them (e.g. git, hg, etc...). A working Go installation on the host
// is also required, since go get will be used to download them.
func StartUpdatingPackages(interval time.Duration) {
	StopUpdatingPackages()
	go updatePackages()
	ticker = time.NewTicker(interval)
	go func() {
		for _ = range ticker.C {
			updatePackages()
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

func updatePackages() {
	for _, gr := range Groups {
		for _, pkg := range gr.Packages {
			if err := updatePackage(pkg); err != nil {
				log.Errorf("error updating %s: %s", pkg, err)
			}
		}
	}
}

func updatePackage(pkg string) error {
	if strings.HasSuffix(pkg, "/") {
		pkg += "..."
	}
	cmd := exec.Command("go", "get", "-u", "-v", pkg)
	log.Debugf("Updating package %s", pkg)
	if log.Level() == log.LDebug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}
