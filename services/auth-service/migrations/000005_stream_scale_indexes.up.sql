-- Scale-oriented indexes for live discovery, channel history, ingest health, and cleanup jobs.

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_streams_live_public_viewers
    ON streams (viewer_count DESC, started_at DESC)
    WHERE status = 'live' AND visibility = 'public';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_streams_channel_status_created
    ON streams (channel_id, status, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_stream_media_ingesting_stream
    ON stream_media (stream_id)
    WHERE status = 'ingesting';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_stream_media_updated_status
    ON stream_media (status, updated_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_channels_live_updated
    ON channels (updated_at DESC)
    WHERE is_live = TRUE;
