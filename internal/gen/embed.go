// +build IGNORE

// This program embeds generates prints a new go source file
// with the contents of the given file arguments as strings.
package main

import (
	"bytes"
	"fmt"
	"gnd.la/internal/gen/genutil"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var buf bytes.Buffer
	for ii := 1; ii < len(os.Args); ii++ {
		filename := os.Args[ii] + ".go"
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(fmt.Errorf("error reading %s: %s", filename, err))
		}
		lines := strings.Split(string(data), "\n")
		var file bytes.Buffer
		for _, v := range lines {
			if v == "" || strings.HasPrefix(v, "// +build") {
				continue
			}
			if strings.HasPrefix(v, "package") {
				if buf.Len() == 0 {
					buf.WriteString(v)
					buf.WriteByte('\n')
				}
				continue
			}
			file.WriteString(v)
			file.WriteByte('\n')
		}
		name := strings.Replace(filepath.Base(filename), ".", "_", -1)
		buf.WriteString(fmt.Sprintf("const %s = `%s`\n", name, file.String()))
	}
	if err := genutil.WriteAutogen("-", buf.Bytes()); err != nil {
		panic(err)
	}
}
