This document contains a design sketch for a CRDT store.

Last major update: Feb 7, 2016

# Data model

## Hierarchy

Service > Collection > Record

Some record types (namely, List and Map) are composable.

## Supported record types

- Register (atomic unit, last-one-wins)
- String
- List
- Map

TODO: Add Struct, i.e. a more strongly typed Map?

# Client API

JavaScript, for now.

Types:
- Non-value types: Store, Collection
- Value types (base class: CValue): CRegister, CString, CList, CMap

Note: The 'C' prefix might stand for "collaborative", or "concurrent", or
"conflict-free", or "CRDT", or something else entirely. It distinguishes our
types from JS built-in types such as String.

## Store

Methods:

    openStore(addr) => {err, Store}
    s.getCollection('name') => {err, Collection}
    s.getOrCreateCollection('name') => {err, Collection, created}
    s.destroyCollection('name') => {err}

Note: For now, there's only one collection, named ''. This collection exists by
default and cannot be destroyed.

## Collection

Collection is similar to CMap, but is not live or observable. (If it were, we'd
quickly hit performance problems.)

TODO:
- Add scan and query methods
- Add API for batches

Methods:

    c.get('key') => {err, CValue}
    c.getOrCreate('key', dtype) => {err, CValue, created}
    // Value must be a native JS type, and will be converted to a Register.
    c.put('key', value) => {err}
    c.delete('key') => {err}

## CValue (base class)

    // Returns a native JS type that represents the value of this CValue.
    m.value() => Object

Note: For now, CValue instances are always "live" and observable: they always
reflect the latest state from the server, and they emit events whenever their
value changes.

## CRegister

Methods:

    r.get() => Object
    r.set(value)

Events:

    Set: {isLocal, value}

## CString

Methods:

    s.getText() => String
    s.getSelectionRange() => []int  // [start, end]
    s.replaceText(pos, len, value)
    s.setSelectionRange(start, end)

Events:

    ReplaceText: {isLocal, pos, len, value}
    SetSelectionRange: {isLocal, start, end}

## CList

TODO: Specify methods and events.

## CMap

Similar to Collection.

    m.get('key') => {err, CValue}
    m.getOrCreate('key', dtype) => {err, CValue, created}
    // Value must be a native JS type, and will be converted to a Register.
    m.put('key', value) => {err}
    m.delete('key') => {err}

TODO: Specify events.

# Server API

## Client-server protocol

Client talks to server over WebSocket, initialized by openStore. For this
initial prototype, communication is message-based (like Mojo), not call-based
(like Vanadium). This works okay for now because WebSocket delivers messages in
order and we panic on any error.

Client-to-server messages:
- Subscribe: {}
- Unsubscribe: {}
- Patch: {key, dtype, valueDelta}

Server-to-client messages:
- SubscribeResponse: {agentId, clientId}
- Value: {key, dtype, value}
- Patch: {agentId, isLocal, key, dtype, valueDelta}

Semantics: When client sends Subscribe, server replies with SubscribeResponse,
followed by Values for every object, followed by a never-ending stream of
Patches for every object. Invariant: Server will never send Patch before Value
for a given key.

## Server-server protocol

Servers talk over WebSocket. As with client-server, server-server communication
is message-based for now. Every server has an agent id, initialized to a random
number the first time the server is started.

Initiator-to-responder messages:
- Subscribe: {agentId, versionVector}
- Unsubscribe: {agentId}

Responder-to-initiator messages:
- SubscribeResponse: {agentId}
- Patch: {agentId, agentSeq, key, dtype, valueDelta}

Semantics: When initiator sends Subscribe, responder replies with
SubscribeResponse followed by a never-ending stream of Patches for every object.
Stream starting point is determined by initiator's version vector.

TODO: Start by sending Value record, as in client-server protocol? CRDTs that
support state merging would deal with this just fine.

# Client implementation

Similar to existing implementation. Watch stream includes updates for all
objects, regardless of whether this client is interested in them. Client is
responsible for maintaining state for all objects. (In the future, clients
should be able watch select keys.)

# Server implementation

- Built around an oplog (of patches) plus a key-value store (of values)
- Oplog records contain sequence number, key, and value delta
- Physical oplog is partitioned by originating agent id; each oplog record
  contains a sequence number tracking its position in this particular agent's
  logical oplog

Note: We store sequence numbers in oplog records so that operations get executed
in the same partial order at every agent, thus satisfying causality. (Some
CRDTs, including Logoot but not Logoot-Undo, require this property.)

## Op handling

Ops are processed atomically, as follows:
1. Perform any sanity checks, e.g. dtype checks
1. Apply patch to value in state store
1. Write patch to oplog
1. Use Sync.Cond.Broadcast to notify watching goroutines

## Client-server impl

Goroutine per client:
- Upon connection, send all object values, then start streaming oplog
- Apply ops as they arrive

## Server-server (sync) impl

Each agent maintains a version vector describing its current knowledge: map of
agent id to sequence number.

Goroutine per peer:
- Upon connection, send current version vector
- Upon receiving peer's version vector, start streaming oplog
- Apply ops (if needed) as they arrive
