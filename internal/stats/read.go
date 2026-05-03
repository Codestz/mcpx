package stats

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"
)

// Filter narrows which records are returned by Iter and Aggregate.
// Zero values mean "no filter". Time bounds are inclusive of Since, exclusive of Until.
type Filter struct {
	Since   time.Time
	Until   time.Time
	Project string // exact match; "" = any
	Server  string // exact match; "" = any
	Tool    string // exact match; "" = any
	Session string // exact match; "" = any
	Agent   string // exact match; "" = any
}

// Iter walks records in the file, calling fn for each match.
// Stop iteration by returning io.EOF from fn. Other errors propagate.
// A missing file yields zero records and no error.
func Iter(path string, f Filter, fn func(Record) error) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(line, &r); err != nil {
			continue // skip corrupt lines silently
		}
		if !f.matches(r) {
			continue
		}
		if err := fn(r); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
	return scanner.Err()
}

func (f Filter) matches(r Record) bool {
	if !f.Since.IsZero() && r.TS.Before(f.Since) {
		return false
	}
	if !f.Until.IsZero() && !r.TS.Before(f.Until) {
		return false
	}
	if f.Project != "" && r.Project != f.Project {
		return false
	}
	if f.Server != "" && r.Server != f.Server {
		return false
	}
	if f.Tool != "" && r.Tool != f.Tool {
		return false
	}
	if f.Session != "" && r.Session != f.Session {
		return false
	}
	if f.Agent != "" && r.Agent != f.Agent {
		return false
	}
	return true
}

// Collect is a convenience wrapper that returns all matching records as a slice.
// For large files prefer Iter to keep memory bounded.
func Collect(path string, f Filter) ([]Record, error) {
	var out []Record
	err := Iter(path, f, func(r Record) error {
		out = append(out, r)
		return nil
	})
	return out, err
}
