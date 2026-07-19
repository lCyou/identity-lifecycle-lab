package identity

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to State
		want     bool
	}{
		{StateEnrolled, StateIssued, true},
		{StateIssued, StateActive, true},
		{StateActive, StateSuspended, true},
		{StateSuspended, StateActive, true},
		{StateActive, StateRevoked, true},
		{StateSuspended, StateRevoked, true},
		{StateRevoked, StateArchived, true},
		{StateRevoked, StateActive, false}, // 失効からの直接復活は不可
		{StateArchived, StateActive, false},
		{StateEnrolled, StateActive, false},
		{StateEnrolled, StateRevoked, false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.want {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}
