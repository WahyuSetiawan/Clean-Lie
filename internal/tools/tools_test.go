package tools

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPreview(t *testing.T) {
	got := Preview([]string{"npm", "cache", "clean", "--force"})
	if got != "npm cache clean --force" {
		t.Fatalf("preview = %q", got)
	}
	got = Preview([]string{"sh", "-c", "rm -rf /tmp/x"})
	if !strings.Contains(got, "'rm -rf /tmp/x'") {
		t.Fatalf("preview tidak mengutip argumen: %q", got)
	}
}

func TestRun(t *testing.T) {
	if !Available("sh") {
		t.Skip("sh tidak ada")
	}
	res := Run(context.Background(), []string{"sh", "-c", "echo halo"}, 2*time.Second)
	if res.Err != nil || res.ExitCode != 0 || strings.TrimSpace(res.Stdout) != "halo" {
		t.Fatalf("run: %+v", res)
	}
}

func TestRunMissing(t *testing.T) {
	res := Run(context.Background(), []string{"cleanlie-tool-yang-tidak-ada"}, time.Second)
	if !errors.Is(res.Err, exec.ErrNotFound) {
		t.Fatalf("err = %v, mau exec.ErrNotFound", res.Err)
	}
}

func TestRunTimeout(t *testing.T) {
	res := Run(context.Background(), []string{"sh", "-c", "sleep 5"}, 50*time.Millisecond)
	if res.Err == nil {
		t.Fatal("mau timeout")
	}
	if res.Duration > 2*time.Second {
		t.Fatalf("timeout tidak bekerja: %s", res.Duration)
	}
}

func TestRunExitCode(t *testing.T) {
	res := Run(context.Background(), []string{"sh", "-c", "exit 3"}, time.Second)
	if res.Err == nil || res.ExitCode != 3 {
		t.Fatalf("exit code = %d err = %v", res.ExitCode, res.Err)
	}
}
