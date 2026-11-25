package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadIni(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.ini")
	content := "# komentar\nabi.type=x86_64\nhw.cpu.arch = x86_64\n\nbrokenline\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readIni(p)
	if got["abi.type"] != "x86_64" || got["hw.cpu.arch"] != "x86_64" {
		t.Fatalf("ini = %v", got)
	}
	if len(got) != 2 {
		t.Fatalf("ini = %v", got)
	}
}

func TestAVDNote(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	avd := filepath.Join(home, ".android", "avd", "voi_api35.avd")
	if err := os.MkdirAll(avd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(avd, "config.ini"), []byte("abi.type=x86_64\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".android", "avd", "voi_api35.ini"), []byte("target=android-35\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	note := avdNote(avd)
	for _, want := range []string{"voi_api35", "API 35", "x86_64"} {
		if !strings.Contains(note, want) {
			t.Errorf("note = %q, tidak memuat %q", note, want)
		}
	}
}

func TestAVDNoteFromSysdir(t *testing.T) {
	dir := t.TempDir()
	avd := filepath.Join(dir, "test_devices.avd")
	if err := os.MkdirAll(avd, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "image.sysdir.1=system-images/android-34/google_apis/arm64-v8a/\nhw.cpu.arch=arm64\n"
	if err := os.WriteFile(filepath.Join(avd, "config.ini"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	note := avdNote(avd)
	if !strings.Contains(note, "API 34") || !strings.Contains(note, "arm64") {
		t.Fatalf("note = %q", note)
	}
}

func TestAPIfromSysdir(t *testing.T) {
	if got := apiFromSysdir("system-images/android-35/google_apis/x86_64/"); got != "35" {
		t.Fatalf("= %q", got)
	}
	if got := apiFromSysdir("tanpa-api"); got != "" {
		t.Fatalf("= %q", got)
	}
}
