//go:build linux

package privileged

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

func TestExecutePkexec(t *testing.T) {
	var got []string
	res := Execute(context.Background(), []string{"journalctl", "--vacuum-size=500M"}, Options{
		HasAgent: true,
		LookPath: fakeLook,
		Runner: func(_ context.Context, cmd []string, _ time.Duration) tools.Result {
			got = cmd
			return tools.Result{ExitCode: 0, Stdout: "ok"}
		},
	})
	if res.ExitCode != 0 || res.Used != "pkexec" {
		t.Fatalf("res = %+v", res)
	}
	want := []string{"/run/current-system/sw/bin/pkexec", "/run/current-system/sw/bin/journalctl", "--vacuum-size=500M"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("cmd = %v", got)
	}
}

func TestExecutePkexecDenied(t *testing.T) {
	res := Execute(context.Background(), []string{"nix-env", "--list-generations"}, Options{
		HasAgent: true,
		LookPath: fakeLook,
		Runner: func(context.Context, []string, time.Duration) tools.Result {
			return tools.Result{ExitCode: 126, Stderr: "not authorized"}
		},
	})
	if !res.NeedsSudo || res.Used != "sudo" || res.SudoCommand == "" {
		t.Fatalf("res = %+v", res)
	}
	if res.Stderr != "not authorized" {
		t.Fatalf("stderr hilang: %+v", res)
	}
}
