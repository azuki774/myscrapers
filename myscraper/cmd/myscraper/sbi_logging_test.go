package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func TestSBICommandRoutingHelper(t *testing.T) {
	if os.Getenv("MYSCRAPER_SBI_ROUTING_HELPER") != "1" {
		return
	}
	os.Args = []string{"myscraper", "sbi", "--passkey", "/tmp/myscrapers-missing-passkey-routing.json"}
	main()
}

func TestSBICommandRoutesLogsToStderr(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestSBICommandRoutingHelper")
	cmd.Env = append(os.Environ(), "MYSCRAPER_SBI_ROUTING_HELPER=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("child exit = %v, want exit code 1", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout contains logs or data: %q", stdout.String())
	}
	scanner := bufio.NewScanner(&stderr)
	count := 0
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("stderr line is not JSON: %q (%v)", scanner.Text(), err)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if count == 0 {
		t.Fatal("stderr contains no structured log records")
	}
}
