package cli

import (
	"bytes"
	"testing"
)

func TestRunNoop(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run(stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
