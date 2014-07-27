package main

import (
	"path/filepath"

	"gnd.la/log"

	"gopkgs.com/command.v1"
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
			Name:     "build",
			Help:     "Build packages",
			Usage:    "[package-1] [package-2] ... [package-n]",
			LongHelp: buildHelp,
			Func:     buildCommand,
			Options:  &buildOptions{Go: "go"},
		},
		{
			Name: "clean",
			Help: "Cleans any Gondola packages which use conditional compilation - DO THIS BEFORE BUILDING A BINARY FOR DEPLOYMENT - see golang.org/issue/3172",
			Func: cleanCommand,
		},
		{
			Name:    "profile",
			Help:    "Show profiling information for a remote server running a Gondola app",
			Usage:   "<url>",
			Func:    profileCommand,
			Options: &profileOptions{Method: "GET"},
		},
		{
			Name:    "gen-app",
			Help:    "Generate boilerplate code for a Gondola app from the appfile.yaml file",
			Func:    genAppCommand,
			Options: &genAppOptions{},
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
			Options: &compileMessagesOptions{Out: "messages.go"},
		},
		{
			Name:    "gen",
			Help:    "Perform code generation in the current directory according the rules in the config file",
			Options: &genOptions{Genfile: "genfile.yaml"},
		},
		{
			Name: "gae-dev",
			Help: "Start the Gondola App Engine development server",
			Func: gaeDevCommand,
		},
		{
			Name:    "gae-test",
			Help:    "Start serving your app on localhost and run gnd.la/app/tester tests against it",
			Func:    gaeTestCommand,
			Options: &gaeTestOptions{},
		},
		{
			Name: "gae-deploy",
			Help: "Deploy your application to App Engine",
			Func: gaeDeployCommand,
		},
	}
)

type commonOptions struct {
	Quiet bool `name:"q" help:"Disable verbose output"`
}

func main() {
	opts := &command.Options{
		Options: &commonOptions{},
		Func: func(opts *commonOptions) {
			if opts.Quiet {
				log.SetLevel(log.LError)
			} else {
				log.SetLevel(log.LDebug)
			}
		},
	}
	command.Exit(command.RunOpts(nil, opts, commands))
}
