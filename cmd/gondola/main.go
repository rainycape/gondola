package main

import (
	"math/rand"
	"path/filepath"
	"time"
	"math/rand"

	"gnd.la/log"

	"github.com/rainycape/command"
)

var (
	commands = []*command.Cmd{
		{
			Name: "dev",
			Help: "Start the Gondola development server",
			Func: devCommand,
			Options: &devOptions{
				Dir:  ".",
				Port: 8888,
			},
		},
		{
			Name:    "new",
			Help:    "Create a new Gondola project",
			Usage:   "<dir>",
			Func:    newCommand,
			Options: &newOptions{Template: "hello"},
		},
		{
			Name:    "profile",
			Help:    "Show profiling information for a remote server running a Gondola app",
			Usage:   "<url>",
			Func:    profileCommand,
			Options: &profileOptions{Method: "GET"},
		},
		{
			Name:    "bake",
			Help:    "Converts all assets in <dir> into Go code and generates a VFS named with <name>",
			Usage:   "-dir=<dir> -name=<name> ... additional flags",
			Options: &bakeOptions{},
			Func:    bakeCommand,
		},
		{
			Name:    "random-string",
			Help:    "Generates a random string suitable for use as the app secret",
			Func:    randomStringCommand,
			Options: &randomStringOptions{Length: defaultRandomLength},
		},
		{
			Name:  "rm-gen",
			Help:  "Remove Gondola generated files (identified by *.gen.*)",
			Usage: "[dir]",
			Func:  rmGenCommand,
		},
		{
			Name:    "make-messages",
			Help:    "Generate strings files from the current package (including its non-package subdirectories, like templates)",
			Func:    makeMessagesCommand,
			Options: &makeMessagesOptions{Out: filepath.Join("_messages", "messages.pot")},
		},
		{
			Name:    "compile-messages",
			Help:    "Compiles all po files from the current directory and its subdirectories",
			Func:    compileMessagesCommand,
			Options: &compileMessagesOptions{Out: "messages.go", Messages: "_messages"},
		},
		{
			Name:    "gae-dev",
			Help:    "Start the Gondola App Engine development server",
			Func:    gaeDevCommand,
			Options: &gaeDevOptions{Host: "localhost", Port: 8080, AdminPort: 8000},
		},
		{
			Name:    "gae-test",
			Help:    "Start serving your app on localhost and run gnd.la/app/tester tests against it",
			Func:    gaeTestCommand,
			Options: &gaeTestOptions{},
		},
		{
			Name:    "gae-deploy",
			Help:    "Deploy your application to App Engine",
			Func:    gaeDeployCommand,
			Options: &gaeDeployOptions{},
		},
	}
)

type commonOptions struct {
	Quiet bool `name:"q" help:"Disable verbose output"`
}

func main() {
	opts := &command.Options{
		Options: &commonOptions{},
		Func: func(_ *command.Cmd, opts *command.Options) error {
			copts := opts.Options.(*commonOptions)
			if copts.Quiet {
				log.SetLevel(log.LError)
			} else {
				log.SetLevel(log.LDebug)
			}
			return nil
		},
	}
	command.Exit(command.RunOpts(nil, opts, commands))
}

func init() {
	rand.Seed(time.Now().Unix())
}
