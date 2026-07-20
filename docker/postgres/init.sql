-- entities: エンティティの現在状態のみを保持する。
CREATE TABLE entities (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    state      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- transitions: 状態遷移の監査ログ。アプリケーションからは一切INSERTしない。
-- entitiesへのINSERT/UPDATEに対するトリガー経由でのみ行が増える。
CREATE TABLE transitions (
    id          TEXT PRIMARY KEY,
    entity_id   TEXT NOT NULL REFERENCES entities(id),
    from_state  TEXT NOT NULL DEFAULT '',
    to_state    TEXT NOT NULL,
    actor       TEXT NOT NULL,
    reason      TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transitions_entity_id_occurred_at ON transitions (entity_id, occurred_at);

-- アプリケーションは INSERT/UPDATE の直前に同一トランザクション内で
--   SELECT set_config('app.actor', $1, true);
--   SELECT set_config('app.reason', $1, true);
-- を実行し、「誰が・なぜ」をトリガーに引き渡す。
-- is_local=true なのでトランザクション終了時に自動でリセットされる。

CREATE OR REPLACE FUNCTION record_entity_enrolment() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO transitions (id, entity_id, from_state, to_state, actor, reason, occurred_at)
    VALUES (
        'tr_' || replace(gen_random_uuid()::text, '-', ''),
        NEW.id,
        '',
        NEW.state,
        COALESCE(current_setting('app.actor', true), 'system'),
        COALESCE(current_setting('app.reason', true), 'initial enrolment'),
        now()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_entities_enrolment
    AFTER INSERT ON entities
    FOR EACH ROW
    EXECUTE FUNCTION record_entity_enrolment();

CREATE OR REPLACE FUNCTION record_entity_transition() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.state IS DISTINCT FROM OLD.state THEN
        INSERT INTO transitions (id, entity_id, from_state, to_state, actor, reason, occurred_at)
        VALUES (
            'tr_' || replace(gen_random_uuid()::text, '-', ''),
            NEW.id,
            OLD.state,
            NEW.state,
            COALESCE(current_setting('app.actor', true), 'unknown'),
            COALESCE(current_setting('app.reason', true), ''),
            now()
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_entities_state_transition
    AFTER UPDATE ON entities
    FOR EACH ROW
    EXECUTE FUNCTION record_entity_transition();
