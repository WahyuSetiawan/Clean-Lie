package clean

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
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
}

// DefaultAuditPath mengembalikan ~/.local/state/cleanlie/history.jsonl.
func DefaultAuditPath() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "cleanlie", "history.jsonl")
	}
	return filepath.Join(homeDir(), ".local", "state", "cleanlie", "history.jsonl")
}

// AppendAudit menambahkan satu entri JSON ke audit log.
func AppendAudit(path string, e Entry) error {
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
	defer f.Close()

	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = f.Write(b)
	return err
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
