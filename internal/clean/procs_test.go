package clean

import "testing"

func TestParsePS(t *testing.T) {
	out := "  123 /Applications/Brave Browser.app/Contents/MacOS/Brave Browser\n" +
		"  124 /usr/libexec/qemu-system-aarch64\n" +
		"  125 idea\n" +
		"bukan-angka /bin/ls\n" +
		"\n"

	procs := parsePS(out)
	if pid, ok := procs["Brave Browser"]; !ok || pid != 123 {
		t.Fatalf("Brave Browser = %d/%v, procs=%v", pid, ok, procs)
	}
	if pid, ok := procs["qemu-system-aarch64"]; !ok || pid != 124 {
		t.Fatalf("qemu = %d/%v", pid, ok)
	}
	if pid, ok := procs["idea"]; !ok || pid != 125 {
		t.Fatalf("idea = %d/%v", pid, ok)
	}
	if _, ok := procs["ls"]; ok {
		t.Fatal("baris dengan pid tidak valid tidak boleh masuk")
	}
}

func TestProcessesTidakPanik(t *testing.T) {
	// Di Linux membaca /proc, di macOS memanggil ps. Keduanya tidak boleh panik.
	if got := Processes(); got == nil {
		t.Fatal("Processes() tidak boleh nil")
	}
}
