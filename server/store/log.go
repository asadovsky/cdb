package store

import (
	"math"
	"sync"
	"time"

	"github.com/asadovsky/cdb/server/types"
)

type log struct {
	cond *sync.Cond
	// Maps device id to patches created by that device.
	m            map[int][]*PatchEnvelope
	head         map[int]int
	nextLocalSeq int
}

// head returns a new version vector representing current knowledge.
// cond.L must be held.
func (l *log) getHead() map[int]int {
	res := map[int]int{}
	for k, v := range l.head {
		res[k] = v
	}
	return res
}

// push writes the given patch (from the given device id) to the log.
// cond.L must be held.
func (l *log) push(deviceId int, key string, dataType string, patch types.Patch) error {
	s := append(l.m[deviceId], &PatchEnvelope{
		LocalSeq: l.nextLocalSeq,
		Time:     time.Now(),
		Key:      key,
		DataType: dataType,
		Patch:    patch,
	})
	l.nextLocalSeq++
	l.m[deviceId] = s
	l.head[deviceId] = len(s)
	l.cond.Broadcast()
	return nil
}

// wait blocks until the log has patches beyond the given version vector.
// cond.L must not be held.
func (l *log) wait(vec map[int]int) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()
	for !leq(l.head, vec) {
		l.cond.Wait()
	}
}

////////////////////////////////////////////////////////////
// LogIterator

type LogIterator struct {
	l   *log
	vec map[int]int
	// Device id and sequence number for staged patch.
	deviceId  int
	deviceSeq int
}

// NewIterator returns an iterator for patches beyond the given version vector.
// Iteration order matches log order. cond.L must be held during calls to
// NewIterator and Advance, but need not be held at other times.
func (l *log) NewIterator(vec map[int]int) *LogIterator {
	return &LogIterator{l: l, vec: vec, deviceId: -1, deviceSeq: -1}
}

// Advance advances the iterator, staging the next patch. Must be called to
// stage the first value. Assumes cond.L is held.
func (it *LogIterator) Advance() bool {
	minLocalSeq, advDeviceId, advDeviceSeq := math.MaxInt32, -1, -1
	for deviceId, patches := range it.l.m {
		deviceSeq, ok := it.vec[deviceId]
		if ok {
			deviceSeq++
		}
		if deviceSeq < len(patches) && patches[deviceSeq].LocalSeq < minLocalSeq {
			minLocalSeq, advDeviceId, advDeviceSeq = patches[deviceSeq].LocalSeq, deviceId, deviceSeq
		}
	}
	if advDeviceId == -1 {
		return false
	}
	it.vec[advDeviceId] = advDeviceSeq
	it.deviceId, it.deviceSeq = advDeviceId, advDeviceSeq
	return true
}

// Value returns the current patch.
func (it *LogIterator) Value() *PatchEnvelope {
	return it.l.m[it.deviceId][it.deviceSeq]
}

// Err returns a non-nil error iff the iterator encountered an error.
func (it *LogIterator) Err() error {
	return nil
}

////////////////////////////////////////////////////////////
// Version vector helpers

// leq returns true iff a[x] <= b[x] for all x in a.
func leq(a, b map[int]int) bool {
	for k, va := range a {
		if vb, ok := b[k]; !ok || vb < va {
			return false
		}
	}
	return true
}
