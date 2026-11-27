package rules

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCustomRules(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	list, warnings := CustomRules([]CustomRule{
		{Path: "~/work/cache", Risk: RiskSafe, Depth: 2},
		{Path: "~/work/cache", Label: "duplikat"},
		{Path: "~/.ssh", Label: "secret"},
		{Path: "relative", Label: "rel"},
		{Path: "~/whole", Whole: true, Risk: RiskDanger},
	})
	if len(list) != 2 {
		t.Fatalf("rules = %+v warnings=%v", list, warnings)
	}
	if list[0].Group != "custom" || list[0].Depth != 2 || !list[0].IncludeFiles || list[0].Risk != RiskSafe {
		t.Fatalf("rule[0] = %+v", list[0])
	}
	if list[1].Whole != true || list[1].Risk != RiskDanger || list[1].Label != "whole" {
		t.Fatalf("rule[1] = %+v", list[1])
	}
	if list[0].Path != filepath.Join(home, "work", "cache") {
		t.Fatalf("path = %s", list[0].Path)
	}
	if len(warnings) < 2 {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestCustomRulesPlan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "work", "cache")
	if err := os.MkdirAll(filepath.Join(root, "one"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "one", "blob"), []byte("xxxx"), 0o644); err != nil {
		t.Fatal(err)
	}
	list, _ := CustomRules([]CustomRule{{Path: root, Label: "cache kerja", Risk: RiskSafe}})
	rep, err := Plan(context.Background(), list, PlanOptions{Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Groups) != 1 || rep.Groups[0].ID != "custom" || len(rep.Groups[0].Items) == 0 {
		t.Fatalf("plan = %+v", rep.Groups)
	}
	if filepath.Base(rep.Groups[0].Items[0].Path) != "one" {
		t.Fatalf("item = %+v", rep.Groups[0].Items[0])
	}
}
