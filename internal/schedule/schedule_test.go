package schedule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeInterval(t *testing.T) {
	for _, in := range []string{"", "weekly", "WEEKLY"} {
		if got, err := NormalizeInterval(in); err != nil || got != Weekly {
			t.Fatalf("%q -> %q %v", in, got, err)
		}
	}
	if got, err := NormalizeInterval("daily"); err != nil || got != Daily {
		t.Fatalf("daily -> %q %v", got, err)
	}
	if _, err := NormalizeInterval("hourly"); err == nil {
		t.Fatal("interval asing harus error")
	}
}

func TestRender(t *testing.T) {
	svc, tmr, err := Render("/home/u/bin/cleanlie", "daily")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svc, "ExecStart=/home/u/bin/cleanlie --auto-clean --yes --json") {
		t.Fatalf("service = %s", svc)
	}
	if !strings.Contains(tmr, "OnCalendar=daily") || !strings.Contains(tmr, "WantedBy=timers.target") {
		t.Fatalf("timer = %s", tmr)
	}
	quoted, _, err := Render("/home/u/My Apps/cleanlie", "")
	if err != nil || !strings.Contains(quoted, `ExecStart="/home/u/My Apps/cleanlie"`) {
		t.Fatalf("quoted = %s err=%v", quoted, err)
	}
}

func TestInstallRemove(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CLEANLIE_NO_SYSTEMCTL", "1")

	st, err := Install("/bin/cleanlie", "weekly", false)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Installed {
		t.Fatalf("status = %+v", st)
	}
	svc, tmr := Paths()
	for _, p := range []string{svc, tmr} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s tidak ditulis: %v", p, err)
		}
	}
	got, err := ReadStatus()
	if err != nil || !got.Installed || got.Interval != Weekly {
		t.Fatalf("read = %+v %v", got, err)
	}

	if _, err := Remove(false); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{svc, tmr} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s masih ada", p)
		}
	}
	if st, _ := ReadStatus(); st.Installed {
		t.Fatalf("status setelah hapus = %+v", st)
	}
}

func TestInstallDryRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	st, err := Install("/bin/cleanlie", "daily", true)
	if err != nil {
		t.Fatal(err)
	}
	if st.Installed || !strings.Contains(st.Detail, "dry-run") {
		t.Fatalf("status = %+v", st)
	}
	svc, _ := Paths()
	if _, err := os.Stat(svc); !os.IsNotExist(err) {
		t.Fatalf("dry-run menulis %s", filepath.Dir(svc))
	}
}
