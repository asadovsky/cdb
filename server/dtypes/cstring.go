package dtypes

import (
	"time"
)

// FIXME: Implement.
type CString struct{}

var (
	_ Value = (*CString)(nil)
)

func NewCString() *CString {
	return &CString{}
}

func (s *CString) Encode() (string, error) {
	return "", nil
}

func (r *CString) ApplyPatch(agentId int, vec *VersionVector, t time.Time, patch string) (string, error) {
	return "", nil
}
