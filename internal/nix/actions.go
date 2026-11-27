package nix

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/privileged"
	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// GenRequest menghapus generasi. Isi Numbers atau OlderThanDays (bukan
// keduanya). Confirm wajib "GC".
type GenRequest struct {
	Profile       string `json:"profile"`
	Numbers       []int  `json:"numbers,omitempty"`
	OlderThanDays int    `json:"olderThanDays,omitempty"`
	Confirm       string `json:"confirm"`
}

// ActionResult adalah hasil aksi Nix (delete/GC/vacuum).
type ActionResult struct {
	Action      string   `json:"action"`
	Command     []string `json:"command"`
	Preview     string   `json:"preview"`
	SudoCommand string   `json:"sudoCommand,omitempty"`
	NeedsSudo   bool     `json:"needsSudo"`
	Used        string   `json:"used"`
	ExitCode    int      `json:"exitCode"`
	Stdout      string   `json:"stdout,omitempty"`
	Stderr      string   `json:"stderr,omitempty"`
	Error       string   `json:"error,omitempty"`
	DurationMs  int64    `json:"durationMs"`
	Deleted     int      `json:"deleted,omitempty"`
}

var vacuumTargetRe = regexp.MustCompile(`^\d{1,4}[MG]$`)

// DeleteGenerations menghapus generasi non-current dari satu profile.
func DeleteGenerations(ctx context.Context, req GenRequest, opts Options) (ActionResult, error) {
	if req.Confirm != "GC" {
		return ActionResult{}, errors.New(`konfirmasi wajib mengetik "GC"`)
	}
	spec, err := findSpec(req.Profile, opts)
	if err != nil {
		return ActionResult{}, err
	}
	_, gens := loadProfile(ctx, spec, opts, false)
	if len(gens) == 0 {
		return ActionResult{}, fmt.Errorf("profile %s tidak punya generasi", req.Profile)
	}
	existing := map[int]bool{}
	current := map[int]bool{}
	for _, g := range gens {
		existing[g.Number] = true
		if g.Current {
			current[g.Number] = true
		}
	}

	var nums []int
	if req.OlderThanDays > 0 {
		for _, g := range gens {
			if !g.Current && g.AgeDays >= req.OlderThanDays {
				nums = append(nums, g.Number)
			}
		}
	} else {
		nums = append(nums, req.Numbers...)
	}
	nums = uniqueInts(nums)
	if len(nums) == 0 {
		return ActionResult{}, errors.New("tidak ada generasi yang memenuhi kriteria")
	}
	for _, n := range nums {
		if !existing[n] {
			return ActionResult{}, fmt.Errorf("generasi %d tidak ada di profile %s", n, req.Profile)
		}
		if current[n] {
			return ActionResult{}, fmt.Errorf("generasi %d sedang aktif dan tidak bisa dihapus", n)
		}
	}

	sort.Ints(nums)
	cmd := []string{"nix-env", "-p", spec.Path, "--delete-generations"}
	for _, n := range nums {
		cmd = append(cmd, strconv.Itoa(n))
	}

	var res ActionResult
	if spec.System {
		pr := privileged.Execute(ctx, cmd, privileged.Options{
			Runner:   privileged.Runner(opts.Run),
			HasAgent: hasAgent(opts),
			Timeout:  10 * time.Minute,
		})
		res = fromPrivileged("delete-generations", pr)
	} else {
		res = runDirect(ctx, "delete-generations", cmd, opts, 10*time.Minute)
	}
	if res.ExitCode == 0 && !res.NeedsSudo {
		res.Deleted = len(nums)
	}
	return res, nil
}

// CollectGarbage menjalankan `nix-collect-garbage -d` sebagai root.
func CollectGarbage(ctx context.Context, confirm string, opts Options) (ActionResult, error) {
	if confirm != "GC" {
		return ActionResult{}, errors.New(`konfirmasi wajib mengetik "GC"`)
	}
	cmd := []string{"nix-collect-garbage", "-d"}
	pr := privileged.Execute(ctx, cmd, privileged.Options{
		Runner:   privileged.Runner(opts.Run),
		HasAgent: hasAgent(opts),
		Timeout:  15 * time.Minute,
	})
	return fromPrivileged("gc", pr), nil
}

