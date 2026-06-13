CREATE TABLE stream_media (
    stream_id       UUID PRIMARY KEY REFERENCES streams(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL DEFAULT 'idle',
    hls_path        TEXT,
    playback_url    TEXT,
    ingest_name     TEXT,
    ffmpeg_pid      INTEGER,
    started_at      TIMESTAMPTZ,
    stopped_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stream_media_status ON stream_media(status);
