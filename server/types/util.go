package types

// Value is an interface for CDB values.
type Value interface {
	// Encode returns the encoded value, suitable for persistent storage.
	Encode() string
	// ApplyPatch applies the given patch to this value and returns a finalized
	// patch suitable for persistent storage. The provided patch may include
	// client-only operations; the returned patch will never contain such
	// operations.
	ApplyPatch(p Patch) (Patch, error)
}

// Patch is an interface for patches to CDB values.
type Patch interface {
	// Encode returns the encoded patch, suitable for persistent storage.
	Encode() string
}

func NewZeroValue(dataType string) Value {
	// FIXME: Implement.
	return nil
}

func DecodePatch(dataType, s string) (Patch, error) {
	// FIXME: Implement.
	return nil, nil
}
