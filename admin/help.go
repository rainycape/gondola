package admin

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

func init() {
	usage := flag.Usage
	flag.Usage = func() {
		usage()
		var cmds []string
		for k, _ := range commands {
			cmds = append(cmds, k)
		}
		sort.Strings(cmds)
		fmt.Fprintf(os.Stderr, "\nAvailable administrative commands:\n")
		for _, v := range cmds {
			if v == "help" || commandIsHidden(v) {
				continue
			}
			fmt.Fprintf(os.Stderr, "  %s\n", v)
		}
		fmt.Fprintf(os.Stderr, "\nType %s help for details.\n", os.Args[0])
	}
}
