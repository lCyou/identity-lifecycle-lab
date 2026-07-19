package identity_test

import (
	"testing"

	"github.com/lCyou/identity-lifecycle-lab/internal/identity"
)

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to identity.State
		want     bool
	}{
		{identity.StateEnrolled, identity.StateIssued, true},
		{identity.StateIssued, identity.StateActive, true},
		{identity.StateActive, identity.StateSuspended, true},
		{identity.StateSuspended, identity.StateActive, true},
		{identity.StateActive, identity.StateRevoked, true},
		{identity.StateSuspended, identity.StateRevoked, true},
		{identity.StateRevoked, identity.StateArchived, true},
		{identity.StateRevoked, identity.StateActive, false}, // 失効からの直接復活は不可
		{identity.StateArchived, identity.StateActive, false},
		{identity.StateEnrolled, identity.StateActive, false},
		{identity.StateEnrolled, identity.StateRevoked, false},
	}
	for _, c := range cases {
		if got := identity.CanTransition(c.from, c.to); got != c.want {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}
