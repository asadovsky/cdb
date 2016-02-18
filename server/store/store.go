// Package store defines Store, a key-value CRDT store.
package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/asadovsky/cdb/server/common"
	"github.com/asadovsky/cdb/server/dtypes/cvalue"
	"github.com/asadovsky/cdb/server/dtypes/util"
)

type Store struct {
	Log *Log
	// Maps key to value.
	m map[string]*ValueEnvelope
}

// OpenStore returns a store.
func OpenStore(mu *sync.Mutex) *Store {
	// TODO: Persistence.
	return &Store{
		Log: &Log{
			cond: sync.NewCond(mu),
			m:    map[uint32][]*PatchEnvelope{},
			head: &common.VersionVector{},
		},
		m: map[string]*ValueEnvelope{},
	}
}

// ApplyPatch applies the given encoded patch and returns the local sequence
// number for the written log record. Mutex must be held.
func (s *Store) ApplyPatch(agentId uint32, key string, dtype string, patch string, isClientPatch bool) (uint32, error) {
	// TODO: Handle deletions in such a way that the deletion trumps concurrent
	// ops on the deleted object. Seems we need a tombstone with an attached
	// version vector.
	if dtype == cvalue.DTypeDelete {
		return 0, errors.New("not implemented")
	}
	valueEnv, ok := s.m[key]
	if !ok {
		zeroValue, err := util.NewZeroValue(dtype)
		if err != nil {
			return 0, err
		}
		valueEnv = &ValueEnvelope{DType: dtype, Value: zeroValue}
		s.m[key] = valueEnv
	}
	// Build incremented version vector to pass to Value.ApplyPatch.
	vec := s.Log.Head()
	seq, ok := vec.Get(agentId)
	if ok {
		seq++
	}
	vec.Put(agentId, seq)
	var err error
	if isClientPatch {
		patch, err = valueEnv.Value.ApplyClientPatch(agentId, vec, time.Now(), patch)
	} else {
		valueEnv.Value.ApplyServerPatch(patch)
	}
	if err != nil {
		return 0, err
	}
	// TODO: Commit changes iff there were no errors.
	return s.Log.push(agentId, key, dtype, patch)
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
