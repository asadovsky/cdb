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
	DeviceId int
	ClientId int // id for this client
}

type ValueS2C struct {
	Type  string
	Key   string
	DType string
	Value string // encoded
}

type PatchS2C struct {
	Type     string
	DeviceId int  // device that created this patch
	IsLocal  bool // true iff patch originated from this client (on this device)
	Key      string
	DType    string // "delete" means, delete this record
	Patch    string // encoded
}

////////////////////////////////////////////////////////////
// Initiator-to-responder messages

type SubscribeI2R struct {
	Type          string
	DeviceId      int // initiator's device id
	VersionVector map[int]int
}

////////////////////////////////////////////////////////////
// Responder-to-initiator messages

type SubscribeResponseR2I struct {
	Type     string
	DeviceId int // responder's device id
}

type PatchR2I struct {
	Type      string
	DeviceId  int // device that created this patch
	DeviceSeq int // position in sequence of patches created by DeviceId
	Key       string
	DType     string
	Patch     string // encoded
}
