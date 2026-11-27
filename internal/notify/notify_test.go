package notify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSendUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if Available() {
		t.Fatal("notify-send tidak boleh tersedia")
	}
	if err := Send(context.Background(), "x", "y"); err != ErrUnavailable {
		t.Fatalf("err = %v", err)
	}
}

func TestSendArguments(t *testing.T) {
	bin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "args.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"$NOTIFY_LOG\"\n"
	if err := os.WriteFile(filepath.Join(bin, "notify-send"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NOTIFY_LOG", logPath)

	if !Available() {
		t.Fatal("notify-send palsu harus terdeteksi")
	}
	if err := Send(context.Background(), "Cleanlie", "2 item · 1 MiB"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{"--app-name=Cleanlie", "Cleanlie", "2 item"} {
		if !strings.Contains(got, want) {
			t.Fatalf("argumen %q tidak ada di %q", want, got)
		}
	}
}
