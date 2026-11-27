package main

// msg memilih teks Go berdasarkan bahasa aktif di config. Dipakai untuk pesan
// yang dibuat backend (notifikasi desktop, detail timer), bukan error teknis.
func (a *App) msg(id, en string) string {
	if a.settings().Language == "en" {
		return en
	}
	return id
}
