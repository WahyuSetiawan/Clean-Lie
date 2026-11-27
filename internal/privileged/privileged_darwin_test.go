//go:build darwin

package privileged

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

func TestExecuteOsascript(t *testing.T) {
	var got []string
	res := Execute(context.Background(), []string{"nix-collect-garbage", "-d"}, Options{
		HasAgent: true,
		LookPath: fakeLook,
		Runner: func(_ context.Context, cmd []string, _ time.Duration) tools.Result {
			got = cmd
			return tools.Result{ExitCode: 0, Stdout: "ok"}
		},
	})
	if res.ExitCode != 0 || res.Used != "osascript" {
		t.Fatalf("res = %+v", res)
	}
	if len(got) != 3 || !strings.HasSuffix(got[0], "/osascript") || got[1] != "-e" {
		t.Fatalf("cmd = %v", got)
	}
	script := got[2]
	if !strings.Contains(script, `do shell script "`) || !strings.HasSuffix(script, `" with administrator privileges`) {
		t.Fatalf("script = %q", script)
	}
	if !strings.Contains(script, "nix-collect-garbage") {
		t.Fatalf("perintah tidak ada di script: %q", script)
	}
}

func TestExecuteOsascriptQuotesArgs(t *testing.T) {
	var got []string
	Execute(context.Background(), []string{"rm", "-rf", "/tmp/a b"}, Options{
		HasAgent: true,
		LookPath: fakeLook,
		Runner: func(_ context.Context, cmd []string, _ time.Duration) tools.Result {
			got = cmd
			return tools.Result{}
		},
	})
	if len(got) != 3 {
		t.Fatalf("cmd = %v", got)
	}
	if !strings.Contains(got[2], `'/tmp/a b'`) {
		t.Fatalf("argumen tidak di-quote: %q", got[2])
	}
}

func TestExecuteOsascriptCanceled(t *testing.T) {
	res := Execute(context.Background(), []string{"nix-collect-garbage", "-d"}, Options{
		HasAgent: true,
		LookPath: fakeLook,
		Runner: func(context.Context, []string, time.Duration) tools.Result {
			return tools.Result{ExitCode: 1, Stderr: "execution error: User canceled. (-128)"}
		},
	})
	if !res.NeedsSudo || res.Used != "sudo" || !strings.HasPrefix(res.SudoCommand, "sudo ") {
		t.Fatalf("res = %+v", res)
	}
}

func TestDetectAgentDarwin(t *testing.T) {
	if !DetectAgent() {
		t.Fatal("macOS harus melaporkan prompt admin tersedia")
	}
}
