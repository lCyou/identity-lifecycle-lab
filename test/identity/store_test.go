package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lCyou/identity-lifecycle-lab/internal/identity"
	"github.com/lCyou/identity-lifecycle-lab/test/dbtest"
)

func TestStoreTransitionLifecycle(t *testing.T) {
	ctx := context.Background()
	s := identity.NewStore(dbtest.Open(t))

	e, err := s.CreateEntity(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
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
		updated, err := s.Transition(ctx, e.ID, to, "admin", "test")
		if err != nil {
			t.Fatalf("transition to %s failed: %v", to, err)
		}
		if updated.State != to {
			t.Fatalf("want state %s, got %s", to, updated.State)
		}
	}

	if _, err := s.Transition(ctx, e.ID, identity.StateActive, "admin", "attempt reactivation from archived"); err == nil {
		t.Fatal("expected error transitioning from archived, got nil")
	}
}

func TestStoreRejectsSkippedTransition(t *testing.T) {
	ctx := context.Background()
	s := identity.NewStore(dbtest.Open(t))

	e, err := s.CreateEntity(ctx, "bob")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.Transition(ctx, e.ID, identity.StateRevoked, "admin", "skip ahead"); err == nil {
		t.Fatal("expected error for enrolled -> revoked, got nil")
	}
}

func TestHistoryRecordsActorAndReason(t *testing.T) {
	ctx := context.Background()
	s := identity.NewStore(dbtest.Open(t))

	e, err := s.CreateEntity(ctx, "carol")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Transition(ctx, e.ID, identity.StateIssued, "registrar", "credentials issued"); err != nil {
		t.Fatal(err)
	}

	hist, err := s.History(ctx, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 { // 初期登録(DBトリガー) + issued(DBトリガー)
		t.Fatalf("want 2 history entries, got %d: %+v", len(hist), hist)
	}
	first := hist[0]
	if first.Actor != "system" || first.Reason != "initial enrolment" || first.ToState != identity.StateEnrolled {
		t.Fatalf("unexpected initial history entry: %+v", first)
	}
	last := hist[len(hist)-1]
	if last.Actor != "registrar" || last.Reason != "credentials issued" || last.FromState != identity.StateEnrolled {
		t.Fatalf("unexpected history entry: %+v", last)
	}
}

func TestGetAndHistoryNotFound(t *testing.T) {
	ctx := context.Background()
	s := identity.NewStore(dbtest.Open(t))

	if _, err := s.GetEntity(ctx, "missing"); !errors.Is(err, identity.ErrEntityNotFound) {
		t.Fatalf("want ErrEntityNotFound, got %v", err)
	}
	if _, err := s.History(ctx, "missing"); !errors.Is(err, identity.ErrEntityNotFound) {
		t.Fatalf("want ErrEntityNotFound, got %v", err)
	}
}
