package hub

import (
	"github.com/asadovsky/cdb/server/common"
)

// Note: We use uint32 (rather than uint64) in various places to ensure that
// these numbers are representable in JavaScript.

// For detecting incoming message type. Each struct below has Type set to the
// struct type name.
type MsgType struct {
	Type string
}

////////////////////////////////////////////////////////////
// Client-to-server messages

type SubscribeC2S struct {
	Type string
}

type PatchC2S struct {
	Type  string
	Key   string
	DType string // "delete" means, delete this record
	Patch string // encoded
}

////////////////////////////////////////////////////////////
// Server-to-client messages

type ValueS2C struct {
	Type  string
	Key   string
	DType string
	Value string // encoded
}

type ValuesDoneS2C struct {
	Type string
}

type PatchS2C struct {
	Type    string
	AgentId uint32 // agent that created this patch
	IsLocal bool   // true iff patch originated from this client (on this agent)
	Key     string
	DType   string // "delete" means, delete this record
	Patch   string // encoded
}

////////////////////////////////////////////////////////////
// Initiator-to-responder messages

type SubscribeI2R struct {
	Type          string
	AgentId       uint32 // initiator's agent id
	Addr          string // initiator's network address
	VersionVector *common.VersionVector
}

////////////////////////////////////////////////////////////
// Responder-to-initiator messages

type PatchR2I struct {
	Type     string
	AgentId  uint32 // agent that created this patch
	AgentSeq uint32 // creator's sequence number for this patch
	Key      string
	DType    string
	Patch    string // encoded
}
