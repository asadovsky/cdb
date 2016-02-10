package dtypes

import (
	"encoding/json"
	"time"
)

type CRegister struct {
	AgentId int
	Vec     *VersionVector
	Time    time.Time
	Value   interface{}
}

var (
	_ Value = (*CRegister)(nil)
)

func NewCRegister() *CRegister {
	return &CRegister{}
}

func (r *CRegister) Encode() (string, error) {
	// TODO: Check that r.Vec gets encoded properly.
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *CRegister) ApplyPatch(agentId int, vec *VersionVector, t time.Time, patch string) (string, error) {
	// TODO: If the patch had no effect on the value, perhaps we should avoid
	// broadcasting it to subscribers.
	if vec.After(r.Vec) || (!vec.Before(r.Vec) && (t.After(r.Time) || (t.Equal(r.Time) && agentId > r.AgentId))) {
		if err := json.Unmarshal([]byte(patch), &r.Value); err != nil {
			return "", err
		}
		r.AgentId = agentId
		r.Time = t
	}
	return patch, nil
}
