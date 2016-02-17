package store

import (
	"math"
	"sync"

	"github.com/asadovsky/cdb/server/common"
)

type Log struct {
	cond *sync.Cond
	// Maps agent id to patches created by that agent.
	m            map[int][]*PatchEnvelope
	head         *common.VersionVector
	nextLocalSeq int
}

// Head returns a new version vector representing current knowledge. cond.L must
// be held.
func (l *Log) Head() *common.VersionVector {
	return l.head.Copy()
}

// Wait blocks until the log has patches beyond the given version vector. cond.L
// must not be held.
func (l *Log) Wait(vec *common.VersionVector) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()
	for l.head.Leq(vec) {
		l.cond.Wait()
	}
}

// push appends the given patch (from the given agent id) to the log and returns
// the local sequence number for the written log record. cond.L must be held.
func (l *Log) push(agentId int, key, dtype string, patch string) (int, error) {
	localSeq := l.nextLocalSeq
	l.nextLocalSeq++
	s := append(l.m[agentId], &PatchEnvelope{
		LocalSeq: localSeq,
		Key:      key,
		DType:    dtype,
		Patch:    patch,
	})
	l.m[agentId] = s
	l.head.Put(agentId, len(s))
	l.cond.Broadcast()
	return localSeq, nil
}

////////////////////////////////////////////////////////////
// LogIterator

type LogIterator struct {
	l   *Log
	vec *common.VersionVector
	// Agent id and sequence number for staged patch.
	agentId  int
	agentSeq int
}

// NewIterator returns an iterator for patches beyond the given version vector.
// Iteration order matches log order. cond.L must be held during calls to
// Advance, but need not be held at other times.
func (l *Log) NewIterator(vec *common.VersionVector) *LogIterator {
	return &LogIterator{l: l, vec: vec, agentId: -1, agentSeq: -1}
}

// Advance advances the iterator, staging the next patch. Must be called to
// stage the first value. Assumes cond.L is held.
func (it *LogIterator) Advance() bool {
	minLocalSeq, advAgentId, advAgentSeq := math.MaxInt32, -1, -1
	for agentId, patches := range it.l.m {
		agentSeq, ok := it.vec.Get(agentId)
		if ok {
			agentSeq++
		}
		if agentSeq < len(patches) && patches[agentSeq].LocalSeq < minLocalSeq {
			minLocalSeq, advAgentId, advAgentSeq = patches[agentSeq].LocalSeq, agentId, agentSeq
		}
	}
	if advAgentId == -1 {
		return false
	}
	it.vec.Put(advAgentId, advAgentSeq)
	it.agentId, it.agentSeq = advAgentId, advAgentSeq
	return true
}

// Value returns the current patch.
func (it *LogIterator) Patch() *PatchEnvelope {
	return it.l.m[it.agentId][it.agentSeq]
}

// AgentId returns the agent id that produced the current patch.
func (it *LogIterator) AgentId() int {
	return it.agentId
}

// VersionVector returns a copy of the current version vector, representing
// which patches the client has already seen.
func (it *LogIterator) VersionVector() *common.VersionVector {
	return it.vec.Copy()
}

// Err returns a non-nil error iff the iterator encountered an error.
func (it *LogIterator) Err() error {
	return nil
}
