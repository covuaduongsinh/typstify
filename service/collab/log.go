package collab

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// maxRecord bounds one stored update, so a corrupted length prefix can't
// make loading allocate gigabytes.
const maxRecord = 64 << 20

// updateLog is an append-only file of length-prefixed Yjs updates. The
// whole log is kept in memory while its room is open (rooms hold one
// file's history, and compaction keeps it short).
type updateLog struct {
	path    string
	records [][]byte
	bytes   int
}

func openLog(path string) (*updateLog, error) {
	l := &updateLog{path: path}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	var valid int64 // end of the last complete record
	for {
		n, err := readUvarint(r)
		if err == io.EOF {
			f.Close()
			return l, nil
		}
		if err != nil || n > maxRecord {
			break
		}
		rec := make([]byte, n)
		if _, err := io.ReadFull(r, rec); err != nil {
			break
		}
		l.records = append(l.records, rec)
		l.bytes += len(rec)
		valid += int64(len(appendVarUint(nil, n))) + int64(n)
	}
	f.Close()
	// A torn final write (crash mid-append) loses only that record; cut it
	// off so the next append does not land behind garbage.
	if err := os.Truncate(path, valid); err != nil {
		return nil, err
	}
	return l, nil
}

func readUvarint(r io.ByteReader) (uint64, error) {
	var v uint64
	for i := 0; i < 10; i++ {
		c, err := r.ReadByte()
		if err != nil {
			if i > 0 && err == io.EOF {
				return 0, io.ErrUnexpectedEOF
			}
			return 0, err
		}
		v |= uint64(c&0x7f) << (7 * i)
		if c < 0x80 {
			return v, nil
		}
	}
	return 0, errors.New("varint overflow")
}

func encodeRecord(rec []byte) []byte {
	return append(appendVarUint(nil, uint64(len(rec))), rec...)
}

func (l *updateLog) append(rec []byte) error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(encodeRecord(rec))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	l.records = append(l.records, rec)
	l.bytes += len(rec)
	return nil
}

// replace rewrites the log atomically with the given records.
func (l *updateLog) replace(records [][]byte) error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(l.path), "."+filepath.Base(l.path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	w := bufio.NewWriter(tmp)
	total := 0
	for _, rec := range records {
		if _, err := w.Write(encodeRecord(rec)); err != nil {
			tmp.Close()
			return err
		}
		total += len(rec)
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		return err
	}
	// Durable before it replaces the log: a crash must leave the old log
	// or the new one, never an empty file (which would reseed the room).
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), l.path); err != nil {
		return err
	}
	l.records = records
	l.bytes = total
	return nil
}
