// Package android membaca AVD dan paket Android SDK, lalu menjalankan aksi
// resmi (avdmanager/sdkmanager/emulator). Semua pembacaan bersifat read-only.
package android

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// SuggestedDays adalah umur AVD yang disarankan untuk dihapus.
const SuggestedDays = 90

// AVD adalah satu Android Virtual Device.
type AVD struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	INIPath     string `json:"iniPath"`
	Target      string `json:"target,omitempty"`
	API         string `json:"api,omitempty"`
	ABI         string `json:"abi,omitempty"`
	SystemImage string `json:"systemImage,omitempty"`
	Bytes       int64  `json:"bytes"`
	LastUsed    string `json:"lastUsed,omitempty"`
	AgeDays     int    `json:"ageDays"`
	Running     bool   `json:"running"`
	Suggested   bool   `json:"suggested"`
	Error       string `json:"error,omitempty"`
}

// Package adalah satu paket SDK yang bisa di-uninstall.
type Package struct {
	Kind      string   `json:"kind"`
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Version   string   `json:"version,omitempty"`
	API       string   `json:"api,omitempty"`
	ABI       string   `json:"abi,omitempty"`
	Tag       string   `json:"tag,omitempty"`
	Bytes     int64    `json:"bytes"`
	AgeDays   int      `json:"ageDays"`
	UsedBy    []string `json:"usedBy,omitempty"`
	Suggested bool     `json:"suggested"`
}

// Status adalah snapshot layar Android.
type Status struct {
	AVDs            []AVD     `json:"avds"`
	Packages        []Package `json:"packages"`
	AVDBytes        int64     `json:"avdBytes"`
	SDKBytes        int64     `json:"sdkBytes"`
	SDKPath         string    `json:"sdkPath"`
	AVDHome         string    `json:"avdHome"`
	EmulatorRunning bool      `json:"emulatorRunning"`
	ScannedAt       string    `json:"scannedAt"`
	DurationMs      int64     `json:"durationMs"`
	Errors          []string  `json:"errors,omitempty"`
}

// Runner adalah fungsi eksekusi yang bisa diganti di test.
type Runner func(ctx context.Context, cmd []string, timeout time.Duration, env []string) tools.Result

// Options mengatur pembacaan dan eksekusi.
type Options struct {
	Home     string
	SDK      string
	AVDHome  string
	ProcRoot string
	Now      func() time.Time
	Run      Runner
	Timeout  time.Duration
}

func (o Options) home() string {
	if o.Home != "" {
		return o.Home
	}
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}

func (o Options) sdk() string {
	if o.SDK != "" {
		return o.SDK
	}
	for _, p := range []string{
		os.Getenv("ANDROID_HOME"),
		os.Getenv("ANDROID_SDK_ROOT"),
		filepath.Join(o.home(), "Android", "Sdk"),
		filepath.Join(o.home(), "Android", "sdk"),
	} {
		if p != "" && fsx.IsDir(p) {
			return p
		}
	}
	return ""
}

func (o Options) avdHome() string {
	if o.AVDHome != "" {
		return o.AVDHome
	}
	if p := os.Getenv("ANDROID_AVD_HOME"); p != "" {
		return p
	}
	return filepath.Join(o.home(), ".android", "avd")
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o Options) procRoot() string {
	if o.ProcRoot != "" {
		return o.ProcRoot
	}
	return "/proc"
}

func (o Options) run(ctx context.Context, cmd []string, timeout time.Duration, env []string) tools.Result {
	if timeout <= 0 {
		timeout = o.Timeout
	}
	if o.Run != nil {
		return o.Run(ctx, cmd, timeout, env)
	}
	return tools.RunEnv(ctx, cmd, timeout, env)
}

func (o Options) sdkEnv() []string {
	sdk := o.sdk()
	if sdk == "" {
		return nil
	}
	return []string{"ANDROID_HOME=" + sdk, "ANDROID_SDK_ROOT=" + sdk}
}