// VacuumJournal mengecilkan journal ke target (mis. "500M").
func VacuumJournal(ctx context.Context, target string, opts Options) (ActionResult, error) {
	if runtime.GOOS == "darwin" {
		return ActionResult{}, errors.New("systemd journal tidak tersedia di macOS")
	}
	target = normalizeVacuumTarget(target)
	if !vacuumTargetRe.MatchString(target) {
		return ActionResult{}, fmt.Errorf("target journal tidak valid: %q", target)
	}
	cmd := []string{"journalctl", "--vacuum-size=" + target}
	pr := privileged.Execute(ctx, cmd, privileged.Options{
		Runner:   privileged.Runner(opts.Run),
		HasAgent: hasAgent(opts),
		Timeout:  5 * time.Minute,
	})
	return fromPrivileged("vacuum-journal", pr), nil
}

func normalizeVacuumTarget(target string) string {
	target = strings.ToUpper(strings.TrimSpace(target))
	target = strings.ReplaceAll(target, " ", "")
	target = strings.TrimSuffix(target, "B")
	return target
}

// GCPreview mengembalikan perintah GC yang akan dijalankan.
func GCPreview(agent bool) string {
	return privileged.Preview([]string{"nix-collect-garbage", "-d"}, agent)
}

// DeletePreview mengembalikan perintah hapus generasi (perkiraan, tanpa
// menjalankan apa pun).
func DeletePreview(profile string, nums []int, home string, agent bool) string {
	spec, err := findSpec(profile, Options{Home: home})
	if err != nil {
		return ""
	}
	nums = uniqueInts(nums)
	if len(nums) == 0 {
		return ""
	}
	cmd := []string{"nix-env", "-p", spec.Path, "--delete-generations"}
	for _, n := range nums {
		cmd = append(cmd, strconv.Itoa(n))
	}
	return privileged.Preview(cmd, agent)
}

func findSpec(name string, opts Options) (profileSpec, error) {
	for _, s := range profileSpecs(opts.home()) {
		if s.Name == name {
			return s, nil
		}
	}
	return profileSpec{}, fmt.Errorf("profile tidak dikenal: %q", name)
}

func hasAgent(opts Options) bool {
	if opts.Agent != nil {
		return opts.Agent()
	}
	return privileged.DetectAgent()
}

func uniqueInts(in []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, n := range in {
		if n <= 0 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func fromPrivileged(action string, pr privileged.Result) ActionResult {
	return ActionResult{
		Action:      action,
		Command:     pr.Command,
		Preview:     pr.Preview,
		SudoCommand: pr.SudoCommand,
		NeedsSudo:   pr.NeedsSudo,
		Used:        pr.Used,
		ExitCode:    pr.ExitCode,
		Stdout:      trimOutput(pr.Stdout),
		Stderr:      trimOutput(pr.Stderr),
		Error:       pr.Error,
		DurationMs:  pr.DurationMs,
	}
}

func runDirect(ctx context.Context, action string, cmd []string, opts Options, timeout time.Duration) ActionResult {
	out := opts.run(ctx, cmd, timeout)
	res := ActionResult{
		Action:     action,
		Command:    cmd,
		Preview:    tools.Preview(cmd),
		Used:       "user",
		ExitCode:   out.ExitCode,
		Stdout:     trimOutput(out.Stdout),
		Stderr:     trimOutput(out.Stderr),
		DurationMs: out.Duration.Milliseconds(),
	}
	if out.Err != nil {
		res.Error = out.Err.Error()
	}
	if res.ExitCode != 0 && res.Error == "" {
		res.Error = fmt.Sprintf("perintah gagal (exit %d)", res.ExitCode)
	}
	return res
}

func trimOutput(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 8000 {
		s = s[:8000] + "\n…"
	}
	return s
}

// JournalPreview mengembalikan perintah vacuum journal.
func JournalPreview(target string, agent bool) string {
	if runtime.GOOS == "darwin" {
		return ""
	}
	return privileged.Preview([]string{"journalctl", "--vacuum-size=" + target}, agent)
}
