package store

import (
	"math"
	"sync"
	"time"

	"github.com/asadovsky/cdb/server/dtypes"
)

type Log struct {
	cond *sync.Cond
	// Maps device id to patches created by that device.
	m            map[int][]*PatchEnvelope
	head         map[int]int
	nextLocalSeq int
}

// Head returns a new version vector representing current knowledge.
// cond.L must be held.
func (l *Log) Head() map[int]int {
	return copyVec(l.head)
}

// Wait blocks until the log has patches beyond the given version vector.
// cond.L must not be held.
func (l *Log) Wait(vec map[int]int) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()
	for !leq(l.head, vec) {
		l.cond.Wait()
	}
}

// push appends the given patch (from the given device id) to the log and
// returns the local sequence number for the written log record.
// cond.L must be held.
func (l *Log) push(deviceId int, key, dtype string, patch dtypes.Patch) (int, error) {
	localSeq := l.nextLocalSeq
	l.nextLocalSeq++
	s := append(l.m[deviceId], &PatchEnvelope{
		LocalSeq: localSeq,
		Time:     time.Now(),
		Key:      key,
		DType:    dtype,
		Patch:    patch,
	})
	l.m[deviceId] = s
	l.head[deviceId] = len(s)
	l.cond.Broadcast()
	return localSeq, nil
}

////////////////////////////////////////////////////////////
// LogIterator

type LogIterator struct {
	l   *Log
	vec map[int]int
	// Device id and sequence number for staged patch.
	deviceId  int
	deviceSeq int
}

// NewIterator returns an iterator for patches beyond the given version vector.
// Iteration order matches log order. cond.L must be held during calls to
// Advance, but need not be held at other times.
func (l *Log) NewIterator(vec map[int]int) *LogIterator {
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
func (it *LogIterator) Patch() *PatchEnvelope {
	return it.l.m[it.deviceId][it.deviceSeq]
}

// DeviceId returns the device id that produced the current patch.
func (it *LogIterator) DeviceId() int {
	return it.deviceId
}

// VersionVector returns a copy of the current version vector, representing
// which patches the client has already seen.
func (it *LogIterator) VersionVector() map[int]int {
	return copyVec(it.vec)
}

// Err returns a non-nil error iff the iterator encountered an error.
func (it *LogIterator) Err() error {
	return nil
}

////////////////////////////////////////////////////////////
// Version vector helpers

// copyVec returns a copy of the given version vector.
func copyVec(vec map[int]int) map[int]int {
	res := map[int]int{}
	for k, v := range vec {
		res[k] = v
	}
	return res
}

// leq returns true iff a[x] <= b[x] for all x in a.
func leq(a, b map[int]int) bool {
	for k, va := range a {
		if vb, ok := b[k]; !ok || vb < va {
			return false
		}
	}
	return true
}
