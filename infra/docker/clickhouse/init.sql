CREATE DATABASE IF NOT EXISTS sahiy_analytics;

CREATE TABLE IF NOT EXISTS sahiy_analytics.viewer_heartbeats
(
    stream_id    String,
    session_id   String,
    concurrent   UInt32,
    unique_count UInt32,
    recorded_at  DateTime64(3, 'UTC')
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(recorded_at)
ORDER BY (stream_id, recorded_at)
TTL recorded_at + INTERVAL 90 DAY;
