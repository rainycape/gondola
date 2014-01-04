package assets

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

func command(path string, args []string, w io.Writer, r io.Reader, opts Options) error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "-")
	cmd := exec.Command(path, cmdArgs...)
	cmd.Stdin = r
	cmd.Stdout = w
	var buf bytes.Buffer
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %s: %s", path, buf.String())
	}
	return nil
}
