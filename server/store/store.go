// Package store defines Store, a key-value CRDT store.
package store

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/asadovsky/cdb/server/common"
	"github.com/asadovsky/cdb/server/dtypes/cvalue"
	"github.com/asadovsky/cdb/server/dtypes/util"
)

var (
	errNotImplemented = errors.New("not implemented")
)

func assert(b bool, v ...interface{}) {
	if !b {
		panic(fmt.Sprint(v...))
	}
}

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

func (s *Store) getOrCreateValueEnvelope(key, dtype string) (*ValueEnvelope, error) {
	valueEnv, ok := s.m[key]
	if !ok {
		zeroValue, err := util.NewZeroValue(dtype)
		if err != nil {
			return nil, err
		}
		valueEnv = &ValueEnvelope{DType: dtype, Value: zeroValue}
		s.m[key] = valueEnv
	}
	return valueEnv, nil
}

// TODO: Make it so a deletion trumps any concurrent ops on the deleted object.
// In other words, the key-value store should behave like a map CRDT. Seems we
// need a tombstone with an attached version vector.

// ApplyServerPatch applies the given encoded patch, if needed. Mutex must be
// held.
func (s *Store) ApplyServerPatch(agentId, agentSeq uint32, key, dtype, patch string) error {
	if dtype == cvalue.DTypeDelete {
		return errNotImplemented
	}
	vec := s.Log.Head()
	wantSeq := vec.Get(agentId) + 1
	if agentSeq > wantSeq {
		return fmt.Errorf("unexpected patch for agent %d: got %d, want %d", agentId, agentSeq, wantSeq)
	} else if agentSeq < wantSeq {
		log.Printf("already got patch for agent %d: got %d, want %d", agentId, agentSeq, wantSeq)
		return nil
	}
	ve, err := s.getOrCreateValueEnvelope(key, dtype)
	if err != nil {
		return err
	}
	if err := ve.Value.ApplyServerPatch(patch); err != nil {
		return err
	}
	// TODO: Commit changes iff there were no errors.
	_, err = s.Log.push(agentId, key, dtype, patch)
	return err
}

// ApplyClientPatch applies the given encoded patch and returns the local
// sequence number for the written log record. Mutex must be held.
func (s *Store) ApplyClientPatch(agentId uint32, key, dtype, patch string) (uint32, error) {
	if dtype == cvalue.DTypeDelete {
		return 0, errNotImplemented
	}
	// Build incremented version vector to pass to Value.ApplyPatch.
	vec := s.Log.Head()
	vec.Put(agentId, vec.Get(agentId)+1)
	ve, err := s.getOrCreateValueEnvelope(key, dtype)
	if err != nil {
		return 0, err
	}
	patch, err = ve.Value.ApplyClientPatch(agentId, vec, time.Now(), patch)
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
