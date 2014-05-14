package assets

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

func command(path string, args []string, w io.Writer, r io.Reader, opts Options) error {
	cmd := exec.Command(path, args...)
	cmd.Stdin = r
	cmd.Stdout = w
	var buf bytes.Buffer
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %s: %s", path, buf.String())
	}
	return nil
}
