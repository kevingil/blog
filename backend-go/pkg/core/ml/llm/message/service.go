package message

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error)
	Update(ctx context.Context, message Message) error
	List(ctx context.Context, sessionID string) ([]Message, error)
	Get(ctx context.Context, messageID string) (Message, error)
	Delete(ctx context.Context, messageID string) error
}

// InMemoryMessageService is a simple in-memory implementation for the blog agent
type InMemoryMessageService struct {
	messages        map[string]Message
	sessionMessages map[string][]string // sessionID -> messageIDs
}

func NewInMemoryMessageService() Service {
	return &InMemoryMessageService{
		messages:        make(map[string]Message),
		sessionMessages: make(map[string][]string),
	}
}

func (s *InMemoryMessageService) Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error) {
	messageID := uuid.New().String()
	now := time.Now().Unix()

	message := Message{
		ID:        messageID,
		SessionID: sessionID,
		Role:      params.Role,
		Parts:     params.Parts,
		Model:     params.Model,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.messages[messageID] = message

	// Add to session messages
	if _, exists := s.sessionMessages[sessionID]; !exists {
		s.sessionMessages[sessionID] = make([]string, 0)
	}
	s.sessionMessages[sessionID] = append(s.sessionMessages[sessionID], messageID)

	return message, nil
}

func (s *InMemoryMessageService) Update(ctx context.Context, message Message) error {
	message.UpdatedAt = time.Now().Unix()
	s.messages[message.ID] = message
	return nil
}

func (s *InMemoryMessageService) List(ctx context.Context, sessionID string) ([]Message, error) {
	messageIDs, exists := s.sessionMessages[sessionID]
	if !exists {
		return []Message{}, nil
	}

	messages := make([]Message, 0, len(messageIDs))
	for _, id := range messageIDs {
		if message, exists := s.messages[id]; exists {
			messages = append(messages, message)
		}
	}

	return messages, nil
}

func (s *InMemoryMessageService) Get(ctx context.Context, messageID string) (Message, error) {
	message, exists := s.messages[messageID]
	if !exists {
		return Message{}, nil
	}
	return message, nil
}

func (s *InMemoryMessageService) Delete(ctx context.Context, messageID string) error {
	// Get the message to find its session
	message, exists := s.messages[messageID]
	if !exists {
		return nil
	}

	// Remove from messages
	delete(s.messages, messageID)

	// Remove from session messages
	if messageIDs, exists := s.sessionMessages[message.SessionID]; exists {
		for i, id := range messageIDs {
			if id == messageID {
				s.sessionMessages[message.SessionID] = append(messageIDs[:i], messageIDs[i+1:]...)
				break
			}
		}
	}

	return nil
}
