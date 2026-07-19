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
