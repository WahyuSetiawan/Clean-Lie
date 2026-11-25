package privileged

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

func fakeLook(name string) (string, error) {
	return "/run/current-system/sw/bin/" + name, nil
}

func TestExecuteNoAgent(t *testing.T) {
	called := false
	res := Execute(context.Background(), []string{"nix-collect-garbage", "-d"}, Options{
		HasAgent: false,
		LookPath: fakeLook,
		Runner: func(context.Context, []string, time.Duration) tools.Result {
			called = true
			return tools.Result{}
		},
	})
	if called {
		t.Fatal("perintah tidak boleh jalan tanpa agent")
	}
	if !res.NeedsSudo || !strings.HasPrefix(res.SudoCommand, "sudo /run/current-system/sw/bin/nix-collect-garbage") {
		t.Fatalf("res = %+v", res)
	}
	if res.ExitCode != -1 {
		t.Fatalf("exit = %d", res.ExitCode)
	}
}

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

func TestHasAgentFrom(t *testing.T) {
	root := t.TempDir()
	write := func(pid, cmd string) {
		dir := filepath.Join(root, pid)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(strings.ReplaceAll(cmd, " ", "\x00")+"\x00"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("100", "/usr/bin/polkitd --no-debug")
	if hasAgentFrom(root) {
		t.Fatal("polkitd bukan agent")
	}
	write("101", "/usr/lib/polkit-gnome-authentication-agent-1")
	if !hasAgentFrom(root) {
		t.Fatal("agent polkit tidak terdeteksi")
	}
	write("102", "hyprpolkitagent")
	if !hasAgentFrom(root) {
		t.Fatal("hyprpolkitagent tidak terdeteksi")
	}
}

func TestPreview(t *testing.T) {
	got := Preview([]string{"nix-collect-garbage", "-d"}, false)
	if !strings.HasPrefix(got, "sudo ") || !strings.Contains(got, "nix-collect-garbage") {
		t.Fatalf("preview = %q", got)
	}
}
