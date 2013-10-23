package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func autogenString() string {
	return fmt.Sprintf("\n// AUTOMATICALLY GENERATED WITH %s -- DO NOT EDIT!\n", strings.Join(os.Args, " "))
}

func isAutogen(filename string) bool {
	b, _ := ioutil.ReadFile(filename)
	if b != nil {
		return bytes.Index(b, []byte("// AUTOMATICALLY GENERATED WITH")) >= 0
	}
	return false
}
