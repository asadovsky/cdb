// Package hub implements the CDB server.
package hub

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
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

// TODO: Hold list of peer addresses to talk to, and periodically attempt to
// connect to each peer.
type hub struct {
	agentId      uint32
	mu           sync.Mutex // protects the fields below
	nextClientId uint32
	store        *store.Store
}

func newHub() *hub {
	// TODO: Attempt to read agent id from persistent storage.
	h := &hub{agentId: uint32(rand.Int31())}
	h.store = store.OpenStore(&h.mu)
	return h
}

type stream struct {
	h           *hub
	conn        *websocket.Conn
	initialized bool
	// Populated if connection is from a client.
	clientId uint32
	// Queue of written local sequence numbers. Used to determine whether a log
	// record originated from this client.
	// TODO: Use a linked list or somesuch.
	localSeqs []uint32
	// Populated if connection is from a server (a peer).
	agentId uint32
	vec     *common.VersionVector
}

func (s *stream) processSubscribeC2S(msg *SubscribeC2S) error {
	s.h.mu.Lock()
	if s.initialized {
		return errAlreadyInitialized
	}
	s.initialized = true
	s.clientId = s.h.nextClientId
	s.h.nextClientId++
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
	if err := s.conn.WriteJSON(&SubscribeResponseS2C{
		Type:     "SubscribeResponseS2C",
		AgentId:  s.h.agentId,
		ClientId: s.clientId,
	}); err != nil {
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
	go s.streamPatches(vec)
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
	if err := s.conn.WriteJSON(&SubscribeResponseR2I{
		Type:    "SubscribeResponseR2I",
		AgentId: s.h.agentId,
	}); err != nil {
		return err
	}
	// TODO: Start streaming patches.
	return nil
}

func (s *stream) processPatchC2S(msg *PatchC2S) error {
	s.h.mu.Lock()
	defer s.h.mu.Unlock()
	if !s.initialized {
		return errors.New("not initialized")
	}
	// Update store and log.
	// TODO: If the patch had no effect on the value, perhaps we should avoid
	// broadcasting it to subscribers.
	localSeq, err := s.h.store.ApplyPatch(s.h.agentId, msg.Key, msg.DType, msg.Patch, true)
	if err != nil {
		return err
	}
	s.localSeqs = append(s.localSeqs, localSeq)
	return nil
}

// streamPatches streams patches to the client until the connection is closed.
func (s *stream) streamPatches(vec *common.VersionVector) {
	go func() {
		closed := false
		for !closed {
			s.h.store.Log.Wait(vec)
			it := s.h.store.Log.NewIterator(vec)
			for {
				s.h.mu.Lock()
				advanced := it.Advance()
				s.h.mu.Unlock()
				if !advanced {
					break
				}
				vec = it.VersionVector()
				patch := it.Patch()
				isLocal := false
				if len(s.localSeqs) > 0 && s.localSeqs[0] == patch.LocalSeq {
					isLocal = true
					s.localSeqs = s.localSeqs[1:]
				}
				err := s.conn.WriteJSON(&PatchS2C{
					Type:    "PatchS2C",
					AgentId: it.AgentId(),
					IsLocal: isLocal,
					Key:     patch.Key,
					DType:   patch.DType,
					Patch:   patch.Patch,
				})
				if err == websocket.ErrCloseSent {
					closed = true
					break
				}
				ok(err)
			}
			ok(it.Err())
		}
	}()
}

func (h *hub) handleConn(w http.ResponseWriter, r *http.Request) {
	const bufSize = 1024
	conn, err := websocket.Upgrade(w, r, nil, bufSize, bufSize)
	ok(err)
	s := &stream{h: h, conn: conn}
	done := make(chan struct{})

	go func() {
		for {
			_, buf, err := conn.ReadMessage()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				close(done)
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
	}()

	<-done
	conn.Close()
}

func Serve(addr string) error {
	h := newHub()
	http.HandleFunc("/", h.handleConn)
	go func() {
		time.Sleep(100 * time.Millisecond)
		gosh.SendVars(map[string]string{"ready": ""})
	}()
	return http.ListenAndServe(addr, nil)
}
