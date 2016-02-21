// Package hub implements the CDB server.
package hub

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/asadovsky/gosh"
	"github.com/gorilla/websocket"

	"github.com/asadovsky/cdb/server/common"
	"github.com/asadovsky/cdb/server/store"
)

var (
	errAlreadyInitialized = errors.New("already initialized")
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func ok(err error, v ...interface{}) {
	if err != nil {
		panic(fmt.Sprintf("%v: %s", err, fmt.Sprint(v...)))
	}
}

func assert(b bool, v ...interface{}) {
	if !b {
		panic(fmt.Sprint(v...))
	}
}

func jsonMarshal(v interface{}) []byte {
	buf, err := json.Marshal(v)
	ok(err)
	return buf
}

func isReadFromClosedConnError(err error) bool {
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure)
}

func isWriteToClosedConnError(err error) bool {
	// TODO: If a client goes away, the read error is CloseGoingAway and the write
	// error is ErrCloseSent. However, if a peer goes away, the read error is
	// CloseAbnormalClosure and the write error is not ErrCloseSent. Is this an
	// unavoidable race between a peer going away and us sending them data?
	return err == websocket.ErrCloseSent || err != nil && strings.Contains(err.Error(), "use of closed network connection")
}

type hub struct {
	agentId uint32
	addr    string
	mu      sync.Mutex // protects the fields below
	store   *store.Store
	peers   map[string]bool // set of active peers, keyed by addr
}

func newHub(addr string, peerAddrs []string) *hub {
	// TODO: Attempt to read agent id from persistent storage.
	h := &hub{
		agentId: uint32(rand.Int31()),
		addr:    addr,
		peers:   make(map[string]bool),
	}
	h.store = store.OpenStore(&h.mu)
	log.Printf("started agent %d", h.agentId)
	// Start streaming updates from peers.
	for _, peerAddr := range peerAddrs {
		if peerAddr != "" {
			go h.requestPatchesFromPeer(peerAddr)
		}
	}
	return h
}

// requestPatchesFromPeer requests patches from the given peer. If the peer is
// available, they will reply with a never-ending stream of patches.
func (h *hub) requestPatchesFromPeer(peerAddr string) {
	h.mu.Lock()
	if h.peers[peerAddr] {
		// We're already streaming patches from this peer.
		h.mu.Unlock()
		return
	}
	h.peers[peerAddr] = true
	vec := h.store.Log.Head()
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.peers, peerAddr)
		h.mu.Unlock()
	}()
	// Dial peer.
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+peerAddr, nil)
	if err != nil {
		log.Printf("peer %s: dial failed: %v", peerAddr, err)
		return
	}
	log.Printf("peer %s: established connection", peerAddr)
	// Send SubscribeI2R message.
	ok(conn.WriteJSON(&SubscribeI2R{
		Type:          "SubscribeI2R",
		AgentId:       h.agentId,
		Addr:          h.addr,
		VersionVector: vec,
	}))
	// Process patches streamed from peer.
	for {
		_, buf, err := conn.ReadMessage()
		if isReadFromClosedConnError(err) {
			log.Printf("peer %s: conn closed: %v", peerAddr, err)
			conn.Close()
			return
		}
		var msg PatchR2I
		ok(json.Unmarshal(buf, &msg))
		assert(msg.Type == "PatchR2I", msg)
		// Update store and log.
		h.mu.Lock()
		err = h.store.ApplyServerPatch(msg.AgentId, msg.AgentSeq, msg.Key, msg.DType, msg.Patch)
		h.mu.Unlock()
		ok(err)
	}
}

// forEachLogEntry iterates over log entries beyond the given version vector.
func (h *hub) forEachLogEntry(vec *common.VersionVector, handleLogEntry func(*store.LogIterator) error) error {
	for {
		h.store.Log.Wait(vec)
		it := h.store.Log.NewIterator(vec)
		for {
			h.mu.Lock()
			advanced := it.Advance()
			h.mu.Unlock()
			if !advanced {
				break
			}
			if err := handleLogEntry(it); err != nil {
				return err
			}
			vec = it.VersionVector()
		}
		if err := it.Err(); err != nil {
			return err
		}
	}
}

type stream struct {
	h    *hub
	conn *websocket.Conn
	mu   sync.Mutex

	// Populated if connection is from a client.
	gotSubscribeC2S bool
	// Queue of written local sequence numbers. Used to determine whether a log
	// record originated from this particular client.
	// TODO: Use a linked list or somesuch.
	localSeqs []uint32

	// Populated if connection is from a peer.
	gotSubscribeI2R bool
	agentId         uint32
}

func (s *stream) snapshot() ([]ValueS2C, *common.VersionVector, error) {
	s.h.mu.Lock()
	defer s.h.mu.Unlock()
	valueMsgs := []ValueS2C{}
	it := s.h.store.NewIterator()
	for it.Advance() {
		valueStr, err := it.Value().Value.Encode()
		if err != nil {
			return nil, nil, err
		}
		valueMsgs = append(valueMsgs, ValueS2C{
			Type:  "ValueS2C",
			Key:   it.Key(),
			DType: it.Value().DType,
			Value: valueStr,
		})
	}
	if err := it.Err(); err != nil {
		return nil, nil, err
	}
	return valueMsgs, s.h.store.Log.Head(), nil
}

