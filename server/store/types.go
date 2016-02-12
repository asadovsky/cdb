package store

import (
	"time"

	"github.com/asadovsky/cdb/server/dtypes"
)

// ValueEnvelope represents a value and its associated metadata.
// Key is of the form [Key], where Key is the object key.
type ValueEnvelope struct {
	DType string
	Value dtypes.CValue
}

// PatchEnvelope represents a patch and its associated metadata.
// Key is of the form [AgentId]:[AgentSeq], where AgentId is the creator's agent
// id and [AgentSeq] is the position in the sequence of patches created by
// AgentId.
type PatchEnvelope struct {
	LocalSeq int // position in local, cross-agent patch log
	Time     time.Time
	Key      string
	DType    string
	Patch    string // encoded
}
