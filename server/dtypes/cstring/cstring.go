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
	"strconv"
	"strings"
	"time"

	"github.com/asadovsky/cdb/server/common"
)

func assert(b bool, v ...interface{}) {
	if !b {
		panic(fmt.Sprint(v...))
	}
}

// Id is a Logoot identifier.
type Id struct {
	Pos     uint32
	AgentId int
}

// Pid is a Logoot position identifier.
type Pid struct {
	Ids []Id
	Seq int // logical clock value for the last Id's agent
}

// Less returns true iff p is less than other.
func (p *Pid) Less(other *Pid) bool {
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
func (p *Pid) Equal(other *Pid) bool {
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

// Encode encodes this Pid.
func (p *Pid) Encode() string {
	idStrs := make([]string, len(p.Ids))
	for i, v := range p.Ids {
		idStrs[i] = fmt.Sprintf("%d.%d", v.Pos, v.AgentId)
	}
	return strings.Join(idStrs, ":") + "~" + strconv.Itoa(p.Seq)
}

// DecodePid decodes the given string into a Pid.
func DecodePid(s string) (*Pid, error) {
	idsAndSeq := strings.Split(s, "~")
	if len(idsAndSeq) != 2 {
		return nil, fmt.Errorf("invalid pid: %s", s)
	}
	seq, err := strconv.Atoi(idsAndSeq[1])
	if err != nil {
		return nil, fmt.Errorf("invalid seq: %s", s)
	}
	idStrs := strings.Split(idsAndSeq[0], ":")
	ids := make([]Id, len(idStrs))
	for i, v := range idStrs {
		parts := strings.Split(v, ".")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid id: %s", v)
		}
		pos, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid pos: %s", v)
		}
		agentId, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid agentId: %s", v)
		}
		ids[i] = Id{Pos: uint32(pos), AgentId: agentId}
	}
	return &Pid{Ids: ids, Seq: seq}, nil
}

// Op is an operation.
type Op interface {
	// Encode encodes this Op.
	Encode() string
}

// ClientInsert represents an atom insertion from a client.
type ClientInsert struct {
	PrevPid *Pid   // nil means start of document
	NextPid *Pid   // nil means end of document
	Value   string // may contain multiple characters
}

// Encode encodes this Op.
func (op *ClientInsert) Encode() string {
	var prevPidStr, nextPidStr string
	if op.PrevPid != nil {
		prevPidStr = op.PrevPid.Encode()
	}
	if op.NextPid != nil {
		nextPidStr = op.NextPid.Encode()
	}
	return fmt.Sprintf("ci,%s,%s,%s", prevPidStr, nextPidStr, op.Value)
}

// Insert represents an atom insertion.
type Insert struct {
	Pid   *Pid
	Value string
}

// Encode encodes this Op.
func (op *Insert) Encode() string {
	return fmt.Sprintf("i,%s,%s", op.Pid.Encode(), op.Value)
}

// Delete represents an atom deletion. Pid is the position identifier of the
// deleted atom. Note, Delete cannot be defined as a [start, end] range because
// it must commute with Insert.
// TODO: To reduce client->server message size, maybe add a ClientDelete
// operation defined as a [start, end] range.
type Delete struct {
	Pid *Pid
}

// Encode encodes this Op.
func (op *Delete) Encode() string {
	return fmt.Sprintf("d,%s", op.Pid.Encode())
}

func newParseError(s string) error {
	return fmt.Errorf("failed to parse op: %s", s)
}

// DecodeOp decodes the given string into an Op.
func DecodeOp(s string) (Op, error) {
	parts := strings.SplitN(s, ",", 2)
	t := parts[0]
	switch t {
	case "ci":
		parts = strings.SplitN(s, ",", 4)
		if len(parts) < 4 {
			return nil, newParseError(s)
		}
		var prevPid, nextPid *Pid
		var err error
		if parts[1] != "" {
			if prevPid, err = DecodePid(parts[1]); err != nil {
				return nil, newParseError(s)
			}
		}
		if parts[2] != "" {
			if nextPid, err = DecodePid(parts[2]); err != nil {
				return nil, newParseError(s)
			}
		}
		if err != nil {
			return nil, newParseError(s)
		}
		return &ClientInsert{prevPid, nextPid, parts[3]}, nil
	case "i":
		parts = strings.SplitN(s, ",", 3)
		if len(parts) < 3 {
			return nil, newParseError(s)
		}
		pid, err := DecodePid(parts[1])
		if err != nil {
			return nil, newParseError(s)
		}
		return &Insert{pid, parts[2]}, nil
	case "d":
		parts = strings.SplitN(s, ",", 2)
		if len(parts) < 2 {
			return nil, newParseError(s)
		}
		pid, err := DecodePid(parts[1])
		if err != nil {
			return nil, newParseError(s)
		}
		return &Delete{pid}, nil
	default:
		return nil, fmt.Errorf("unknown op type: %s", t)
	}
}

