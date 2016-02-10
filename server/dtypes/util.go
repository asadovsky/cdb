package dtypes

import (
	"fmt"
	"time"
)

const (
	DTypeCRegister = "cregister"
	DTypeCString   = "cstring"
	DTypeDelete    = "delete"
)

// Value is an interface for CDB values.
type Value interface {
	// Encode returns the encoded value, suitable for persistent storage.
	Encode() (string, error)
	// ApplyPatch applies the given encoded patch to this value and returns a
	// finalized encoded patch suitable for persistent storage. The provided patch
	// may include client-only operations; the returned patch will never contain
	// such operations.
	ApplyPatch(agentId int, vec *VersionVector, t time.Time, patch string) (string, error)
}

func NewZeroValue(dtype string) (Value, error) {
	switch dtype {
	case DTypeCRegister:
		return NewCRegister(), nil
	case DTypeCString:
		return NewCString(), nil
	default:
		return nil, fmt.Errorf("unknown dtype: %s", dtype)
	}
}