// ReadStatus memindai AVD dan paket SDK. Tidak menjalankan perintah.
func ReadStatus(ctx context.Context, opts Options) Status {
	started := opts.now()
	st := Status{
		SDKPath:   opts.sdk(),
		AVDHome:   opts.avdHome(),
		ScannedAt: started.Format(time.RFC3339),
	}
	if st.SDKPath == "" {
		st.Errors = append(st.Errors, "SDK Android tidak ditemukan (cek ANDROID_HOME)")
	}

	lines := emulatorProcesses(opts.procRoot())
	st.EmulatorRunning = len(lines) > 0
	st.AVDs = readAVDs(ctx, st.AVDHome, opts.now(), lines, &st)
	for _, a := range st.AVDs {
		st.AVDBytes += a.Bytes
	}
	if st.SDKPath != "" {
		st.Packages = readPackages(ctx, st.SDKPath, opts.now(), st.AVDs, &st)
		for _, p := range st.Packages {
			st.SDKBytes += p.Bytes
		}
	}

	sort.Slice(st.AVDs, func(i, j int) bool { return st.AVDs[i].Name < st.AVDs[j].Name })
	sort.Slice(st.Packages, func(i, j int) bool {
		if kindOrder(st.Packages[i].Kind) != kindOrder(st.Packages[j].Kind) {
			return kindOrder(st.Packages[i].Kind) < kindOrder(st.Packages[j].Kind)
		}
		return st.Packages[i].ID < st.Packages[j].ID
	})
	st.DurationMs = time.Since(started).Milliseconds()
	return st
}

func kindOrder(k string) int {
	switch k {
	case "system-image":
		return 0
	case "ndk":
		return 1
	case "build-tools":
		return 2
	case "platform":
		return 3
	case "emulator":
		return 4
	}
	return 9
}

func readAVDs(ctx context.Context, avdHome string, now time.Time, procLines []string, st *Status) []AVD {
	entries, err := os.ReadDir(avdHome)
	if err != nil {
		if !os.IsNotExist(err) {
			st.Errors = append(st.Errors, "avd: "+err.Error())
		}
		return nil
	}
	seen := map[string]bool{}
	var out []AVD
	for _, e := range entries {
		name := ""
		switch {
		case strings.HasSuffix(e.Name(), ".ini") && !e.IsDir():
			name = strings.TrimSuffix(e.Name(), ".ini")
		case e.IsDir() && strings.HasSuffix(e.Name(), ".avd"):
			name = strings.TrimSuffix(e.Name(), ".avd")
		default:
			continue
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, loadAVD(ctx, avdHome, name, now, procLines, st))
	}
	return out
}

func loadAVD(ctx context.Context, avdHome, name string, now time.Time, procLines []string, st *Status) AVD {
	a := AVD{Name: name, INIPath: filepath.Join(avdHome, name+".ini")}
	ini := readIni(a.INIPath)
	a.Path = ini["path"]
	if a.Path == "" {
		a.Path = filepath.Join(avdHome, name+".avd")
	} else if !filepath.IsAbs(a.Path) {
		a.Path = filepath.Join(avdHome, a.Path)
	}
	if !fsx.IsDir(a.Path) {
		a.Error = "folder AVD tidak ditemukan"
		return a
	}
	if !fsx.Exists(a.INIPath) {
		a.Error = "file .ini tidak ditemukan"
	}

	cfg := readIni(filepath.Join(a.Path, "config.ini"))
	a.Target = ini["target"]
	a.API = strings.TrimPrefix(a.Target, "android-")
	sysdir := cfg["image.sysdir.1"]
	if api, tag, abi := imageParts(sysdir); api != "" {
		a.API = strings.TrimPrefix(api, "android-")
		a.ABI = abi
		if tag != "" && abi != "" {
			a.SystemImage = "system-images;" + api + ";" + tag + ";" + abi
		}
	}
	if a.ABI == "" {
		a.ABI = cfg["abi.type"]
	}
	if a.ABI == "" {
		a.ABI = cfg["hw.cpu.arch"]
	}
	if a.API == "" {
		a.API = apiFromSysdir(sysdir)
	}

	if bytes, _, err := fsx.Usage(ctx, a.Path); err == nil {
		a.Bytes = bytes
	} else {
		st.Errors = append(st.Errors, name+": "+err.Error())
	}
	if used := lastUsed(a.Path); !used.IsZero() {
		a.LastUsed = used.Format("2006-01-02 15:04")
		a.AgeDays = int(now.Sub(used).Hours() / 24)
		if a.AgeDays < 0 {
			a.AgeDays = 0
		}
	}
	a.Running = processUsesAVD(procLines, name, a.Path)
	a.Suggested = !a.Running && a.Error == "" && a.AgeDays >= SuggestedDays
	return a
}

