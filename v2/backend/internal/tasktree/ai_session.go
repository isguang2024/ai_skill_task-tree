package tasktree

import (
	"sync"
	"time"
)

// aiMessage mirrors Anthropic Messages API message structure.
// Content can be a plain string or []aiContentBlock.
type aiMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type aiContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   any    `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

type aiSession struct {
	// Anthropic Messages API history
	Messages []aiMessage
	// OpenAI Responses API input array (flat, contains messages + function_call + function_call_output items)
	OAIInput []map[string]any
	LastUsed time.Time
}

type aiSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*aiSession
}

func newAISessionStore() *aiSessionStore {
	return &aiSessionStore{sessions: make(map[string]*aiSession)}
}

func (s *aiSessionStore) getOrCreate(id string) *aiSession {
	if sess, ok := s.sessions[id]; ok {
		sess.LastUsed = time.Now()
		return sess
	}
	sess := &aiSession{LastUsed: time.Now()}
	s.sessions[id] = sess
	return sess
}

// Anthropic history

func (s *aiSessionStore) get(id string) []aiMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.getOrCreate(id)
	out := make([]aiMessage, len(sess.Messages))
	copy(out, sess.Messages)
	return out
}

func (s *aiSessionStore) set(id string, msgs []aiMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.getOrCreate(id)
	sess.Messages = msgs
}

// OpenAI Responses API history

func (s *aiSessionStore) getOAI(id string) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.getOrCreate(id)
	out := make([]map[string]any, len(sess.OAIInput))
	copy(out, sess.OAIInput)
	return out
}

func (s *aiSessionStore) setOAI(id string, input []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.getOrCreate(id)
	sess.OAIInput = input
}

func (s *aiSessionStore) clear(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

