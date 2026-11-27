package fsx

import "testing"

func TestClassifyHealth(t *testing.T) {
	const gib = int64(1 << 30)
	cases := []struct {
		name  string
		total int64
		free  int64
		want  Health
	}{
		{"tanpa total", 0, 10 * gib, HealthUnknown},
		{"free negatif", 100 * gib, -1, HealthUnknown},
		{"penuh", 100 * gib, 0, HealthCritical},
		{"persen kritis", 100 * gib, 4 * gib, HealthCritical},
		{"absolut kritis", 1000 * gib, 1 * gib, HealthCritical},
		{"persen tipis", 100 * gib, 6 * gib, HealthWarn},
		{"absolut tipis", 150 * gib, 9 * gib, HealthWarn},
		{"sehat", 100 * gib, 40 * gib, HealthOK},
		{"kecil tapi lega", 8 * gib, 7 * gib, HealthWarn},
	}
	for _, c := range cases {
		if got := ClassifyHealth(c.total, c.free); got != c.want {
			t.Errorf("%s: ClassifyHealth(%d,%d) = %s, mau %s", c.name, c.total, c.free, got, c.want)
		}
	}
}
