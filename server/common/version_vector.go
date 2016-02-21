package common

import (
	"encoding/json"
)

type VersionVector map[uint32]uint32

var (
	_ json.Marshaler   = (*VersionVector)(nil)
	_ json.Unmarshaler = (*VersionVector)(nil)
)

// MarshalJSON marshals to JSON.
func (vec *VersionVector) MarshalJSON() ([]byte, error) {
	m := map[string]uint32{}
	for k, v := range *vec {
		m[Itoa(k)] = v
	}
	return json.Marshal(m)
}

// UnmarshalJSON unmarshals from JSON.
func (vec *VersionVector) UnmarshalJSON(data []byte) error {
	m := map[string]uint32{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*vec = VersionVector{}
	for k, v := range m {
		ki, err := Atoi(k)
		if err != nil {
			return err
		}
		(*vec)[ki] = v
	}
	return nil
}

// Get returns the sequence number for the given agent.
func (vec *VersionVector) Get(agentId uint32) uint32 {
	return (*vec)[agentId]
}

// Put stores the given sequence number for the given agent.
func (vec *VersionVector) Put(agentId, seq uint32) {
	(*vec)[agentId] = seq
}

// Copy returns a copy of the version vector.
func (vec *VersionVector) Copy() *VersionVector {
	res := &VersionVector{}
	for k, v := range *vec {
		res.Put(k, v)
	}
	return res
}

// Leq returns true iff vec[x] <= other[x] for all x in vec.
func (vec *VersionVector) Leq(other *VersionVector) bool {
	for k, v := range *vec {
		if other.Get(k) < v {
			return false
		}
	}
	return true
}

// Before returns true iff the following two properties hold:
// 1. vec[x] <= other[x] for all x in vec, i.e. vec.Leq(other)
// 2. vec[x] < other[x] for some x in vec
func (vec *VersionVector) Before(other *VersionVector) bool {
	if !vec.Leq(other) {
		return false
	}
	for k, v := range *vec {
		if v < other.Get(k) {
			return true
		}
	}
	return false
}

// After returns other.Before(vec).
func (vec *VersionVector) After(other *VersionVector) bool {
	return other.Before(vec)
}
