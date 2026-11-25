package rules

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// androidRules sengaja berisiko tinggi: isinya bukan "regenerate", tapi
// "hilang lalu harus diunduh ulang". UI menampilkannya terpisah.
func androidRules() []Rule {
	h := Home()
	p := func(elem ...string) string { return filepath.Join(append([]string{h}, elem...)...) }

	return []Rule{
		{
			ID: "avd", Group: "dev.android", Label: "AVD", Path: p(".android", "avd"),
			Depth: 1, Risk: RiskDanger, Priority: 30,
			Match:    func(n string) bool { return strings.HasSuffix(n, ".avd") },
			NoteFunc: avdNote,
		},
		{
			ID: "android-sysimg", Group: "dev.android", Label: "system-images", Path: p("Android", "Sdk", "system-images"),
			Depth: 3, Risk: RiskDanger, Priority: 30,
			Note: "system image emulator; harus diunduh ulang",
		},
		{
			ID: "android-ndk", Group: "dev.android", Label: "NDK", Path: p("Android", "Sdk", "ndk"),
			Depth: 1, Risk: RiskDanger, Priority: 30,
			Note: "NDK; build native butuh unduh ulang",
		},
		{
			ID: "android-platforms", Group: "dev.android", Label: "platforms", Path: p("Android", "Sdk", "platforms"),
			Depth: 1, Risk: RiskWarn, Priority: 30,
			Note: "platform SDK per API",
		},
		{
			ID: "android-buildtools", Group: "dev.android", Label: "build-tools", Path: p("Android", "Sdk", "build-tools"),
			Depth: 1, Risk: RiskWarn, Priority: 30,
			Note: "build-tools per versi",
		},
		{
			ID: "android-emulator", Group: "dev.android", Label: "emulator", Path: p("Android", "Sdk", "emulator"),
			Whole: true, Risk: RiskDanger, Priority: 30,
			Note: "emulator; aplikasi tidak bisa dijalankan tanpa ini",
		},
	}
}

// avdNote membaca nama, API, dan ABI dari file .ini AVD.
func avdNote(avdPath string) string {
	name := strings.TrimSuffix(filepath.Base(avdPath), ".avd")
	base := filepath.Dir(avdPath)

	ini := readIni(filepath.Join(base, name+".ini"))
	cfg := readIni(filepath.Join(avdPath, "config.ini"))

	api := strings.TrimPrefix(ini["target"], "android-")
	if api == "" {
		api = apiFromSysdir(cfg["image.sysdir.1"])
	}
	abi := cfg["abi.type"]
	if abi == "" {
		abi = cfg["hw.cpu.arch"]
	}

	parts := []string{name}
	if api != "" {
		parts = append(parts, "API "+api)
	}
	if abi != "" {
		parts = append(parts, abi)
	}
	if used := lastUsed(avdPath); !used.IsZero() {
		parts = append(parts, "terakhir dipakai "+used.Format("2006-01-02"))
	}
	return strings.Join(parts, " · ")
}

func lastUsed(avdPath string) time.Time {
	var newest time.Time
	for _, f := range []string{"userdata-qemu.img", "snapshots", "config.ini"} {
		if fi, err := os.Stat(filepath.Join(avdPath, f)); err == nil && fi.ModTime().After(newest) {
			newest = fi.ModTime()
		}
	}
	return newest
}

// readIni membaca file key=value sederhana (format .ini Android).
func readIni(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

func apiFromSysdir(s string) string {
	for _, part := range strings.Split(s, "/") {
		if strings.HasPrefix(part, "android-") {
			return strings.TrimPrefix(part, "android-")
		}
	}
	return ""
}
