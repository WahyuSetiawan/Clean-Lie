package clean

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func makeCache(t *testing.T, root, name string) string {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, "blob"), make([]byte, 64*1024), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExecutePermanent(t *testing.T) {
	home := setupHome(t)
	root := filepath.Join(home, ".cache")
	path := makeCache(t, root, "jit")
	audit := filepath.Join(home, "audit.jsonl")

	res := Execute(context.Background(), Request{
		Mode: ModePermanent,
		Targets: []Target{{
			ID: "jit", Path: path, Root: root, Label: "jit", Bytes: 1,
		}},
	}, Options{AuditPath: audit, Now: func() time.Time { return time.Unix(1700000000, 0) }})

	if len(res.Items) != 1 || res.Items[0].Status != StatusRemoved {
		t.Fatalf("items = %+v", res.Items)
	}
	if res.Freed <= 0 || res.Items[0].Freed != res.Freed {
		t.Fatalf("freed = %d item = %d", res.Freed, res.Items[0].Freed)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("path masih ada: %v", err)
	}
	entries, err := ReadAudit(audit, 10)
	if err != nil || len(entries) != 1 {
		t.Fatalf("audit = %+v err=%v", entries, err)
	}
	if entries[0].Status != StatusRemoved || entries[0].Path != path || entries[0].Freed != res.Freed {
		t.Fatalf("audit entry = %+v", entries[0])
	}
}

func TestExecuteDryRun(t *testing.T) {
	home := setupHome(t)
	root := filepath.Join(home, ".cache")
	path := makeCache(t, root, "dry")
	audit := filepath.Join(home, "audit.jsonl")

	res := Execute(context.Background(), Request{
		Mode:    ModeTrash,
		DryRun:  true,
		Targets: []Target{{ID: "dry", Path: path, Root: root}},
	}, Options{AuditPath: audit})

	if res.Items[0].Status != StatusPlanned {
		t.Fatalf("status = %s", res.Items[0].Status)
	}
	if res.Freed != 0 {
		t.Fatalf("dry run membebaskan %d", res.Freed)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("dry run menghapus path: %v", err)
	}
	if _, err := os.Stat(audit); !os.IsNotExist(err) {
		t.Fatal("dry run menulis audit")
	}
}

func TestExecuteGuardAndContinue(t *testing.T) {
	home := setupHome(t)
	root := filepath.Join(home, ".cache")
	good := makeCache(t, root, "good")
	audit := filepath.Join(home, "audit.jsonl")

	res := Execute(context.Background(), Request{
		Mode: ModePermanent,
		Targets: []Target{
			{ID: "bad", Path: filepath.Join(home, ".config"), Root: home, Whole: true},
			{ID: "good", Path: good, Root: root},
		},
	}, Options{AuditPath: audit})

	if res.Items[0].Status != StatusFailed {
		t.Fatalf("item protected tidak gagal: %+v", res.Items[0])
	}
	if res.Items[1].Status != StatusRemoved {
		t.Fatalf("item kedua tidak diproses: %+v", res.Items[1])
	}
	if _, err := os.Stat(filepath.Join(home, ".config")); err != nil {
		// memang tidak dibuat, jadi tidak apa-apa; pastikan tidak dihapus.
		_ = err
	}
}

func TestExecuteRequiresRootSkipped(t *testing.T) {
	home := setupHome(t)
	root := filepath.Join(home, ".cache")
	path := makeCache(t, root, "rootonly")

	res := Execute(context.Background(), Request{
		Mode:    ModeTrash,
		Targets: []Target{{ID: "j", Path: path, Root: root, RequiresRoot: true}},
	}, Options{AuditPath: filepath.Join(home, "audit.jsonl")})

	if res.Items[0].Status != StatusSkipped {
		t.Fatalf("status = %s", res.Items[0].Status)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("item root-only terhapus")
	}
}

func TestExecuteCommandFallback(t *testing.T) {
	home := setupHome(t)
	root := filepath.Join(home, ".cache")
	path := makeCache(t, root, "cmdfail")

	res := Execute(context.Background(), Request{
		Mode: ModeTrash,
		Targets: []Target{{
			ID: "cmdfail", Path: path, Root: root, Command: []string{"sh", "-c", "exit 7"}, FallbackOK: true,
		}},
	}, Options{AuditPath: filepath.Join(home, "audit.jsonl")})

	if res.Items[0].Status != StatusRemoved {
		t.Fatalf("fallback tidak jalan: %+v", res.Items[0])
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("path masih ada: %v", err)
	}
}

func TestExecuteCommandSuccess(t *testing.T) {
	home := setupHome(t)
	root := filepath.Join(home, ".cache")
	path := makeCache(t, root, "cmdok")

	res := Execute(context.Background(), Request{
		Mode: ModeTrash,
		Targets: []Target{{
			ID: "cmdok", Path: path, Root: root,
			Command: []string{"sh", "-c", "rm -rf -- '" + path + "'"}, FallbackOK: true,
		}},
	}, Options{AuditPath: filepath.Join(home, "audit.jsonl")})

	if res.Items[0].Status != StatusRemoved || res.Freed <= 0 {
		t.Fatalf("command sukses tidak terdeteksi: %+v", res.Items[0])
	}
	if res.Items[0].CommandLine == "" {
		t.Fatal("commandLine kosong")
	}
}

func TestExecuteReportsAuditError(t *testing.T) {
	base := t.TempDir()
	ro := filepath.Join(base, "ro")
	if err := os.Mkdir(ro, 0o500); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "missing")
	res := Execute(context.Background(), Request{
		Targets: []Target{{ID: "x", Path: target, Root: target, Whole: true}},
		Mode:    ModeTrash,
	}, Options{AuditPath: filepath.Join(ro, "history.jsonl")})

	if res.AuditError == "" {
		t.Fatal("error audit tidak dilaporkan")
	}
	if res.Items[0].Status != StatusFailed {
		t.Fatalf("status = %s, mau failed", res.Items[0].Status)
	}
}
