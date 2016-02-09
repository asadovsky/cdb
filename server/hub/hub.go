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

	"github.com/asadovsky/cdb/server/store"
	"github.com/asadovsky/cdb/server/types"
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
	deviceId     int
	mu           sync.Mutex // protects the fields below
	nextClientId int
	store        *store.Store
}

func newHub() *hub {
	// TODO: Check whether deviceId exists in store.
	return &hub{
		deviceId: rand.Int(),
	}
}

type stream struct {
	h    *hub
	conn *websocket.Conn
	// Populated if connection is from a client.
	clientId *int
	// Populated if connection is from a server (a peer).
	deviceId      *int
	versionVector map[int]int
}

func (s *stream) processSubscribeC2S(msg *SubscribeC2S) error {
	s.h.mu.Lock()
	if s.clientId != nil || s.deviceId != nil {
		return errAlreadyInitialized
	}
	*s.clientId = s.h.nextClientId
	s.h.nextClientId++
	// While holding mu, snapshot current values and version vector.
	valueMsgs := []ValueS2C{}
	it := s.h.store.NewIterator()
	for it.Advance() {
		valueMsgs = append(valueMsgs, ValueS2C{
			Type:     "ValueS2C",
			Key:      it.Key(),
			DataType: it.Value().DataType,
			Value:    it.Value().Value.Encode(),
		})
	}
	if err := it.Err(); err != nil {
		return err
	}
	versionVector := s.h.store.Head()
	s.h.mu.Unlock()
	res := &SubscribeResponseS2C{
		Type:     "SubscribeResponseS2C",
		DeviceId: s.h.deviceId,
		ClientId: *s.clientId,
	}
	if err := s.conn.WriteJSON(res); err != nil {
		return err
	}
	for valueMsg := range valueMsgs {
		if err := s.conn.WriteJSON(valueMsg); err != nil {
			return err
		}
	}
	// FIXME: Start streaming patches.
	return nil
}

func (s *stream) processSubscribeI2R(msg *SubscribeI2R) error {
	s.h.mu.Lock()
	if s.clientId != nil || s.deviceId != nil {
		return errAlreadyInitialized
	}
	*s.deviceId = msg.DeviceId
	s.h.mu.Unlock()
	res := &SubscribeResponseR2I{
		Type:     "SubscribeResponseR2I",
		DeviceId: s.h.deviceId,
	}
	if err := s.conn.WriteJSON(res); err != nil {
		return err
	}
	// TODO: Start streaming patches.
	return nil
}

func (s *stream) processPatchC2S(msg *PatchC2S) error {
	s.h.mu.Lock()
	defer s.h.mu.Unlock()
	if s.clientId != nil || s.deviceId != nil {
		return errors.New("not initialized")
	}
	patch, err := types.DecodePatch(msg.DataType, msg.Patch)
	if err != nil {
		return err
	}
	// Update store and log.
	return s.h.store.ApplyPatch(s.h.deviceId, msg.Key, msg.DataType, patch)
}

func (h *hub) handleConn(w http.ResponseWriter, r *http.Request) {
	const bufSize = 1024
	conn, err := websocket.Upgrade(w, r, nil, bufSize, bufSize)
	ok(err)
	s := &stream{h: h, conn: conn}
	eof, done := make(chan struct{}), make(chan struct{})

	go func() {
		for {
			_, buf, err := conn.ReadMessage()
			if ce, ok := err.(*websocket.CloseError); ok && (ce.Code == websocket.CloseNormalClosure || ce.Code == websocket.CloseGoingAway) {
				close(eof)
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
