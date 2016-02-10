package dtypes

type VersionVector map[int]int

// Get returns the sequence number for the given agent.
func (vec *VersionVector) Get(agentId int) (int, bool) {
	seq, ok := (*vec)[agentId]
	return seq, ok
}

// Put stores the given sequence number for the given agent.
func (vec *VersionVector) Put(agentId, seq int) {
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
