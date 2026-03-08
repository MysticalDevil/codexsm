package tui

import (
	"bufio"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

func loadPreviewIndexEntry(path, key string) ([]string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var found previewIndexRecord
	ok := false
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec previewIndexRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		if rec.Key != key {
			continue
		}
		if !ok || rec.TouchedAtUnix >= found.TouchedAtUnix {
			found = rec
			ok = true
		}
	}
	if err := sc.Err(); err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return append([]string(nil), found.Lines...), true, nil
}

func upsertPreviewIndex(path string, cap int, record previewIndexRecord) error {
	if cap <= 0 {
		cap = 5000
	}
	entries := make(map[string]previewIndexRecord, cap)

	f, err := os.Open(path)
	if err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := sc.Bytes()
			if len(line) == 0 {
				continue
			}
			var rec previewIndexRecord
			if err := json.Unmarshal(line, &rec); err != nil {
				continue
			}
			if rec.Key == "" {
				continue
			}
			old, ok := entries[rec.Key]
			if !ok || rec.TouchedAtUnix >= old.TouchedAtUnix {
				entries[rec.Key] = rec
			}
		}
		_ = f.Close()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if record.Key != "" {
		entries[record.Key] = record
	}

	list := make([]previewIndexRecord, 0, len(entries))
	for _, rec := range entries {
		list = append(list, rec)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].TouchedAtUnix == list[j].TouchedAtUnix {
			return list[i].Key < list[j].Key
		}
		return list[i].TouchedAtUnix > list[j].TouchedAtUnix
	})
	if len(list) > cap {
		list = list[:cap]
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".preview-index-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	encErr := error(nil)
	for _, rec := range list {
		b, err := json.Marshal(rec)
		if err != nil {
			encErr = err
			break
		}
		if _, err := tmp.Write(b); err != nil {
			encErr = err
			break
		}
		if _, err := tmp.Write([]byte{'\n'}); err != nil {
			encErr = err
			break
		}
	}
	closeErr := tmp.Close()
	if encErr != nil {
		_ = os.Remove(tmpPath)
		return encErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
