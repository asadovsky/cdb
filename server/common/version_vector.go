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
func (vec *VersionVector) Get(agentId uint32) (uint32, bool) {
	seq, ok := (*vec)[agentId]
	return seq, ok
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
	for k, va := range *vec {
		if vb, ok := other.Get(k); !ok || vb < va {
			return false
		}
	}
	return true
}

// Before returns true iff the following two properties hold:
// 1. vec[x] <= other[x] for all x in vec
// 2. vec[x] < other[x] for some x in vec
func (vec *VersionVector) Before(other *VersionVector) bool {
	less := false
	for k, va := range *vec {
		vb, ok := other.Get(k)
		if !ok || vb < va {
			return false
		}
		if va < vb {
			less = true
		}
	}
	return less
}

// After returns other.Before(vec).
func (vec *VersionVector) After(other *VersionVector) bool {
	return other.Before(vec)
}