func encodePatch(ops []Op) (string, error) {
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

func decodePatch(s string) ([]Op, error) {
	strs := []string{}
	if err := json.Unmarshal([]byte(s), &strs); err != nil {
		return nil, err
	}
	ops := make([]Op, len(strs))
	for i, v := range strs {
		op, err := DecodeOp(v)
		if err != nil {
			return nil, err
		}
		ops[i] = op
	}
	return ops, nil
}

// Exported (along with Pid and Id) to support CString.Encode.
type Atom struct {
	Pid *Pid
	// TODO: Switch to rune?
	Value string
}

// CString is a CRDT string (Logoot).
type CString struct {
	atoms []Atom
	value string
}

// New returns a new CString.
func New() *CString {
	return &CString{}
}

// Value implements CValue.Value.
// FIXME: Drop this method, here and elsewhere?
func (s *CString) Value() interface{} {
	return s.value
}

// Encode implements CValue.Encode.
func (s *CString) Encode() (string, error) {
	atoms := s.atoms
	if atoms == nil {
		atoms = []Atom{}
	}
	buf, err := json.Marshal(atoms)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// ApplyPatch implements CValue.ApplyPatch.
func (s *CString) ApplyPatch(agentId int, vec *common.VersionVector, t time.Time, patch string) (string, error) {
	agentSeq, ok := vec.Get(agentId)
	if !ok {
		return "", fmt.Errorf("unknown agent: %d", agentId)
	}
	ops, err := decodePatch(patch)
	if err != nil {
		return "", err
	}
	appliedOps := make([]Op, 0, len(ops))
	gotClientInsert := false
	for _, op := range ops {
		switch v := op.(type) {
		case *ClientInsert:
			if gotClientInsert {
				return "", errors.New("cannot apply multiple ClientInsert ops")
			}
			gotClientInsert = true
			// TODO: Smarter pid allocation.
			prevPid := v.PrevPid
			for j := 0; j < len(v.Value); j++ {
				x := &Insert{genPid(agentId, agentSeq, prevPid, v.NextPid), string(v.Value[j])}
				s.applyInsertText(x)
				appliedOps = append(appliedOps, x)
				prevPid = x.Pid
			}
		case *Insert:
			s.applyInsertText(v)
			appliedOps = append(appliedOps, op)
		case *Delete:
			s.applyDeleteText(v)
			appliedOps = append(appliedOps, op)
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
func genIds(agentId int, prev, next []Id) []Id {
	if len(prev) == 0 {
		prev = []Id{{Pos: 0, AgentId: agentId}}
	}
	if len(next) == 0 {
		next = []Id{{Pos: math.MaxUint32, AgentId: agentId}}
	}
	if prev[0].Pos+1 < next[0].Pos {
		return []Id{{Pos: randUint32Between(prev[0].Pos, next[0].Pos), AgentId: agentId}}
	}
	return append([]Id{prev[0]}, genIds(agentId, prev[1:], next[1:])...)
}

func genPid(agentId, agentSeq int, prev, next *Pid) *Pid {
	prevIds, nextIds := []Id{}, []Id{}
	if prev != nil {
		prevIds = prev.Ids
	}
	if next != nil {
		nextIds = next.Ids
	}
	return &Pid{Ids: genIds(agentId, prevIds, nextIds), Seq: agentSeq}
}

func (s *CString) applyInsertText(op *Insert) {
	a := s.atoms
	p := s.search(op.Pid)
	if p != len(a) && a[p].Pid.Equal(op.Pid) {
		assert(a[p].Value == op.Value)
		return
	}
	// https://github.com/golang/go/wiki/SliceTricks
	a = append(a, Atom{})
	copy(a[p+1:], a[p:])
	a[p] = Atom{Pid: op.Pid, Value: op.Value}
	s.atoms = a
	s.value = s.value[:p] + op.Value + s.value[p:]
}

func (s *CString) applyDeleteText(op *Delete) {
	a := s.atoms
	p := s.search(op.Pid)
	if p == len(a) || !a[p].Pid.Equal(op.Pid) {
		return
	}
	// https://github.com/golang/go/wiki/SliceTricks
	a, a[len(a)-1] = append(a[:p], a[p+1:]...), Atom{}
	s.atoms = a
	s.value = s.value[:p] + s.value[p+1:]
}

// search returns the position of the first atom with pid >= the given pid.
func (s *CString) search(pid *Pid) int {
	return sort.Search(len(s.atoms), func(i int) bool { return !s.atoms[i].Pid.Less(pid) })
}
