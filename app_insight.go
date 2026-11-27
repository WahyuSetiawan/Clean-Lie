package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/insight"
)

// Insight mengembalikan ringkasan mingguan, tren disk, dan sesi terakhir.
func (a *App) Insight() (insight.Report, error) {
	return insight.Collect(a.baseCtx(), a.version, time.Now())
}

// ExportReport merender laporan ke state dir Cleanlie dan mengembalikan path.
// Format yang diterima: md, json, csv.
func (a *App) ExportReport(format string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	ext := format
	if ext == "markdown" {
		ext = "md"
	}
	if ext != "md" && ext != "json" && ext != "csv" {
		return "", fmt.Errorf("format laporan tidak dikenal: %s (pakai md, json, atau csv)", format)
	}
	rep, err := a.Insight()
	if err != nil {
		return "", err
	}
	content, err := insight.Render(rep, format)
	if err != nil {
		return "", err
	}
	dir := insight.DefaultReportsDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := "cleanlie-" + time.Now().Format("20060102-150405") + "." + ext
	path := filepath.Join(dir, name)
	tmp, err := os.CreateTemp(dir, ".report-*."+ext)
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return "", err
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", err
	}
	return path, nil
}
