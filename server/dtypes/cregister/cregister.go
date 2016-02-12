// Package cregister defines CRegister, a simple last-one-wins CRDT value.
package cregister

import (
	"encoding/json"
	"time"

	"github.com/asadovsky/cdb/server/common"
)

// CRegister is a CRDT register (last-one-wins).
// Fields are exported to support CRegister.Encode.
type CRegister struct {
	AgentId int
	Vec     *common.VersionVector
	Time    time.Time
	Val     interface{}
}

// New returns a new CRegister.
func New() *CRegister {
	return &CRegister{}
}

// Value implements CValue.Value.
func (r *CRegister) Value() interface{} {
	return r.Val
}

// Encode implements CValue.Encode.
func (r *CRegister) Encode() (string, error) {
	buf, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func decodePatch(s string) (interface{}, error) {
	var res interface{}
	err := json.Unmarshal([]byte(s), &res)
	return res, err
}

// ApplyPatch implements CValue.ApplyPatch.
func (r *CRegister) ApplyPatch(agentId int, vec *common.VersionVector, t time.Time, patch string) (string, error) {
	val, err := decodePatch(patch)
	if err != nil {
		return "", err
	}
	// TODO: If the patch had no effect on the value, perhaps we should avoid
	// broadcasting it to subscribers.
	if vec.After(r.Vec) || (!vec.Before(r.Vec) && (t.After(r.Time) || (t.Equal(r.Time) && agentId > r.AgentId))) {
		r.AgentId = agentId
		r.Time = t
		r.Val = val
	}
	return patch, nil
}