func (s *stream) processSubscribeC2S(msg *SubscribeC2S) error {
	s.mu.Lock()
	if s.gotSubscribeC2S || s.gotSubscribeI2R {
		return errAlreadyInitialized
	}
	s.gotSubscribeC2S = true
	s.mu.Unlock()
	valueMsgs, vec, err := s.snapshot()
	if err != nil {
		return err
	}
	for _, valueMsg := range valueMsgs {
		if err := s.conn.WriteJSON(valueMsg); err != nil {
			return err
		}
	}
	if err := s.conn.WriteJSON(&ValuesDoneS2C{
		Type: "ValuesDoneS2C",
	}); err != nil {
		return err
	}
	go func() {
		ok(s.h.forEachLogEntry(vec, func(it *store.LogIterator) error {
			patch := it.Patch()
			isLocal := false
			s.mu.Lock()
			if len(s.localSeqs) > 0 && s.localSeqs[0] == patch.LocalSeq {
				isLocal = true
				s.localSeqs = s.localSeqs[1:]
			}
			s.mu.Unlock()
			// TODO: If the patch had no effect on the value, perhaps we should
			// somehow avoid broadcasting it to subscribers.
			err := s.conn.WriteJSON(&PatchS2C{
				Type:    "PatchS2C",
				AgentId: it.AgentId(),
				IsLocal: isLocal,
				Key:     patch.Key,
				DType:   patch.DType,
				Patch:   patch.Patch,
			})
			if isWriteToClosedConnError(err) {
				return nil
			}
			return err
		}))
	}()
	return nil
}

func (s *stream) processSubscribeI2R(msg *SubscribeI2R) error {
	s.mu.Lock()
	if s.gotSubscribeC2S || s.gotSubscribeI2R {
		return errAlreadyInitialized
	}
	s.gotSubscribeI2R = true
	s.agentId = msg.AgentId
	s.mu.Unlock()
	go func() {
		ok(s.h.forEachLogEntry(msg.VersionVector, func(it *store.LogIterator) error {
			// TODO: Update our notion of the peer's knowledge based on patches we
			// receive from them.
			if s.agentId == it.AgentId() {
				return nil
			}
			patch := it.Patch()
			err := s.conn.WriteJSON(&PatchR2I{
				Type:     "PatchR2I",
				AgentId:  it.AgentId(),
				AgentSeq: it.AgentSeq(),
				Key:      patch.Key,
				DType:    patch.DType,
				Patch:    patch.Patch,
			})
			if isWriteToClosedConnError(err) {
				return nil
			}
			return err
		}))
	}()
	// Turn around and request patches from this peer.
	go s.h.requestPatchesFromPeer(msg.Addr)
	return nil
}

func (s *stream) processPatchC2S(msg *PatchC2S) error {
	s.mu.Lock()
	if !s.gotSubscribeC2S {
		s.mu.Unlock()
		return errors.New("did not get SubscribeC2S message")
	}
	s.mu.Unlock()
	// Update store and log.
	s.h.mu.Lock()
	localSeq, err := s.h.store.ApplyClientPatch(s.h.agentId, msg.Key, msg.DType, msg.Patch)
	s.h.mu.Unlock()
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.localSeqs = append(s.localSeqs, localSeq)
	s.mu.Unlock()
	return nil
}

func (h *hub) handleConn(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, nil, 0, 0)
	ok(err)
	s := &stream{h: h, conn: conn}

	for {
		_, buf, err := conn.ReadMessage()
		if isReadFromClosedConnError(err) {
			log.Printf("conn closed: %v", err)
			break
		}
		ok(err)
		// TODO: Avoid decoding multiple times.
		var mt MsgType
		ok(json.Unmarshal(buf, &mt))
		switch mt.Type {
		case "SubscribeC2S":
			var msg SubscribeC2S
			ok(json.Unmarshal(buf, &msg))
			ok(s.processSubscribeC2S(&msg))
		case "SubscribeI2R":
			var msg SubscribeI2R
			ok(json.Unmarshal(buf, &msg))
			ok(s.processSubscribeI2R(&msg))
		case "PatchC2S":
			var msg PatchC2S
			ok(json.Unmarshal(buf, &msg))
			ok(s.processPatchC2S(&msg))
		default:
			panic(fmt.Errorf("unknown message type: %s", mt.Type))
		}
	}

	conn.Close()
}

func Serve(addr string, peerAddrs []string) error {
	h := newHub(addr, peerAddrs)
	http.HandleFunc("/", h.handleConn)
	go func() {
		time.Sleep(100 * time.Millisecond)
		gosh.SendVars(map[string]string{"ready": ""})
	}()
	return http.ListenAndServe(addr, nil)
}
