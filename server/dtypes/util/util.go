// Package util defines various helper functions.
package util

import (
	"fmt"

	"github.com/asadovsky/cdb/server/dtypes/cregister"
	"github.com/asadovsky/cdb/server/dtypes/cstring"
	"github.com/asadovsky/cdb/server/dtypes/cvalue"
)

// DecodeValue decodes the given value.
func DecodeValue(dtype, value string) (cvalue.CValue, error) {
	switch dtype {
	case cvalue.DTypeCRegister:
		return cregister.Decode(value)
	case cvalue.DTypeCString:
		return cstring.Decode(value)
	default:
		return nil, fmt.Errorf("unknown dtype: %s", dtype)
	}
}

// NewZeroValue returns a new zero value of the given dtype.
func NewZeroValue(dtype string) (cvalue.CValue, error) {
	switch dtype {
	case cvalue.DTypeCRegister:
		return cregister.New(), nil
	case cvalue.DTypeCString:
		return cstring.New(), nil
	default:
		return nil, fmt.Errorf("unknown dtype: %s", dtype)
	}
}
