package dtypes

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
	// Value returns the native type for this value.
	Value() interface{}
	// Encode encodes this value.
	Encode() (string, error)
	// ApplyPatch applies the given encoded patch to this value and returns a
	// finalized encoded patch suitable for persistent storage. The provided patch
	// may include client-only operations; the returned patch will never contain
	// such operations.
	ApplyPatch(agentId int, vec *common.VersionVector, t time.Time, patch string) (string, error)
}
