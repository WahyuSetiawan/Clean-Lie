package fsx

// Health adalah klasifikasi kapasitas filesystem untuk indikator dashboard.
// Ini bukan SMART; hanya sinyal ruang tersisa.
type Health string

const (
	HealthOK       Health = "ok"
	HealthWarn     Health = "warn"
	HealthCritical Health = "critical"
	HealthUnknown  Health = "unknown"
)

// ClassifyHealth memberi level berdasarkan persentase dan ruang absolut.
// Ambang: kritis <5% atau <2 GiB; tipis <12% atau <10 GiB.
func ClassifyHealth(total, free int64) Health {
	if total <= 0 || free < 0 {
		return HealthUnknown
	}
	if free == 0 {
		return HealthCritical
	}
	pct := float64(free) / float64(total)
	switch {
	case pct < 0.05 || free < 2<<30:
		return HealthCritical
	case pct < 0.12 || free < 10<<30:
		return HealthWarn
	default:
		return HealthOK
	}
}
