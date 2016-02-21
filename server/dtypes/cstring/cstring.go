// Package cstring defines CString, an implementation of the Logoot CRDT,
// representing a sequence of characters.
// https://hal.inria.fr/inria-00432368/document
package cstring

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/asadovsky/cdb/server/common"
	"github.com/asadovsky/cdb/server/dtypes/cvalue"
)

func assert(b bool, v ...interface{}) {
	if !b {
		panic(fmt.Sprint(v...))
	}
}

// id is a Logoot identifier.
type id struct {
	Pos     uint32
	AgentId uint32
}

// pid is a Logoot position identifier.
type pid struct {
	Ids []id
	Seq uint32 // logical clock value for the last id's agent
}

// Less returns true iff p is less than other.
func (p *pid) Less(other *pid) bool {
	for i, v := range p.Ids {
		if i == len(other.Ids) {
			return false
		}
		vo := other.Ids[i]
		if v.Pos != vo.Pos {
			return v.Pos < vo.Pos
		} else if v.AgentId != vo.AgentId {
			return v.AgentId < vo.AgentId
		}
	}
	if len(p.Ids) == len(other.Ids) {
		return p.Seq < other.Seq
	}
	return true
}

// Equal returns true iff p is equal to other.
func (p *pid) Equal(other *pid) bool {
	if len(p.Ids) != len(other.Ids) || p.Seq != other.Seq {
		return false
	}
	for i, v := range p.Ids {
		vo := other.Ids[i]
		if v.Pos != vo.Pos || v.AgentId != vo.AgentId {
			return false
		}
	}
	return true
}

// Encode encodes this pid.
func (p *pid) Encode() string {
	idStrs := make([]string, len(p.Ids))
	for i, v := range p.Ids {
		idStrs[i] = fmt.Sprintf("%d.%d", v.Pos, v.AgentId)
	}
	return strings.Join(idStrs, ":") + "~" + common.Itoa(p.Seq)
}

// decodePid decodes the given string into a pid.
func decodePid(s string) (*pid, error) {
	idsAndSeq := strings.Split(s, "~")
	if len(idsAndSeq) != 2 {
		return nil, fmt.Errorf("invalid pid: %s", s)
	}
	seq, err := common.Atoi(idsAndSeq[1])
	if err != nil {
		return nil, fmt.Errorf("invalid seq: %s", s)
	}
	idStrs := strings.Split(idsAndSeq[0], ":")
	ids := make([]id, len(idStrs))
	for i, v := range idStrs {
		parts := strings.Split(v, ".")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid id: %s", v)
		}
		pos, err := common.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid pos: %s", v)
		}
		agentId, err := common.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid agentId: %s", v)
		}
		ids[i] = id{Pos: uint32(pos), AgentId: agentId}
	}
	return &pid{Ids: ids, Seq: seq}, nil
}

// op is an operation.
type op interface {
	// Encode encodes this op.
	Encode() string
}

// clientInsert represents an atom insertion from a client.
type clientInsert struct {
	PrevPid *pid   // nil means start of document
	NextPid *pid   // nil means end of document
	Value   string // may contain multiple characters
}

// Encode encodes this op.
func (op *clientInsert) Encode() string {
	var prevPidStr, nextPidStr string
	if op.PrevPid != nil {
		prevPidStr = op.PrevPid.Encode()
	}
	if op.NextPid != nil {
		nextPidStr = op.NextPid.Encode()
	}
	return fmt.Sprintf("ci,%s,%s,%s", prevPidStr, nextPidStr, op.Value)
}

// insert represents an atom insertion.
type insert struct {
	Pid   *pid
	Value string
}

// Encode encodes this op.
func (op *insert) Encode() string {
	return fmt.Sprintf("i,%s,%s", op.Pid.Encode(), op.Value)
}

// delete represents an atom deletion. Pid is the position identifier of the
// deleted atom. Note, delete cannot be defined as a [start, end] range because
// it must commute with insert.
// TODO: To reduce client->server message size, maybe add a clientDelete
// operation defined as a [start, end] range.
type delete struct {
	Pid *pid
}

// Encode encodes this op.
func (op *delete) Encode() string {
	return fmt.Sprintf("d,%s", op.Pid.Encode())
}

func newParseError(s string) error {
	return fmt.Errorf("failed to parse op: %s", s)
}

// decodeOp decodes the given string into an op.
func decodeOp(s string) (op, error) {
	parts := strings.SplitN(s, ",", 2)
	t := parts[0]
	switch t {
	case "ci":
		parts = strings.SplitN(s, ",", 4)
		if len(parts) < 4 {
			return nil, newParseError(s)
		}
		var prevPid, nextPid *pid
		var err error
		if parts[1] != "" {
			if prevPid, err = decodePid(parts[1]); err != nil {
				return nil, newParseError(s)
			}
		}
		if parts[2] != "" {
			if nextPid, err = decodePid(parts[2]); err != nil {
				return nil, newParseError(s)
			}
		}
		if err != nil {
			return nil, newParseError(s)
		}
		return &clientInsert{prevPid, nextPid, parts[3]}, nil
	case "i":
		parts = strings.SplitN(s, ",", 3)
		if len(parts) < 3 {
			return nil, newParseError(s)
		}
		pid, err := decodePid(parts[1])
		if err != nil {
			return nil, newParseError(s)
		}
		return &insert{pid, parts[2]}, nil
	case "d":
		parts = strings.SplitN(s, ",", 2)
		if len(parts) < 2 {
			return nil, newParseError(s)
		}
		pid, err := decodePid(parts[1])
		if err != nil {
			return nil, newParseError(s)
		}
		return &delete{pid}, nil
	default:
		return nil, fmt.Errorf("unknown op type: %s", t)
	}
}

