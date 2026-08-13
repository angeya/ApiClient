package request

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// WSManager manages active WebSocket connections.
type WSManager struct {
	connections map[string]*websocket.Conn
	mu          sync.RWMutex
	ctxGetter   func() context.Context
}

// NewWSManager creates a WebSocket manager.
func NewWSManager(ctxGetter func() context.Context) *WSManager {
	return &WSManager{
		connections: make(map[string]*websocket.Conn),
		ctxGetter:   ctxGetter,
	}
}

// Connect establishes a WebSocket connection.
func (m *WSManager) Connect(id string, url string, headers http.Header) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	}
	conn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return err
	}
	m.mu.Lock()
	old, exists := m.connections[id]
	m.connections[id] = conn
	m.mu.Unlock()
	if exists && old != nil {
		_ = old.Close()
	}
	go m.readLoop(id, conn)
	return nil
}

func (m *WSManager) readLoop(id string, conn *websocket.Conn) {
	defer func() {
		_ = conn.Close()
		m.mu.Lock()
		if m.connections[id] == conn {
			delete(m.connections, id)
		}
		m.mu.Unlock()
	}()
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			m.emit("websocket:error", id, err.Error())
			return
		}
		if msgType == websocket.TextMessage {
			m.emit("websocket:message", id, string(data))
		} else {
			m.emit("websocket:message", id, "[binary message]")
		}
	}
}

func (m *WSManager) emit(event string, id string, data string) {
	ctx := m.ctxGetter()
	if ctx != nil {
		runtime.EventsEmit(ctx, event, id, data)
	}
}

// Send sends a text message over an existing connection.
func (m *WSManager) Send(id string, message string) error {
	m.mu.RLock()
	conn, ok := m.connections[id]
	m.mu.RUnlock()
	if !ok {
		return errors.New("connection not found")
	}
	return conn.WriteMessage(websocket.TextMessage, []byte(message))
}

// Disconnect closes a WebSocket connection.
func (m *WSManager) Disconnect(id string) error {
	m.mu.Lock()
	conn, ok := m.connections[id]
	if ok {
		delete(m.connections, id)
	}
	m.mu.Unlock()
	if !ok {
		return errors.New("connection not found")
	}
	return conn.Close()
}
