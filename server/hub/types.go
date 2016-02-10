package hub

// For detecting incoming message type.
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

type SubscribeResponseS2C struct {
	Type     string
	AgentId  int
	ClientId int // id for this client
}

type ValueS2C struct {
	Type  string
	Key   string
	DType string
	Value string // encoded
}

type PatchS2C struct {
	Type    string
	AgentId int  // agent that created this patch
	IsLocal bool // true iff patch originated from this client (on this agent)
	Key     string
	DType   string // "delete" means, delete this record
	Patch   string // encoded
}

////////////////////////////////////////////////////////////
// Initiator-to-responder messages

type SubscribeI2R struct {
	Type          string
	AgentId       int // initiator's agent id
	VersionVector map[int]int
}

////////////////////////////////////////////////////////////
// Responder-to-initiator messages

type SubscribeResponseR2I struct {
	Type    string
	AgentId int // responder's agent id
}

type PatchR2I struct {
	Type     string
	AgentId  int // agent that created this patch
	AgentSeq int // position in sequence of patches created by AgentId
	Key      string
	DType    string
	Patch    string // encoded
}
