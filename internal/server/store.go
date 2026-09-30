package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type Event struct {
	Sequence int       `json:"sequence"`
	Type     string    `json:"type"`
	Revision int       `json:"revision"`
	At       time.Time `json:"at"`
}
type Session struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Format    string    `json:"format"`
	Mode      string    `json:"mode"`
	ExpiresAt time.Time `json:"expiresAt"`
	Revision  int       `json:"revision"`
	Events    []Event   `json:"events"`
	Sequence  int       `json:"sequence"`
	Data      []byte    `json:"data,omitempty"`
}

func (s *Session) event(kind string) {
	s.Sequence++
	s.Events = append(s.Events, Event{s.Sequence, kind, s.Revision, time.Now().UTC()})
	if len(s.Events) > 256 {
		s.Events = s.Events[len(s.Events)-256:]
	}
}

type Storage interface {
	Get(string) (*Session, error)
	Put(*Session) error
	Delete(string) error
	Sweep(time.Time) (int, error)
}
type DiskStore struct{ Dir string }

var validID = regexp.MustCompile(`^[a-f0-9]{48}$`)

func (s DiskStore) filename(id string) (string, error) {
	if !validID.MatchString(id) {
		return "", errors.New("invalid session ID")
	}
	return filepath.Join(s.Dir, id+".json"), nil
}
func (s DiskStore) Get(id string) (*Session, error) {
	p, e := s.filename(id)
	if e != nil {
		return nil, e
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return nil, e
	}
	var v Session
	e = json.Unmarshal(b, &v)
	return &v, e
}
func (s DiskStore) Put(v *Session) error {
	p, e := s.filename(v.ID)
	if e != nil {
		return e
	}
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(s.Dir, ".snapshot-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(name, p)
}
func (s DiskStore) Delete(id string) error {
	p, e := s.filename(id)
	if e != nil {
		return e
	}
	return os.Remove(p)
}
func (s DiskStore) Sweep(now time.Time) (int, error) {
	entries, e := os.ReadDir(s.Dir)
	if e != nil {
		return 0, e
	}
	count := 0
	for _, f := range entries {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		id := f.Name()[:len(f.Name())-5]
		if !validID.MatchString(id) {
			continue
		}
		v, e := s.Get(id)
		if e != nil {
			return count, e
		}
		if !now.Before(v.ExpiresAt) {
			if e = s.Delete(id); e != nil {
				return count, e
			}
		} else {
			count++
		}
	}
	return count, nil
}
