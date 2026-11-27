package main

import "testing"

func TestMsg(t *testing.T) {
	a := &App{}
	a.cfg.Language = "id"
	if got := a.msg("halo", "hello"); got != "halo" {
		t.Fatalf("id = %q", got)
	}
	a.cfg.Language = "en"
	if got := a.msg("halo", "hello"); got != "hello" {
		t.Fatalf("en = %q", got)
	}
}

func TestScheduleDetailEN(t *testing.T) {
	cases := map[string]string{
		"timer aktif":                                 "timer active",
		"timer belum dipasang":                        "timer not installed",
		"unit ditulis; enable gagal: exit 1":          "unit written; enable failed: exit 1",
		"unit ditulis; daemon-reload gagal: timeout":  "unit written; daemon-reload failed: timeout",
		"kalimat yang tidak dikenal tetap apa adanya": "kalimat yang tidak dikenal tetap apa adanya",
	}
	for in, want := range cases {
		if got := scheduleDetailEN(in); got != want {
			t.Errorf("%q = %q, mau %q", in, got, want)
		}
	}
}
