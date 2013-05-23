package admin

import (
	"fmt"
	"gondola/log"
	"gondola/mux"
	"gondola/util"
	"flag"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

const tabWidth = 8

var (
	commands = map[string]*Command{}
)

type Command struct {
	Handler mux.Handler
	Name    string
	Help    string
}

func F(f mux.Handler) *Command {
	return &Command{Handler: f}
}

func N(f mux.Handler, name string) *Command {
	return &Command{Handler: f, Name: name}
}

func H(f mux.Handler, help string) *Command {
	return &Command{Handler: f, Help: help}
}

func NH(f mux.Handler, name string, help string) *Command {
	return &Command{Handler: f, Name: name, Help: help}
}

func Register(cmds ...*Command) {
	for _, c := range cmds {
		if c.Name == "" {
			name := runtime.FuncForPC(reflect.ValueOf(c.Handler).Pointer()).Name()
			p := strings.Split(name, ".")
			c.Name = p[len(p)-1]
			if c.Name == "" {
				log.Fatalf("Could not determine name for function %v. Please, use admin.N() or admin.NH() to provide a name.", c.Handler)
			}
		}
		cmdName := util.UnCamelCase(c.Name, "-")
		if _, ok := commands[cmdName]; ok {
			log.Fatalf("Duplicate command name %q", c.Name)
		}
		commands[cmdName] = c
	}
}

func Perform(m *mux.Mux) bool {
	args := flag.Args()
	if len(args) > 0 {
		cmd := strings.ToLower(args[0])
		for k, v := range commands {
			if cmd == k {
				ctx := m.NewContext(args)
				defer m.CloseContext(ctx)
				v.Handler(ctx)
				return true
			}
		}
	}
	return false
}

func help(ctx *mux.Context) {
	fmt.Fprintf(os.Stderr, "Administrative commands:\n")
	var cmds []string
	maxLen := 0
	for k, _ := range commands {
		if l := len(k); l > maxLen {
			maxLen = l
		}
		cmds = append(cmds, k)
	}
	sort.Strings(cmds)
	for _, v := range cmds {
		tabs := strings.Repeat("\t", (maxLen/tabWidth)-((len(v)+1)/tabWidth)+1)
		fmt.Fprintf(os.Stderr, "%s:%s%s\n", v, tabs, commands[v].Help)
	}
}

func init() {
	Register(H(help, "Show available commands with their respective help."))
}
