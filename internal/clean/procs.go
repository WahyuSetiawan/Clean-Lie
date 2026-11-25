package clean

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type procRule struct {
	needle string
	procs  []string
	label  string
}

// procRules memetakan path cache ke proses yang membuatnya berbahaya untuk
// dihapus saat masih hidup. Sengaja sempit: hanya browser/IDE/emulator.
var procRules = []procRule{
	{"BraveSoftware", []string{"brave"}, "Brave"},
	{"mozilla", []string{"firefox"}, "Firefox"},
	{"google-chrome", []string{"chrome"}, "Chrome"},
	{"chromium", []string{"chromium"}, "Chromium"},
	{"microsoft-edge", []string{"msedge"}, "Edge"},
	{"JetBrains", []string{"idea", "pycharm", "webstorm", "goland", "clion", "rustrover", "datagrip", "phpstorm"}, "JetBrains IDE"},
	{"Android/Sdk/emulator", []string{"emulator"}, "emulator Android"},
	{".android/avd", []string{"qemu-system"}, "emulator Android"},
}

// Processes membaca /proc/*/comm → pid. Bisa diarahkan ke root palsu di test.
func Processes() map[string]int {
	return processesFrom("/proc")
}

func processesFrom(procRoot string) map[string]int {
	out := map[string]int{}
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return out
	}
	self := os.Getpid()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == self {
			continue
		}
		b, err := os.ReadFile(filepath.Join(procRoot, e.Name(), "comm"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(b))
		if name != "" {
			out[name] = pid
		}
	}
	return out
}

// BlockedReason mengembalikan alasan item tidak boleh dihapus, atau "".
func BlockedReason(path string) string {
	return blockedReason(path, Processes())
}

func blockedReason(path string, procs map[string]int) string {
	for _, r := range procRules {
		if !strings.Contains(path, r.needle) {
			continue
		}
		for _, name := range r.procs {
			if pid, ok := findProc(procs, name); ok {
				return r.label + " sedang berjalan (pid " + strconv.Itoa(pid) + ") — tutup dulu"
			}
		}
	}
	return ""
}

func findProc(procs map[string]int, name string) (int, bool) {
	if pid, ok := procs[name]; ok {
		return pid, true
	}
	// comm dibatasi 15 karakter, jadi "qemu-system-x86" muncul terpotong.
	for comm, pid := range procs {
		if strings.HasPrefix(comm, name+"-") || strings.HasPrefix(comm, name[:min(len(name), 15)]) {
			return pid, true
		}
	}
	return 0, false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
