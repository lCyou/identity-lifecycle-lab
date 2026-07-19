package identity

import "time"

// Entity はライフサイクル管理の対象となるアイデンティティ。
type Entity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	State     State     `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

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
