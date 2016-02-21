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

func isCloseError(err error) bool {
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure)
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
			go h.recvPatches(peerAddr)
		}
	}
	return h
}

// recvPatches requests patches from the given peer. If the peer is available,
// they will stream back patches. We forget about this peer once the connection
// is closed.
func (h *hub) recvPatches(peerAddr string) {
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
		if isCloseError(err) {
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

type stream struct {
	h           *hub
	conn        *websocket.Conn
	initialized bool
	// Queue of written local sequence numbers, populated if connection is from a
	// client. Used to determine whether a log record originated from this
	// particular client.
	// TODO: Use a linked list or somesuch.
	localSeqs []uint32
	// Populated if connection is from a peer.
	agentId uint32
	vec     *common.VersionVector
}

func (s *stream) processSubscribeC2S(msg *SubscribeC2S) error {
	s.h.mu.Lock()
	if s.initialized {
		return errAlreadyInitialized
	}
	s.initialized = true
	// While holding mu, snapshot current values and version vector.
	valueMsgs := []ValueS2C{}
	it := s.h.store.NewIterator()
	for it.Advance() {
		valueStr, err := it.Value().Value.Encode()
		if err != nil {
			return err
		}
		valueMsgs = append(valueMsgs, ValueS2C{
			Type:  "ValueS2C",
			Key:   it.Key(),
			DType: it.Value().DType,
			Value: valueStr,
		})
	}
	if err := it.Err(); err != nil {
		return err
	}
	vec := s.h.store.Log.Head()
	s.h.mu.Unlock()
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
	go s.sendPatches(vec, true)
	return nil
}

func (s *stream) processSubscribeI2R(msg *SubscribeI2R) error {
	s.h.mu.Lock()
	if s.initialized {
		return errAlreadyInitialized
	}
	s.initialized = true
	s.agentId = msg.AgentId
	s.h.mu.Unlock()
	go s.h.recvPatches(msg.Addr)
	go s.sendPatches(msg.VersionVector, false)
	return nil
}

func (s *stream) processPatchC2S(msg *PatchC2S) error {
	s.h.mu.Lock()
	defer s.h.mu.Unlock()
	if !s.initialized {
		return errors.New("not initialized")
	}
	// Update store and log.
	localSeq, err := s.h.store.ApplyClientPatch(s.h.agentId, msg.Key, msg.DType, msg.Patch)
	if err != nil {
		return err
	}
	s.localSeqs = append(s.localSeqs, localSeq)
	return nil
}

// sendPatches streams patches to the client or peer server until the connection
// is closed.
func (s *stream) sendPatches(vec *common.VersionVector, isClient bool) {
	for {
		s.h.store.Log.Wait(vec)
		it := s.h.store.Log.NewIterator(vec)
		for {
			s.h.mu.Lock()
			advanced := it.Advance()
			s.h.mu.Unlock()
			if !advanced {
				break
			}
			patch := it.Patch()
			var err error
			// TODO: If the patch had no effect on the value, perhaps we should
			// somehow avoid broadcasting it to subscribers.
			if isClient {
				isLocal := false
				if len(s.localSeqs) > 0 && s.localSeqs[0] == patch.LocalSeq {
					isLocal = true
					s.localSeqs = s.localSeqs[1:]
				}
				err = s.conn.WriteJSON(&PatchS2C{
					Type:    "PatchS2C",
					AgentId: it.AgentId(),
					IsLocal: isLocal,
					Key:     patch.Key,
					DType:   patch.DType,
					Patch:   patch.Patch,
				})
			} else {
				// TODO: Update our notion of the peer's knowledge based on patches we
				// receive from them.
				if s.agentId != it.AgentId() {
					err = s.conn.WriteJSON(&PatchR2I{
						Type:     "PatchR2I",
						AgentId:  it.AgentId(),
						AgentSeq: it.AgentSeq(),
						Key:      patch.Key,
						DType:    patch.DType,
						Patch:    patch.Patch,
					})
				}
			}
			// TODO: If a client goes away, the ReadMessage error is CloseGoingAway,
			// and the error here is ErrCloseSent. However, if a peer server goes
			// away, the ReadMessage error is CloseAbnormalClosure, and the error here
			// is not ErrCloseSent. Is this an unavoidable race between a peer server
			// going away and us sending them a patch?
			if err == websocket.ErrCloseSent || err != nil && strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			ok(err)
		}
		ok(it.Err())
	}
}

func (h *hub) handleConn(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, nil, 0, 0)
	ok(err)
	s := &stream{h: h, conn: conn}

	for {
		_, buf, err := conn.ReadMessage()
		if isCloseError(err) {
			log.Printf("conn closed: %v", err)
			conn.Close()
			return
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
