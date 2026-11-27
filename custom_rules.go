package main

import (
	"github.com/Juragan-Koding/cleanlie/internal/config"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

// customRules memetakan aturan config ke katalog rules. Validasi utama sudah
// dilakukan config.Normalize; rules.CustomRules menjadi pertahanan kedua.
func customRules(cfg config.Config) []rules.Rule {
	list := make([]rules.CustomRule, 0, len(cfg.CustomRules))
	for _, c := range cfg.CustomRules {
		list = append(list, rules.CustomRule{
			Path: c.Path, Label: c.Label, Risk: rules.Risk(c.Risk), Whole: c.Whole, Depth: c.Depth,
		})
	}
	out, _ := rules.CustomRules(list)
	return out
}
