package dtypes

import (
	"fmt"

	"github.com/asadovsky/cdb/server/dtypes/cregister"
	"github.com/asadovsky/cdb/server/dtypes/cstring"
)

func NewZeroValue(dtype string) (CValue, error) {
	switch dtype {
	case DTypeCRegister:
		return cregister.New(), nil
	case DTypeCString:
		return cstring.New(), nil
	default:
		return nil, fmt.Errorf("unknown dtype: %s", dtype)
	}
}
