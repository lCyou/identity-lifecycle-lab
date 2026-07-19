package identity_test

import (
	"errors"
	"testing"

	"github.com/lCyou/identity-lifecycle-lab/internal/identity"
)

func TestStoreTransitionLifecycle(t *testing.T) {
	s := identity.NewStore()
	e := s.CreateEntity("alice")
	if e.State != identity.StateEnrolled {
		t.Fatalf("want enrolled, got %s", e.State)
	}

	steps := []identity.State{
		identity.StateIssued,
		identity.StateActive,
		identity.StateSuspended,
		identity.StateActive,
		identity.StateRevoked,
		identity.StateArchived,
	}
	for _, to := range steps {
		updated, err := s.Transition(e.ID, to, "admin", "test")
		if err != nil {
			t.Fatalf("transition to %s failed: %v", to, err)
		}
		if updated.State != to {
			t.Fatalf("want state %s, got %s", to, updated.State)
		}
	}

	if _, err := s.Transition(e.ID, identity.StateActive, "admin", "attempt reactivation from archived"); err == nil {
		t.Fatal("expected error transitioning from archived, got nil")
	}
}

func TestStoreRejectsSkippedTransition(t *testing.T) {
	s := identity.NewStore()
	e := s.CreateEntity("bob")

	if _, err := s.Transition(e.ID, identity.StateRevoked, "admin", "skip ahead"); err == nil {
		t.Fatal("expected error for enrolled -> revoked, got nil")
	}
}

func TestHistoryRecordsActorAndReason(t *testing.T) {
	s := identity.NewStore()
	e := s.CreateEntity("carol")
	if _, err := s.Transition(e.ID, identity.StateIssued, "registrar", "credentials issued"); err != nil {
		t.Fatal(err)
	}

	hist, err := s.History(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 { // 初期登録 + issued
		t.Fatalf("want 2 history entries, got %d", len(hist))
	}
	last := hist[len(hist)-1]
	if last.Actor != "registrar" || last.Reason != "credentials issued" {
		t.Fatalf("unexpected history entry: %+v", last)
	}
}

func TestGetAndHistoryNotFound(t *testing.T) {
	s := identity.NewStore()
	if _, err := s.GetEntity("missing"); !errors.Is(err, identity.ErrEntityNotFound) {
		t.Fatalf("want ErrEntityNotFound, got %v", err)
	}
	if _, err := s.History("missing"); !errors.Is(err, identity.ErrEntityNotFound) {
		t.Fatalf("want ErrEntityNotFound, got %v", err)
	}
}
