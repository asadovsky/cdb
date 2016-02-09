package store

import (
	"sort"
	"sync"

	"github.com/asadovsky/cdb/server/types"
)

type Store struct {
	// Maps key to value.
	m map[string]*ValueEnvelope
	l *log
}

// OpenStore returns a store.
func OpenStore(mu *sync.Mutex) *Store {
	// TODO: Persistence.
	return &Store{
		m: map[string]*ValueEnvelope{},
		l: &log{
			cond: sync.NewCond(mu),
			m:    map[int][]*PatchEnvelope{},
			head: map[int]int{},
		},
	}
}

// Head returns a new version vector representing current knowledge.
// Mutex must be held.
func (s *Store) Head() map[int]int {
	return s.l.getHead()
}

// ApplyPatch applies the given patch.
// Mutex must be held.
func (s *Store) ApplyPatch(deviceId int, key string, dataType string, patch types.Patch) error {
	value, ok := s.m[key]
	if !ok {
		value = &ValueEnvelope{
			DataType: dataType,
			Value:    types.NewZeroValue(dataType),
		}
		s.m[key] = value
	}
	var err error
	patch, err = value.Value.ApplyPatch(patch)
	if err != nil {
		return err
	}
	// TODO: Commit changes iff there were no errors.
	return s.l.push(deviceId, key, dataType, patch)
}

////////////////////////////////////////////////////////////
// StoreIterator

type StoreIterator struct {
	s    *Store
	keys []string
	pos  int // current position within keys
}

// NewIterator returns an iterator for stored key-value pairs. Iteration order
// matches lexicographic key order. The store must not be modified while the
// iterator is in use.
func (s *Store) NewIterator() *StoreIterator {
	keys := make([]string, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return &StoreIterator{s: s, keys: keys, pos: -1}
}

// Advance advances the iterator, staging the next value. Must be called to
// stage the first value.
func (it *StoreIterator) Advance() bool {
	it.pos++
	return it.pos < len(it.keys)
}

// Key returns the current key.
func (it *StoreIterator) Key() string {
	return it.keys[it.pos]
}

// Value returns the current value.
func (it *StoreIterator) Value() *ValueEnvelope {
	return it.s.m[it.keys[it.pos]]
}

// Err returns a non-nil error iff the iterator encountered an error.
func (it *StoreIterator) Err() error {
	return nil
}
