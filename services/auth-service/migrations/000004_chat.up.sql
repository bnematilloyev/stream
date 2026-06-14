-- Phase 5: real-time chat

CREATE TABLE chat_messages (
    id              BIGSERIAL PRIMARY KEY,
    stream_id       UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    username        VARCHAR(50) NOT NULL,
    display_name    VARCHAR(100) NOT NULL DEFAULT '',
    content         TEXT NOT NULL,
    type            VARCHAR(20) NOT NULL DEFAULT 'text',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_stream_time ON chat_messages(stream_id, created_at DESC);

CREATE TABLE chat_bans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id       UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    banned_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    reason          TEXT,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(stream_id, user_id)
);

CREATE INDEX idx_chat_bans_stream_user ON chat_bans(stream_id, user_id);
