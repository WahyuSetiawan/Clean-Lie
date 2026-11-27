package clean

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"
)

// Session adalah satu batch audit. Batch lama tanpa field batch
// dikelompokkan berdasarkan waktu + mode.
type Session struct {
	ID              string  `json:"id"`
	Batch           string  `json:"batch,omitempty"`
	Action          string  `json:"action,omitempty"`
	Mode            Mode    `json:"mode"`
	StartedAt       string  `json:"startedAt"`
	Removed         int     `json:"removed"`
	RemovedBytes    int64   `json:"removedBytes"`
	Failed          int     `json:"failed"`
	Restorable      int     `json:"restorable"`
	Freed           int64   `json:"freed"`
	RestorableBytes int64   `json:"restorableBytes"`
	Items           []Entry `json:"items"`
	ItemsTruncated  bool    `json:"itemsTruncated,omitempty"`
}

// SessionPage adalah hasil ReadSessions.
type SessionPage struct {
	Sessions  []Session `json:"sessions"`
	Truncated bool      `json:"truncated"`
}

const sessionItemCap = 200

// ReadSessions membaca sesi terbaru lebih dulu. limit <= 0 = 50, maks 200.
func ReadSessions(path string, limit int) (SessionPage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if path == "" {
		path = DefaultAuditPath()
	}

	archive, archTrunc, err := auditTailLines(path + ".1")
	if err != nil {
		return SessionPage{}, err
	}
	current, curTrunc, err := auditTailLines(path)
	if err != nil {
		return SessionPage{}, err
	}
	all := make([]Entry, 0, len(archive)+len(current))
	all = append(all, archive...)
	all = append(all, current...)
	truncated := archTrunc || curTrunc

	page := groupSessions(all, truncated)
	if len(page.Sessions) > limit {
		page.Sessions = page.Sessions[:limit]
		page.Truncated = true
	}
	return page, nil
}

func groupSessions(entries []Entry, tailTruncated bool) SessionPage {
	byKey := map[string]int{}
	var sessions []Session
	for _, e := range entries {
		key := entrySessionKey(e)
		idx, ok := byKey[key]
		if !ok {
			idx = len(sessions)
			byKey[key] = idx
			id := e.Batch
			if id == "" {
				id = legacySessionID(key)
			}
			sessions = append(sessions, Session{
				ID:        id,
				Batch:     e.Batch,
				Action:    e.Action,
				Mode:      e.Mode,
				StartedAt: e.Time,
			})
		}
		s := &sessions[idx]
		if s.Action == "" && e.Action != "" {
			s.Action = e.Action
		}
		if e.Status == StatusRemoved {
			s.Removed++
			s.RemovedBytes += e.BytesBefore
		}
		if e.Status == StatusFailed {
			s.Failed++
		}
		s.Freed += e.Freed
		if restorableEntry(e) {
			s.Restorable++
			if e.BytesBefore > 0 {
				s.RestorableBytes += e.BytesBefore
			}
		}
		if len(s.Items) < sessionItemCap {
			s.Items = append(s.Items, e)
		} else {
			s.ItemsTruncated = true
		}
	}

	sort.SliceStable(sessions, func(i, j int) bool {
		ti, ei := time.Parse(time.RFC3339, sessions[i].StartedAt)
		tj, ej := time.Parse(time.RFC3339, sessions[j].StartedAt)
		if ei == nil && ej == nil && !ti.Equal(tj) {
			return ti.After(tj)
		}
		if sessions[i].StartedAt != sessions[j].StartedAt {
			return sessions[i].StartedAt > sessions[j].StartedAt
		}
		return sessions[i].ID > sessions[j].ID
	})

	page := SessionPage{Sessions: sessions, Truncated: tailTruncated}
	// Ekor file bisa memotong sesi tertua; lebih baik tidak menampilkannya
	// daripada menampilkan sesi setengah.
	if tailTruncated && len(page.Sessions) > 1 {
		page.Sessions = page.Sessions[:len(page.Sessions)-1]
	}
	return page
}

func entrySessionKey(e Entry) string {
	if e.Batch != "" {
		return "batch:" + e.Batch
	}
	return "legacy:" + e.Time + "|" + string(e.Mode)
}

func legacySessionID(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])[:12]
}

// RestorableEntry melaporkan apakah audit entry bisa di-undo.
func RestorableEntry(e Entry) bool { return restorableEntry(e) }

func restorableEntry(e Entry) bool {
	if e.Action != "" && e.Action != "clean" {
		return false
	}
	if e.Mode != ModeTrash || e.Status != StatusRemoved {
		return false
	}
	return e.TrashPath != "" || e.CommandLine == ""
}

// ReadSessionEntries membaca semua item satu sesi dari audit, termasuk
// arsip history.jsonl.1. Dipakai Undo agar tidak terpotong SessionPage.
func ReadSessionEntries(path, id string) ([]Entry, bool, error) {
	if id == "" {
		return nil, false, nil
	}
	if path == "" {
		path = DefaultAuditPath()
	}
	archive, err := ReadAudit(path+".1", 0)
	if err != nil {
		return nil, false, err
	}
	current, err := ReadAudit(path, 0)
	if err != nil {
		return nil, false, err
	}
	all := make([]Entry, 0, len(archive)+len(current))
	all = append(all, archive...)
	all = append(all, current...)

	var out []Entry
	for _, e := range all {
		key := entrySessionKey(e)
		if e.Batch == id || (e.Batch == "" && legacySessionID(key) == id) {
			out = append(out, e)
		}
	}
	return out, len(out) > 0, nil
}

// ReadSessionBatch adalah alias yang lebih eksplisit untuk Undo.
func ReadSessionBatch(path, batch string) ([]Entry, bool, error) {
	return ReadSessionEntries(path, batch)
}
