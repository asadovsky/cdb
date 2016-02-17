// Package cvalue defines the CValue interface and related things.
package cvalue

import (
	"time"

	"github.com/asadovsky/cdb/server/common"
)

const (
	DTypeCRegister = "cregister"
	DTypeCString   = "cstring"
	DTypeDelete    = "delete"
)

// TODO: Switch to using []byte for encoded values, here and elsewhere.

// CValue is an interface for CDB values.
type CValue interface {
	// DType returns this value's dtype.
	DType() string

	// Encode encodes this value.
	Encode() (string, error)

	// ApplyServerPatch applies the given encoded patch to this value.
	ApplyServerPatch(patch string) error

	// ApplyClientPatch applies the given encoded patch to this value and returns
	// an encoded "server patch" suitable for persistent storage. The provided
	// patch may include client-only operations; the returned patch will never
	// contain such operations.
	ApplyClientPatch(agentId int, vec *common.VersionVector, t time.Time, patch string) (string, error)
}
