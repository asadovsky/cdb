package store

import (
	"github.com/asadovsky/cdb/server/dtypes/cvalue"
)

// ValueEnvelope represents a value and its associated metadata.
// Key is of the form [Key], where Key is the object key.
type ValueEnvelope struct {
	DType string
	Value cvalue.CValue
}

// PatchEnvelope represents a patch and its associated metadata.
// Key is of the form [AgentId]:[AgentSeq], where AgentId is the creator's agent
// id and [AgentSeq] is the creator's sequence number for this patch.
type PatchEnvelope struct {
	LocalSeq uint32 // one-based position in local, cross-agent patch log
	Key      string
	DType    string
	Patch    string // encoded
}
