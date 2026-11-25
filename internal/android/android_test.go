package android

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

var testNow = time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)

func fixture(t *testing.T) (Options, string) {
	t.Helper()
	home := t.TempDir()
	avdHome := filepath.Join(home, ".android", "avd")
	sdk := filepath.Join(home, "Android", "Sdk")
	proc := filepath.Join(home, "proc")
	for _, d := range []string{
		filepath.Join(avdHome, "test.avd"),
		filepath.Join(sdk, "system-images", "android-36", "google_apis", "x86_64"),
		filepath.Join(sdk, "system-images", "android-35", "google_apis", "x86_64"),
		filepath.Join(sdk, "ndk", "27.0.12077973"),
		filepath.Join(sdk, "build-tools", "35.0.0"),
		filepath.Join(sdk, "platforms", "android-36"),
		filepath.Join(sdk, "emulator"),
		filepath.Join(sdk, "cmdline-tools", "latest", "bin"),
		proc,
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p string, b []byte) {
		t.Helper()
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := testNow.Add(-100 * 24 * time.Hour)
	write(filepath.Join(avdHome, "test.ini"), []byte("avd.ini.encoding=UTF-8\npath="+filepath.Join(avdHome, "test.avd")+"\ntarget=android-36\n"))
	write(filepath.Join(avdHome, "test.avd", "config.ini"), []byte("abi.type=x86_64\nimage.sysdir.1=system-images/android-36/google_apis/x86_64/\n"))
	write(filepath.Join(avdHome, "test.avd", "userdata-qemu.img.qcow2"), make([]byte, 32*1024))
	write(filepath.Join(avdHome, "test.avd", "emu-launch-params.txt"), []byte("x"))
	write(filepath.Join(sdk, "system-images", "android-36", "google_apis", "x86_64", "system.img"), make([]byte, 16*1024))
	write(filepath.Join(sdk, "system-images", "android-35", "google_apis", "x86_64", "system.img"), make([]byte, 16*1024))
	write(filepath.Join(sdk, "ndk", "27.0.12077973", "source.properties"), []byte("Pkg.Revision=27\n"))
	write(filepath.Join(sdk, "build-tools", "35.0.0", "aapt"), []byte("x"))
	write(filepath.Join(sdk, "platforms", "android-36", "android.jar"), []byte("x"))
	write(filepath.Join(sdk, "emulator", "emulator"), []byte("#!/bin/sh\n"))
	write(filepath.Join(sdk, "cmdline-tools", "latest", "bin", "sdkmanager"), []byte("#!/bin/sh\n"))
	write(filepath.Join(sdk, "cmdline-tools", "latest", "bin", "avdmanager"), []byte("#!/bin/sh\n"))
	if err := os.Chmod(filepath.Join(sdk, "emulator", "emulator"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(avdHome, "test.avd", "userdata-qemu.img.qcow2"),
		filepath.Join(avdHome, "test.avd", "emu-launch-params.txt"),
		filepath.Join(sdk, "system-images", "android-36"),
		filepath.Join(sdk, "system-images", "android-36", "google_apis", "x86_64"),
		filepath.Join(sdk, "system-images", "android-35"),
		filepath.Join(sdk, "system-images", "android-35", "google_apis", "x86_64"),
		filepath.Join(sdk, "ndk", "27.0.12077973"),
		filepath.Join(sdk, "build-tools", "35.0.0"),
		filepath.Join(sdk, "platforms", "android-36"),
		filepath.Join(sdk, "emulator"),
	} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
	return Options{Home: home, SDK: sdk, AVDHome: avdHome, ProcRoot: proc, Now: func() time.Time { return testNow }}, home
}

func TestReadStatus(t *testing.T) {
	opts, _ := fixture(t)
	st := ReadStatus(context.Background(), opts)
	if len(st.AVDs) != 1 {
		t.Fatalf("avd = %+v", st.AVDs)
	}
	a := st.AVDs[0]
	if a.Name != "test" || a.API != "36" || a.ABI != "x86_64" || a.SystemImage != "system-images;android-36;google_apis;x86_64" {
		t.Fatalf("avd isi = %+v", a)
	}
	if a.Bytes == 0 || !a.Suggested || a.AgeDays < 90 {
		t.Fatalf("avd ukuran/saran = %+v", a)
	}
	if !strings.Contains(a.LastUsed, "2026-08-17") {
		t.Fatalf("last used = %q", a.LastUsed)
	}
	if st.AVDBytes != a.Bytes || st.SDKBytes == 0 {
		t.Fatalf("total = %+v", st)
	}
	var used, unused *Package
	for i := range st.Packages {
		switch st.Packages[i].ID {
		case "system-images;android-36;google_apis;x86_64":
			used = &st.Packages[i]
		case "system-images;android-35;google_apis;x86_64":
			unused = &st.Packages[i]
		}
	}
	if used == nil || len(used.UsedBy) != 1 || used.UsedBy[0] != "test" || used.Suggested {
		t.Fatalf("paket dipakai = %+v", used)
	}
	if unused == nil || len(unused.UsedBy) != 0 || !unused.Suggested {
		t.Fatalf("paket tidak dipakai = %+v", unused)
	}
	for _, p := range st.Packages {
		if p.Kind == "ndk" && p.Suggested {
			t.Fatalf("NDK tidak boleh disarankan otomatis: %+v", p)
		}
	}
}

func TestDeleteAVDFallback(t *testing.T) {
	opts, _ := fixture(t)
	// avdmanager ada, tetapi gagal seolah AVD tidak dikenal; fallback manual.
	opts.Run = func(context.Context, []string, time.Duration, []string) tools.Result {
		return tools.Result{ExitCode: 1, Stderr: "Error: AVD 'test' does not exist."}
	}
	res, err := DeleteAVD(context.Background(), "test", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Deleted || res.Freed <= 0 || res.Used != "manual" {
		t.Fatalf("res = %+v", res)
	}
	if _, err := os.Stat(filepath.Join(opts.AVDHome, "test.avd")); !os.IsNotExist(err) {
		t.Fatalf("folder AVD masih ada: %v", err)
	}
	if _, err := os.Stat(filepath.Join(opts.AVDHome, "test.ini")); !os.IsNotExist(err) {
		t.Fatalf("ini AVD masih ada: %v", err)
	}
}

func TestDeleteAVDRefusesRunning(t *testing.T) {
	opts, _ := fixture(t)
	pid := filepath.Join(opts.ProcRoot, "123")
	if err := os.MkdirAll(pid, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pid, "cmdline"), []byte("emulator\x00-avd\x00test\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteAVD(context.Background(), "test", opts); err == nil {
		t.Fatal("AVD yang berjalan harus ditolak")
	}
}

func TestUninstallPackage(t *testing.T) {
	opts, _ := fixture(t)
	opts.Run = func(_ context.Context, cmd []string, _ time.Duration, env []string) tools.Result {
		if !strings.Contains(strings.Join(cmd, " "), "--uninstall system-images;android-35;google_apis;x86_64") {
			t.Fatalf("cmd = %v", cmd)
		}
		if len(env) == 0 || !strings.Contains(strings.Join(env, " "), "ANDROID_HOME=") {
			t.Fatalf("env = %v", env)
		}
		p := filepath.Join(opts.SDK, "system-images", "android-35", "google_apis", "x86_64")
		if err := os.RemoveAll(p); err != nil {
			t.Fatal(err)
		}
		return tools.Result{ExitCode: 0}
	}
	res, err := UninstallPackage(context.Background(), "system-images;android-35;google_apis;x86_64", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Deleted || res.Freed <= 0 {
		t.Fatalf("res = %+v", res)
	}

	if _, err := UninstallPackage(context.Background(), "system-images;android-36;google_apis;x86_64", opts); err == nil {
		t.Fatal("paket yang dipakai AVD harus ditolak")
	}
}

func TestWipeAVDCommand(t *testing.T) {
	opts, _ := fixture(t)
	var got []string
	opts.Run = func(_ context.Context, cmd []string, _ time.Duration, _ []string) tools.Result {
		got = cmd
		return tools.Result{ExitCode: 0}
	}
	res, err := WipeAVD(context.Background(), "test", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(got, " "), "-wipe-data") || !strings.Contains(strings.Join(got, " "), "-quit-after-boot") {
		t.Fatalf("cmd = %v", got)
	}
	if res.Action != "wipe" || res.Target != "test" {
		t.Fatalf("res = %+v", res)
	}
}

func TestDeleteAVDOfficial(t *testing.T) {
	opts, _ := fixture(t)
	var got []string
	opts.Run = func(_ context.Context, cmd []string, _ time.Duration, _ []string) tools.Result {
		got = cmd
		if err := os.RemoveAll(filepath.Join(opts.AVDHome, "test.avd")); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(opts.AVDHome, "test.ini")); err != nil {
			t.Fatal(err)
		}
		return tools.Result{ExitCode: 0}
	}
	res, err := DeleteAVD(context.Background(), "test", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Deleted || res.Used != "avdmanager" || !strings.HasSuffix(got[0], "avdmanager") {
		t.Fatalf("res = %+v cmd = %v", res, got)
	}
	if strings.Join(got, " ") != got[0]+" delete avd -n test" {
		t.Fatalf("cmd = %v", got)
	}
}

func TestPreviewUsesAbsoluteTool(t *testing.T) {
	opts, _ := fixture(t)
	if got := Preview("delete-avd", "test", opts); !strings.Contains(got, filepath.Join(opts.SDK, "cmdline-tools", "latest", "bin", "avdmanager")) {
		t.Fatalf("preview = %q", got)
	}
	if got := Preview("uninstall", "system-images;android-35;google_apis;x86_64", opts); !strings.Contains(got, "--uninstall") {
		t.Fatalf("preview = %q", got)
	}
	if got := Preview("wipe", "test", opts); !strings.Contains(got, "-wipe-data") {
		t.Fatalf("preview = %q", got)
	}
}
