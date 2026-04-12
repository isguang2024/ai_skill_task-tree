package tasktree

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"
)

const (
	mcpSessionHeader      = "Mcp-Session-Id"
	mcpLastEventIDHeader  = "Last-Event-ID"
	mcpSessionHistorySize = 256
)

type mcpSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*mcpSession
}

type mcpSession struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	NextSeq   int64
	Events    []mcpStreamEvent
}

type mcpStreamEvent struct {
	ID   string
	Name string
	Data []byte
}

func newMCPSessionStore() *mcpSessionStore {
	return &mcpSessionStore{sessions: map[string]*mcpSession{}}
}

func (s *mcpSessionStore) create() *mcpSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := &mcpSession{
		ID:        newID("mcps"),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.sessions[session.ID] = session
	return cloneMCPSession(session)
}

func (s *mcpSessionStore) get(sessionID string) (*mcpSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	session.UpdatedAt = time.Now().UTC()
	return cloneMCPSession(session), true
}

func (s *mcpSessionStore) delete(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[sessionID]; !ok {
		return false
	}
	delete(s.sessions, sessionID)
	return true
}

func (s *mcpSessionStore) appendResponse(sessionID string, resp *rpcResponse) (mcpStreamEvent, bool, error) {
	if sessionID == "" || resp == nil {
		return mcpStreamEvent{}, false, nil
	}
	body, err := json.Marshal(resp)
	if err != nil {
		return mcpStreamEvent{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return mcpStreamEvent{}, false, nil
	}
	session.NextSeq++
	event := mcpStreamEvent{
		ID:   strconv.FormatInt(session.NextSeq, 10),
		Name: "message",
		Data: body,
	}
	session.Events = append(session.Events, event)
	if len(session.Events) > mcpSessionHistorySize {
		session.Events = append([]mcpStreamEvent(nil), session.Events[len(session.Events)-mcpSessionHistorySize:]...)
	}
	session.UpdatedAt = time.Now().UTC()
	return event, true, nil
}

func (s *mcpSessionStore) eventsSince(sessionID, lastEventID string) ([]mcpStreamEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	session.UpdatedAt = time.Now().UTC()
	if lastEventID == "" {
		return append([]mcpStreamEvent(nil), session.Events...), true
	}
	lastSeq, err := strconv.ParseInt(lastEventID, 10, 64)
	if err != nil {
		return append([]mcpStreamEvent(nil), session.Events...), true
	}
	out := make([]mcpStreamEvent, 0, len(session.Events))
	for _, event := range session.Events {
		seq, err := strconv.ParseInt(event.ID, 10, 64)
		if err != nil || seq > lastSeq {
			out = append(out, event)
		}
	}
	return out, true
}

func cloneMCPSession(src *mcpSession) *mcpSession {
	if src == nil {
		return nil
	}
	out := *src
	if len(src.Events) > 0 {
		out.Events = append([]mcpStreamEvent(nil), src.Events...)
	}
	return &out
}

