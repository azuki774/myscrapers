package cli

import "io"

func Run(stdout, stderr io.Writer) int {
	_ = stdout
	_ = stderr
	return 0
}