func encodePatch(ops []op) (string, error) {
	strs := make([]string, len(ops))
	for i, v := range ops {
		strs[i] = v.Encode()
	}
	buf, err := json.Marshal(strs)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func decodePatch(s string) ([]op, error) {
	strs := []string{}
	if err := json.Unmarshal([]byte(s), &strs); err != nil {
		return nil, err
	}
	ops := make([]op, len(strs))
	for i, v := range strs {
		op, err := decodeOp(v)
		if err != nil {
			return nil, err
		}
		ops[i] = op
	}
	return ops, nil
}

// atom is an atom in a Logoot document.
type atom struct {
	Pid *pid
	// TODO: Switch to rune?
	Value string
}

var _ json.Marshaler = (*atom)(nil)

// MarshalJSON marshals to JSON.
func (a *atom) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Pid   string
		Value string
	}{
		Pid:   a.Pid.Encode(),
		Value: a.Value,
	})
}

// CString is a CRDT string (Logoot).
type CString struct {
	atoms []atom
	text  string
}

// New returns a new CString.
func New() *CString {
	return &CString{}
}

// DType implements CValue.DType.
func (s *CString) DType() string {
	return cvalue.DTypeCString
}

// Encode implements CValue.Encode.
func (s *CString) Encode() (string, error) {
	buf, err := json.Marshal(s.atoms)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// Decode decodes the given value into a CString.
func Decode(s string) (*CString, error) {
	return nil, errors.New("not implemented")
}

// ApplyServerPatch implements CValue.ApplyServerPatch.
func (s *CString) ApplyServerPatch(patch string) error {
	ops, err := decodePatch(patch)
	if err != nil {
		return err
	}
	for _, op := range ops {
		switch v := op.(type) {
		case *insert:
			s.applyInsertText(v)
		case *delete:
			s.applyDeleteText(v)
		default:
			return fmt.Errorf("invalid op type: %T", v)
		}
	}
	return nil
}

// ApplyClientPatch implements CValue.ApplyClientPatch.
func (s *CString) ApplyClientPatch(agentId uint32, vec *common.VersionVector, _ time.Time, patch string) (string, error) {
	agentSeq := vec.Get(agentId)
	// Sanity check.
	if agentSeq == 0 {
		return "", fmt.Errorf("unknown agent: %d", agentId)
	}
	ops, err := decodePatch(patch)
	if err != nil {
		return "", err
	}
	appliedOps := make([]op, 0, len(ops))
	gotClientInsert := false
	for _, op := range ops {
		switch v := op.(type) {
		case *clientInsert:
			if gotClientInsert {
				return "", errors.New("cannot apply multiple clientInsert ops")
			}
			gotClientInsert = true
			// TODO: Smarter pid allocation.
			prevPid := v.PrevPid
			for j := 0; j < len(v.Value); j++ {
				x := &insert{genPid(agentId, agentSeq, prevPid, v.NextPid), string(v.Value[j])}
				s.applyInsertText(x)
				appliedOps = append(appliedOps, x)
				prevPid = x.Pid
			}
		case *insert:
			s.applyInsertText(v)
			appliedOps = append(appliedOps, op)
		case *delete:
			s.applyDeleteText(v)
			appliedOps = append(appliedOps, op)
		default:
			return "", fmt.Errorf("unknown op type: %T", v)
		}
	}
	return encodePatch(appliedOps)
}

func randUint32Between(prev, next uint32) uint32 {
	return prev + 1 + uint32(rand.Int63n(int64(next-prev-1)))
}

// TODO: Smarter pid allocation, e.g. LSEQ. Also, maybe do something to ensure
// that concurrent multi-atom insertions from different agents do not get
// interleaved.
func genIds(agentId uint32, prev, next []id) []id {
	if len(prev) == 0 {
		prev = []id{{Pos: 0, AgentId: agentId}}
	}
	if len(next) == 0 {
		next = []id{{Pos: math.MaxUint32, AgentId: agentId}}
	}
	if prev[0].Pos+1 < next[0].Pos {
		return []id{{Pos: randUint32Between(prev[0].Pos, next[0].Pos), AgentId: agentId}}
	}
	return append([]id{prev[0]}, genIds(agentId, prev[1:], next[1:])...)
}

func genPid(agentId, agentSeq uint32, prev, next *pid) *pid {
	prevIds, nextIds := []id{}, []id{}
	if prev != nil {
		prevIds = prev.Ids
	}
	if next != nil {
		nextIds = next.Ids
	}
	return &pid{Ids: genIds(agentId, prevIds, nextIds), Seq: agentSeq}
}

func (s *CString) applyInsertText(op *insert) {
	a := s.atoms
	p := s.search(op.Pid)
	if p != len(a) && a[p].Pid.Equal(op.Pid) {
		assert(a[p].Value == op.Value)
		return
	}
	// https://github.com/golang/go/wiki/SliceTricks
	a = append(a, atom{})
	copy(a[p+1:], a[p:])
	a[p] = atom{Pid: op.Pid, Value: op.Value}
	s.atoms = a
	s.text = s.text[:p] + op.Value + s.text[p:]
}

func (s *CString) applyDeleteText(op *delete) {
	a := s.atoms
	p := s.search(op.Pid)
	if p == len(a) || !a[p].Pid.Equal(op.Pid) {
		return
	}
	// https://github.com/golang/go/wiki/SliceTricks
	a, a[len(a)-1] = append(a[:p], a[p+1:]...), atom{}
	s.atoms = a
	s.text = s.text[:p] + s.text[p+1:]
}

// search returns the position of the first atom with pid >= the given pid.
func (s *CString) search(pid *pid) int {
	return sort.Search(len(s.atoms), func(i int) bool { return !s.atoms[i].Pid.Less(pid) })
}
