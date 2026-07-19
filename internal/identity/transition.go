package identity

import "time"

// TransitionRecord は状態遷移の監査ログ1件分。
// 「誰が(Actor)・いつ(OccurredAt)・なぜ(Reason)」を記録する。
type TransitionRecord struct {
	ID         string    `json:"id"`
	EntityID   string    `json:"entity_id"`
	FromState  State     `json:"from_state"`
	ToState    State     `json:"to_state"`
	Actor      string    `json:"actor"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}
