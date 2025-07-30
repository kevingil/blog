package session

import (
	"context"
	"fmt"
	"time"
)

type Session struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Cost             float64 `json:"cost"`
	CompletionTokens int64   `json:"completion_tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	SummaryMessageID string  `json:"summary_message_id"`
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
}

type Service interface {
	Get(ctx context.Context, id string) (Session, error)
	Save(ctx context.Context, session Session) (Session, error)
	Create(ctx context.Context, title string) (Session, error)
	List(ctx context.Context) ([]Session, error)
	Delete(ctx context.Context, id string) error
}

// InMemorySessionService is a simple in-memory implementation for the blog agent
type InMemorySessionService struct {
	sessions map[string]Session
}

func NewInMemorySessionService() Service {
	return &InMemorySessionService{
		sessions: make(map[string]Session),
	}
}

func (s *InMemorySessionService) Get(ctx context.Context, id string) (Session, error) {
	session, exists := s.sessions[id]
	if !exists {
		// Create a new session if it doesn't exist
		return s.Create(ctx, "Untitled Session")
	}
	return session, nil
}

func (s *InMemorySessionService) Save(ctx context.Context, session Session) (Session, error) {
	session.UpdatedAt = time.Now().Unix()
	s.sessions[session.ID] = session
	return session, nil
}

func (s *InMemorySessionService) Create(ctx context.Context, title string) (Session, error) {
	session := Session{
		ID:        generateSessionID(),
		Title:     title,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	s.sessions[session.ID] = session
	return session, nil
}

func (s *InMemorySessionService) List(ctx context.Context) ([]Session, error) {
	sessions := make([]Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (s *InMemorySessionService) Delete(ctx context.Context, id string) error {
	delete(s.sessions, id)
	return nil
}

func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
