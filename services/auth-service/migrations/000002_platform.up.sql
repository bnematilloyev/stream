-- Phase 2: channels, categories, social, streams

CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) UNIQUE NOT NULL,
    icon_url    TEXT,
    parent_id   UUID REFERENCES categories(id),
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE channels (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slug            VARCHAR(50) UNIQUE NOT NULL,
    title           VARCHAR(100) NOT NULL,
    description     TEXT,
    banner_url      TEXT,
    avatar_url      TEXT,
    category_id     UUID REFERENCES categories(id),
    is_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    is_live         BOOLEAN NOT NULL DEFAULT FALSE,
    follower_count  INTEGER NOT NULL DEFAULT 0,
    settings        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_channels_user_id ON channels(user_id);
CREATE INDEX idx_channels_slug ON channels(slug);
CREATE INDEX idx_channels_is_live ON channels(is_live) WHERE is_live = TRUE;

CREATE TABLE followers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    follower_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    notifications   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(follower_id, channel_id)
);

CREATE INDEX idx_followers_channel_id ON followers(channel_id);
CREATE INDEX idx_followers_follower_id ON followers(follower_id);

CREATE TABLE stream_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    key_lookup      VARCHAR(64) UNIQUE NOT NULL,
    key_prefix      VARCHAR(16) NOT NULL,
    label           VARCHAR(50) NOT NULL DEFAULT 'default',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stream_keys_channel_id ON stream_keys(channel_id);
CREATE INDEX idx_stream_keys_active ON stream_keys(channel_id) WHERE is_active = TRUE;

CREATE TABLE streams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    thumbnail_url   TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    ingest_protocol VARCHAR(10) NOT NULL DEFAULT 'rtmp',
    latency_mode    VARCHAR(20) NOT NULL DEFAULT 'standard',
    visibility      VARCHAR(20) NOT NULL DEFAULT 'public',
    category_id     UUID REFERENCES categories(id),
    tags            TEXT[] NOT NULL DEFAULT '{}',
    scheduled_at    TIMESTAMPTZ,
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,
    viewer_count    INTEGER NOT NULL DEFAULT 0,
    peak_viewers    INTEGER NOT NULL DEFAULT 0,
    duration_sec    INTEGER,
    ingest_region   VARCHAR(20),
    settings        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_streams_channel_id ON streams(channel_id);
CREATE INDEX idx_streams_status ON streams(status);
CREATE INDEX idx_streams_live ON streams(status) WHERE status = 'live';
CREATE INDEX idx_streams_scheduled_at ON streams(scheduled_at) WHERE status = 'scheduled';

-- Seed default categories
INSERT INTO categories (name, slug, sort_order) VALUES
    ('Gaming', 'gaming', 1),
    ('Music', 'music', 2),
    ('Talk Shows', 'talk-shows', 3),
    ('Education', 'education', 4),
    ('Sports', 'sports', 5),
    ('Creative', 'creative', 6),
    ('Technology', 'technology', 7),
    ('Just Chatting', 'just-chatting', 8);
