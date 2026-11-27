package clean

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	auditMaxBytes = int64(8 << 20) // rotasi saat file lebih besar dari 8 MiB
	auditTailMax  = int64(4 << 20) // batas baca ekor
	auditTailLine = 10000
)

// Entry adalah satu baris audit log (append-only).
type Entry struct {
	Time        string `json:"time"`
	Mode        Mode   `json:"mode"`
	ID          string `json:"id"`
	Path        string `json:"path"`
	BytesBefore int64  `json:"bytesBefore"`
	BytesAfter  int64  `json:"bytesAfter"`
	Freed       int64  `json:"freed"`
	Status      Status `json:"status"`
	Reason      string `json:"reason,omitempty"`
	TrashPath   string `json:"trashPath,omitempty"`
	CommandLine string `json:"commandLine,omitempty"`
	Batch       string `json:"batch,omitempty"`
	Action      string `json:"action,omitempty"`
}

// DefaultAuditPath mengembalikan ~/.local/state/cleanlie/history.jsonl.
func DefaultAuditPath() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "cleanlie", "history.jsonl")
	}
	return filepath.Join(homeDir(), ".local", "state", "cleanlie", "history.jsonl")
}

// NewBatch membuat ID batch: <prefix><UTC>-<4 byte hex>.
func NewBatch(prefix string) string {
	ts := time.Now().UTC().Format("20060102T150405.000Z")
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s%s-%08x", prefix, ts, time.Now().UnixNano())
	}
	return prefix + ts + "-" + hex.EncodeToString(b[:])
}

// AppendAudit menambahkan satu entri JSON ke audit log.
func AppendAudit(path string, e Entry) error {
	return AppendAuditBatch(path, []Entry{e})
}

// AppendAuditBatch menulis beberapa entri dengan satu open/write.
func AppendAuditBatch(path string, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}
	if path == "" {
		path = DefaultAuditPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, e := range entries {
		b, err := json.Marshal(e)
		if err != nil {
			f.Close()
			return err
		}
		if _, err := w.Write(b); err != nil {
			f.Close()
			return err
		}
		if err := w.WriteByte('\n'); err != nil {
			f.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// RotateAudit memindahkan history.jsonl ke history.jsonl.1 bila sudah besar.
// Dipanggil sekali sebelum operasi tulis, bukan per item.
func RotateAudit(path string) error {
	if path == "" {
		path = DefaultAuditPath()
	}
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Size() <= auditMaxBytes {
		return nil
	}
	if err := os.Rename(path, path+".1"); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

// ReadAudit membaca audit log; limit <= 0 berarti semua. Entri terbaru
// tetap di urutan asli (lama → baru).
func ReadAudit(path string, limit int) ([]Entry, error) {
	if path == "" {
		path = DefaultAuditPath()
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Entry
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for s.Scan() {
		var e Entry
		if err := json.Unmarshal(s.Bytes(), &e); err != nil {
			continue // baris rusak tidak menghentikan pembacaan
		}
		out = append(out, e)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

// ReadAuditTail membaca ekor audit (maks 4 MiB / 10.000 baris). Flag
// truncated true bila file lebih besar atau baris dibuang.
func ReadAuditTail(path string, maxLines int) ([]Entry, bool, error) {
	if path == "" {
		path = DefaultAuditPath()
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	truncated := false
	start := int64(0)
	if st, err := f.Stat(); err == nil && st.Size() > auditTailMax {
		start = st.Size() - auditTailMax
		truncated = true
	}
	if start > 0 {
		if _, err := f.Seek(start, 0); err != nil {
			return nil, false, err
		}
	}
	br := bufio.NewReader(f)
	if start > 0 {
		// Baris pertama hampir pasti terpotong.
		if _, err := br.ReadString('\n'); err != nil {
			return nil, truncated, nil
		}
	}

	var out []Entry
	s := bufio.NewScanner(br)
	s.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for s.Scan() {
		var e Entry
		if err := json.Unmarshal(s.Bytes(), &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	if err := s.Err(); err != nil {
		return nil, truncated, err
	}
	if maxLines > 0 && len(out) > maxLines {
		out = out[len(out)-maxLines:]
		truncated = true
	}
	return out, truncated, nil
}

// auditTailLines dipakai history.go untuk membaca kedua file dengan batas
// yang sama.
func auditTailLines(path string) ([]Entry, bool, error) {
	return ReadAuditTail(path, auditTailLine)
}
