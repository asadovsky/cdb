package store

import (
	"time"

	"github.com/asadovsky/cdb/server/types"
)

////////////////////////////////////////////////////////////
// In-memory types

// ValueEnvelope is the in-memory representation of a Value and its associated
// metadata.
// Key is of the form [Key], where Key is the object key.
type ValueEnvelope struct {
	DataType string
	Value    types.Value
}

// PatchEnvelope is the in-memory representation of a Patch and its associated
// metadata.
// Key is of the form [DeviceId]:[DeviceSeq], where DeviceId is the creator's
// device id and [DeviceSeq] is the position in the sequence of patches created
// by DeviceId.
type PatchEnvelope struct {
	LocalSeq int // position in local, cross-device patch log
	Time     time.Time
	Key      string
	DataType string
	Patch    types.Patch
}

////////////////////////////////////////////////////////////
// Persisted types

// Note: These types are currently unused, since we do not persist anything.

// SValueEnvelope is an encodable ValueEnvelope.
type SValueEnvelope struct {
	DataType string
	Value    string // encoded
}

// SPatchEnvelope is an encodable PatchEnvelope.
type SPatchEnvelope struct {
	LocalSeq int   // position in local, cross-device patch log
	Time     int64 // UnixNano
	Key      string
	DataType string
	Patch    string // encoded
}
