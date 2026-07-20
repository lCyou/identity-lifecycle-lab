package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrEntityNotFound    = errors.New("entity not found")
	ErrInvalidTransition = errors.New("invalid state transition")
)

// Store はPostgreSQL上のentities/transitionsテーブルに対する読み書きを行う。
//
// 遷移の合法性チェック（CanTransition）はここGoアプリ側で行うが、
// 監査ログ(transitionsテーブルへの記録)はアプリからは一切書き込まず、
// entitiesテーブルへのINSERT/UPDATEに張られたDBトリガーに完全に委ねる。
// そのためTransitionは actor/reason を同一トランザクション内で
// set_config('app.actor', ...) 経由でトリガーに引き渡す。
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func setActorReason(ctx context.Context, tx *sql.Tx, actor, reason string) error {
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.actor', $1, true)`, actor); err != nil {
		return fmt.Errorf("set app.actor: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.reason', $1, true)`, reason); err != nil {
		return fmt.Errorf("set app.reason: %w", err)
	}
	return nil
}

// CreateEntity は Enrolled 状態でエンティティを新規作成する。
// 初回のenrolment記録はentitiesへのINSERTトリガーが行う。
func (s *Store) CreateEntity(ctx context.Context, name string) (*Entity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := setActorReason(ctx, tx, "system", "initial enrolment"); err != nil {
		return nil, err
	}

	e := &Entity{ID: newID("ent"), Name: name, State: StateEnrolled}
	row := tx.QueryRowContext(ctx, `
		INSERT INTO entities (id, name, state)
		VALUES ($1, $2, $3)
		RETURNING id, name, state, created_at, updated_at
	`, e.ID, e.Name, e.State)
	if err := row.Scan(&e.ID, &e.Name, &e.State, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Store) GetEntity(ctx context.Context, id string) (*Entity, error) {
	e := &Entity{}
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, state, created_at, updated_at FROM entities WHERE id = $1
	`, id)
	if err := row.Scan(&e.ID, &e.Name, &e.State, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEntityNotFound
		}
		return nil, err
	}
	return e, nil
}

func (s *Store) ListEntities(ctx context.Context) ([]*Entity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, state, created_at, updated_at FROM entities ORDER BY created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*Entity, 0)
	for rows.Next() {
		e := &Entity{}
		if err := rows.Scan(&e.ID, &e.Name, &e.State, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// Transition は許可された遷移のみを適用する。現在の状態は行ロック(FOR UPDATE)
// を取って読み、同時遷移リクエストが競合しないようにする。
// 監査ログへの記録はentitiesへのUPDATEトリガーが行う。
func (s *Store) Transition(ctx context.Context, entityID string, to State, actor, reason string) (*Entity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var current State
	row := tx.QueryRowContext(ctx, `SELECT state FROM entities WHERE id = $1 FOR UPDATE`, entityID)
	if err := row.Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEntityNotFound
		}
		return nil, err
	}

	if !CanTransition(current, to) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current, to)
	}

	if err := setActorReason(ctx, tx, actor, reason); err != nil {
		return nil, err
	}

	e := &Entity{}
	updateRow := tx.QueryRowContext(ctx, `
		UPDATE entities SET state = $1, updated_at = now()
		WHERE id = $2
		RETURNING id, name, state, created_at, updated_at
	`, to, entityID)
	if err := updateRow.Scan(&e.ID, &e.Name, &e.State, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Store) History(ctx context.Context, entityID string) ([]TransitionRecord, error) {
	if _, err := s.GetEntity(ctx, entityID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, entity_id, from_state, to_state, actor, reason, occurred_at
		FROM transitions
		WHERE entity_id = $1
		ORDER BY occurred_at
	`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hist := make([]TransitionRecord, 0)
	for rows.Next() {
		var r TransitionRecord
		if err := rows.Scan(&r.ID, &r.EntityID, &r.FromState, &r.ToState, &r.Actor, &r.Reason, &r.OccurredAt); err != nil {
			return nil, err
		}
		hist = append(hist, r)
	}
	return hist, rows.Err()
}
