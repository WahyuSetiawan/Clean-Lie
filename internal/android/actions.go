package android

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// Result adalah hasil satu aksi Android.
type Result struct {
	Action     string   `json:"action"`
	Target     string   `json:"target"`
	Command    []string `json:"command,omitempty"`
	Preview    string   `json:"preview"`
	Used       string   `json:"used,omitempty"`
	ExitCode   int      `json:"exitCode"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	Error      string   `json:"error,omitempty"`
	DurationMs int64    `json:"durationMs"`
	Freed      int64    `json:"freed"`
	Deleted    bool     `json:"deleted"`
}

// DeleteAVD menghapus AVD berdasarkan nama dari inventory. Perintah resmi
// avdmanager dipakai lebih dulu; fallback manual hanya bila AVD memang tidak
// dikenali avdmanager (mis. AVD lama/invalid).
func DeleteAVD(ctx context.Context, name string, opts Options) (Result, error) {
	st := ReadStatus(ctx, opts)
	avd, ok := findAVD(st, name)
	if !ok {
		return Result{}, fmt.Errorf("AVD %q tidak ditemukan", name)
	}
	if st.EmulatorRunning || avd.Running {
		return Result{}, fmt.Errorf("emulator sedang berjalan; matikan dulu sebelum menghapus AVD")
	}
	if !within(st.AVDHome, avd.Path) {
		return Result{}, fmt.Errorf("path AVD di luar folder AVD: %s", avd.Path)
	}

	before := avd.Bytes + pathSize(ctx, avd.INIPath)
	tool := sdkTool(st.SDKPath, "avdmanager")
	res := Result{Action: "delete-avd", Target: name, ExitCode: -1}
	var out tools.Result
	if tool != "" {
		cmd := []string{tool, "delete", "avd", "-n", name}
		res.Command = cmd
		res.Preview = tools.Preview(cmd)
		res.Used = "avdmanager"
		out = opts.run(ctx, cmd, 10*time.Minute, opts.sdkEnv())
		res.ExitCode, res.Stdout, res.Stderr, res.DurationMs = out.ExitCode, out.Stdout, out.Stderr, out.Duration.Milliseconds()
		if out.Err != nil {
			res.Error = out.Err.Error()
		}
		if res.ExitCode != 0 && res.Error == "" {
			res.Error = fmt.Sprintf("avdmanager gagal (exit %d)", res.ExitCode)
		}
	} else {
		res.Used = "manual"
		res.Preview = "rm -rf " + avd.Path + " " + avd.INIPath
		res.Error = "avdmanager tidak ditemukan"
	}

	if tool == "" || (out.ExitCode != 0 && (out.ExitCode == 126 || out.ExitCode == 127 || manualFallback(out))) {
		if err := removeAVD(st.AVDHome, avd.Path, avd.INIPath); err != nil {
			if res.Error != "" {
				res.Error += " · "
			}
			res.Error += err.Error()
		} else {
			res.Used = "manual"
			res.Error = ""
			res.ExitCode = 0
		}
	}

	if !fsx.Exists(avd.Path) && !fsx.Exists(avd.INIPath) {
		res.Deleted = true
		res.ExitCode = 0
	} else if res.ExitCode == 0 {
		res.Error = "perintah selesai tetapi folder/ini AVD masih ada"
	}
	res.Freed = clampFreed(before - pathSize(ctx, avd.Path, avd.INIPath))
	return res, nil
}

func removeAVD(avdHome, path, ini string) error {
	if !strings.HasSuffix(filepath.Base(path), ".avd") || !within(avdHome, path) {
		return fmt.Errorf("path AVD tidak valid: %s", path)
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if ini != "" && within(avdHome, ini) {
		if err := os.Remove(ini); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func manualFallback(out tools.Result) bool {
	s := strings.ToLower(out.Stdout + " " + out.Stderr)
	return strings.Contains(s, "does not exist") ||
		strings.Contains(s, "not found") ||
		strings.Contains(s, "unknown avd") ||
		strings.Contains(s, "tidak ditemukan")
}

// UninstallPackage meng-uninstall satu paket SDK. Paket yang dipakai AVD
// ditolak, dan semua aksi diblokir saat emulator hidup.
func UninstallPackage(ctx context.Context, id string, opts Options) (Result, error) {
	st := ReadStatus(ctx, opts)
	pkg, ok := findPackage(st, id)
	if !ok {
		return Result{}, fmt.Errorf("paket %q tidak ditemukan", id)
	}
	if len(pkg.UsedBy) > 0 {
		return Result{}, fmt.Errorf("paket %s dipakai AVD: %s", id, strings.Join(pkg.UsedBy, ", "))
	}
	if st.EmulatorRunning {
		return Result{}, fmt.Errorf("emulator sedang berjalan; matikan dulu sebelum uninstall")
	}
	if !within(st.SDKPath, pkg.Path) {
		return Result{}, fmt.Errorf("path paket di luar SDK: %s", pkg.Path)
	}
	tool := sdkTool(st.SDKPath, "sdkmanager")
	if tool == "" {
		return Result{}, missingTool("sdkmanager")
	}
	cmd := []string{tool, "--sdk_root=" + st.SDKPath, "--uninstall", pkg.ID}
	before := pkg.Bytes
	out := opts.run(ctx, cmd, 15*time.Minute, opts.sdkEnv())
	res := Result{
		Action: "uninstall", Target: id, Command: cmd, Preview: tools.Preview(cmd), Used: "sdkmanager",
		ExitCode: out.ExitCode, Stdout: out.Stdout, Stderr: out.Stderr, DurationMs: out.Duration.Milliseconds(),
	}
	if out.Err != nil {
		res.Error = out.Err.Error()
	}
	if !fsx.Exists(pkg.Path) {
		res.Deleted = true
		res.ExitCode = 0
		res.Error = ""
	} else if res.ExitCode == 0 {
		res.Error = "perintah selesai tetapi folder paket masih ada"
	}
	res.Freed = clampFreed(before - pathSize(ctx, pkg.Path))
	return res, nil
}

// WipeAVD menjalankan wipe data resmi. Emulator harus mati; perintah memakai
// -quit-after-boot agar tidak menggantung.
func WipeAVD(ctx context.Context, name string, opts Options) (Result, error) {
	st := ReadStatus(ctx, opts)
	avd, ok := findAVD(st, name)
	if !ok {
		return Result{}, fmt.Errorf("AVD %q tidak ditemukan", name)
	}
	if st.EmulatorRunning || avd.Running {
		return Result{}, fmt.Errorf("emulator sedang berjalan; matikan dulu sebelum wipe")
	}
	tool := sdkTool(st.SDKPath, "emulator")
	if tool == "" {
		return Result{}, missingTool("emulator")
	}
	cmd := []string{
		tool, "-avd", name, "-wipe-data", "-no-window", "-no-audio",
		"-no-snapshot", "-no-boot-anim", "-quit-after-boot", "180",
	}
	before := avd.Bytes
	out := opts.run(ctx, cmd, 6*time.Minute, opts.sdkEnv())
	res := Result{
		Action: "wipe", Target: name, Command: cmd, Preview: tools.Preview(cmd), Used: "emulator",
		ExitCode: out.ExitCode, Stdout: out.Stdout, Stderr: out.Stderr, DurationMs: out.Duration.Milliseconds(),
	}
	if out.Err != nil {
		res.Error = out.Err.Error()
	}
	res.Freed = clampFreed(before - pathSize(ctx, avd.Path))
	return res, nil
}

// Preview mengembalikan perintah yang akan dijalankan, tanpa eksekusi.
func Preview(kind, target string, opts Options) string {
	sdk := opts.sdk()
	switch kind {
	case "delete-avd":
		if tool := sdkTool(sdk, "avdmanager"); tool != "" {
			return tools.Preview([]string{tool, "delete", "avd", "-n", target})
		}
		return "rm -rf " + filepath.Join(opts.avdHome(), target+".avd") + " " + filepath.Join(opts.avdHome(), target+".ini")
	case "uninstall":
		if tool := sdkTool(sdk, "sdkmanager"); tool != "" {
			return tools.Preview([]string{tool, "--sdk_root=" + sdk, "--uninstall", target})
		}
		return "sdkmanager --sdk_root=" + sdk + " --uninstall " + target
	case "wipe":
		if tool := sdkTool(sdk, "emulator"); tool != "" {
			return tools.Preview([]string{tool, "-avd", target, "-wipe-data", "-no-window", "-quit-after-boot", "180"})
		}
		return "emulator -avd " + target + " -wipe-data"
	}
	return ""
}

func sdkTool(sdk, name string) string {
	if sdk == "" {
		return ""
	}
	var candidates []string
	candidates = append(candidates, filepath.Join(sdk, "cmdline-tools", "latest", "bin", name))
	for _, v := range childDirs(filepath.Join(sdk, "cmdline-tools")) {
		candidates = append(candidates, filepath.Join(sdk, "cmdline-tools", v, "bin", name))
	}
	candidates = append(candidates,
		filepath.Join(sdk, "tools", "bin", name),
		filepath.Join(sdk, "emulator", name),
	)
	for _, p := range candidates {
		if fsx.Exists(p) {
			return p
		}
	}
	return ""
}

func pathSize(ctx context.Context, paths ...string) int64 {
	var total int64
	for _, p := range paths {
		if p == "" {
			continue
		}
		if n, _, err := fsx.Usage(ctx, p); err == nil {
			total += n
		}
	}
	return total
}

func clampFreed(n int64) int64 {
	if n < 0 {
		return 0
	}
	return n
}
