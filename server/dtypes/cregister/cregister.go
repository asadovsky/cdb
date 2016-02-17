// Package cregister defines CRegister, a simple last-one-wins CRDT value.
package cregister

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/asadovsky/cdb/server/common"
	"github.com/asadovsky/cdb/server/dtypes/cvalue"
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

// DType implements CValue.DType.
func (r *CRegister) DType() string {
	return cvalue.DTypeCRegister
}

// Encode implements CValue.Encode.
func (r *CRegister) Encode() (string, error) {
	buf, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// Decode decodes the given value into a CRegister.
func Decode(s string) (*CRegister, error) {
	return nil, errors.New("not implemented")
}

func (r *CRegister) applyPatch(other *CRegister) {
	if other.Vec.After(r.Vec) || (!other.Vec.Before(r.Vec) && (other.Time.After(r.Time) || (other.Time.Equal(r.Time) && other.AgentId > r.AgentId))) {
		*r = *other
	}
}

// ApplyServerPatch implements CValue.ApplyServerPatch.
func (r *CRegister) ApplyServerPatch(patch string) error {
	// For server patches, 'patch' is an encoded CRegister.
	var other CRegister
	if err := json.Unmarshal([]byte(patch), &other); err != nil {
		return err
	}
	r.applyPatch(&other)
	return nil
}

// ApplyClientPatch implements CValue.ApplyClientPatch.
func (r *CRegister) ApplyClientPatch(agentId int, vec *common.VersionVector, t time.Time, patch string) (string, error) {
	// For client patches, 'patch' is an encoded value.
	var val interface{}
	if err := json.Unmarshal([]byte(patch), &val); err != nil {
		return "", err
	}
	other := &CRegister{
		AgentId: agentId,
		Vec:     vec,
		Time:    t,
		Val:     val,
	}
	res, err := other.Encode()
	if err != nil {
		return "", err
	}
	r.applyPatch(other)
	return res, nil
}
