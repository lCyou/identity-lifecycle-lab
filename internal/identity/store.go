package identity

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrEntityNotFound    = errors.New("entity not found")
	ErrInvalidTransition = errors.New("invalid state transition")
)

// Store はエンティティと遷移履歴を保持するインメモリストア。
// 学習用途のため永続化は行わない。
type Store struct {
	mu          sync.Mutex
	entities    map[string]*Entity
	transitions map[string][]TransitionRecord
}

func NewStore() *Store {
	return &Store{
		entities:    make(map[string]*Entity),
		transitions: make(map[string][]TransitionRecord),
	}
}

// CreateEntity は Enrolled 状態でエンティティを新規作成する。
func (s *Store) CreateEntity(name string) *Entity {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	e := &Entity{
		ID:        newID("ent"),
		Name:      name,
		State:     StateEnrolled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.entities[e.ID] = e
	s.transitions[e.ID] = []TransitionRecord{
		{
			ID:         newID("tr"),
			EntityID:   e.ID,
			FromState:  "",
			ToState:    StateEnrolled,
			Actor:      "system",
			Reason:     "initial enrolment",
			OccurredAt: now,
		},
	}

	cp := *e
	return &cp
}

func (s *Store) GetEntity(id string) (*Entity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.entities[id]
	if !ok {
		return nil, ErrEntityNotFound
	}
	cp := *e
	return &cp, nil
}

func (s *Store) ListEntities() []*Entity {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := make([]*Entity, 0, len(s.entities))
	for _, e := range s.entities {
		cp := *e
		list = append(list, &cp)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	return list
}

// Transition は許可された遷移のみを適用し、監査ログに記録する。
func (s *Store) Transition(entityID string, to State, actor, reason string) (*Entity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.entities[entityID]
	if !ok {
		return nil, ErrEntityNotFound
	}
	if !CanTransition(e.State, to) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, e.State, to)
	}

	now := time.Now().UTC()
	from := e.State
	e.State = to
	e.UpdatedAt = now

	s.transitions[entityID] = append(s.transitions[entityID], TransitionRecord{
		ID:         newID("tr"),
		EntityID:   entityID,
		FromState:  from,
		ToState:    to,
		Actor:      actor,
		Reason:     reason,
		OccurredAt: now,
	})

	cp := *e
	return &cp, nil
}

func (s *Store) History(entityID string) ([]TransitionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[entityID]; !ok {
		return nil, ErrEntityNotFound
	}
	hist := s.transitions[entityID]
	cp := make([]TransitionRecord, len(hist))
	copy(cp, hist)
	return cp, nil
}