func readPackages(ctx context.Context, sdk string, now time.Time, avds []AVD, st *Status) []Package {
	var out []Package
	add := func(kind, id, path, version, api, abi, tag string) {
		p := Package{Kind: kind, ID: id, Name: version, Path: path, Version: version, API: api, ABI: abi, Tag: tag}
		if bytes, _, err := fsx.Usage(ctx, path); err == nil {
			p.Bytes = bytes
		} else {
			st.Errors = append(st.Errors, id+": "+err.Error())
		}
		if fi, err := os.Stat(path); err == nil {
			p.AgeDays = int(now.Sub(fi.ModTime()).Hours() / 24)
			if p.AgeDays < 0 {
				p.AgeDays = 0
			}
		}
		out = append(out, p)
	}

	sysBase := filepath.Join(sdk, "system-images")
	for _, api := range childDirs(sysBase) {
		for _, tag := range childDirs(filepath.Join(sysBase, api)) {
			for _, abi := range childDirs(filepath.Join(sysBase, api, tag)) {
				id := "system-images;" + api + ";" + tag + ";" + abi
				add("system-image", id, filepath.Join(sysBase, api, tag, abi), api, strings.TrimPrefix(api, "android-"), abi, tag)
			}
		}
	}

	for _, item := range []struct {
		dir  string
		kind string
		pre  string
	}{
		{"ndk", "ndk", "ndk"},
		{"build-tools", "build-tools", "build-tools"},
		{"platforms", "platform", "platforms"},
	} {
		base := filepath.Join(sdk, item.dir)
		for _, v := range childDirs(base) {
			add(item.kind, item.pre+";"+v, filepath.Join(base, v), v, "", "", "")
		}
	}
	if dir := filepath.Join(sdk, "emulator"); fsx.IsDir(dir) {
		add("emulator", "emulator", dir, "emulator", "", "", "")
	}

	for i := range out {
		for _, a := range avds {
			if a.SystemImage != "" && a.SystemImage == out[i].ID {
				out[i].UsedBy = append(out[i].UsedBy, a.Name)
				continue
			}
			if out[i].Kind == "system-image" && a.API != "" && a.ABI != "" && a.API == out[i].API && a.ABI == out[i].ABI {
				out[i].UsedBy = append(out[i].UsedBy, a.Name)
			}
		}
		out[i].Suggested = out[i].Kind == "system-image" && len(out[i].UsedBy) == 0 && out[i].AgeDays >= SuggestedDays
	}
	return out
}

func childDirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func imageParts(sysdir string) (api, tag, abi string) {
	parts := strings.Split(filepath.ToSlash(sysdir), "/")
	for i, p := range parts {
		if p == "system-images" && i+3 < len(parts) {
			return parts[i+1], parts[i+2], parts[i+3]
		}
	}
	return "", "", ""
}

func apiFromSysdir(s string) string {
	for _, p := range strings.Split(filepath.ToSlash(s), "/") {
		if strings.HasPrefix(p, "android-") {
			return strings.TrimPrefix(p, "android-")
		}
	}
	return ""
}

func lastUsed(avdPath string) time.Time {
	var newest time.Time
	names := []string{"emu-launch-params.txt", "snapshots"}
	for _, n := range names {
		if fi, err := os.Stat(filepath.Join(avdPath, n)); err == nil && fi.ModTime().After(newest) {
			newest = fi.ModTime()
		}
	}
	for _, pat := range []string{"userdata-qemu.img*", "cache.img*"} {
		for _, p := range glob(filepath.Join(avdPath, pat)) {
			if fi, err := os.Stat(p); err == nil && fi.ModTime().After(newest) {
				newest = fi.ModTime()
			}
		}
	}
	if newest.IsZero() {
		if fi, err := os.Stat(avdPath); err == nil {
			newest = fi.ModTime()
		}
	}
	return newest
}

func glob(pat string) []string {
	m, _ := filepath.Glob(pat)
	return m
}

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

func emulatorProcesses(procRoot string) []string {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		b, err := os.ReadFile(filepath.Join(procRoot, e.Name(), "cmdline"))
		if err != nil || len(b) == 0 {
			continue
		}
		line := strings.ToLower(strings.ReplaceAll(string(b), "\x00", " "))
		if strings.Contains(line, "qemu-system") || (strings.Contains(line, "emulator") && strings.Contains(line, "-avd")) {
			out = append(out, line)
		}
	}
	return out
}

func processUsesAVD(lines []string, name, path string) bool {
	n := strings.ToLower(name)
	p := strings.ToLower(path)
	for _, line := range lines {
		if strings.Contains(line, p) || strings.Contains(line, "-avd "+n) {
			return true
		}
	}
	return false
}

func findAVD(st Status, name string) (AVD, bool) {
	for _, a := range st.AVDs {
		if a.Name == name {
			return a, true
		}
	}
	return AVD{}, false
}

func findPackage(st Status, id string) (Package, bool) {
	for _, p := range st.Packages {
		if p.ID == id {
			return p, true
		}
	}
	return Package{}, false
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func missingTool(name string) error {
	return fmt.Errorf("%s tidak ditemukan di SDK Android", name)
}
