# Sahiy Stream — Database Design Reference

> PostgreSQL asosiy schema, Redis cache patterns, ClickHouse analytics.
> To'liq SQL migratsiyalar `services/*/migrations/` da saqlanadi.

---

## Entity Relationship Diagram

```
users ──────────────┬────────── channels ──────────┬────────── streams
  │                 │                │              │
  │                 │                │              ├──── chat_messages
  │                 │                │              │
  │                 │                ├──── followers│
  │                 │                │              ├──── vod_recordings
  │                 │                │              │
  │                 │                ├──── subscriptions
  │                 │                │
  │                 │                ├──── stream_keys
  │                 │                │
  │                 │                └──── chat_bans
  │                 │
  ├──── oauth_accounts
  ├──── sessions
  ├──── reports
  └──── payments

categories ──── channels
           └─── streams
```

---

## Indexing strategiyasi

### Critical queries va ularning indexlari

| Query | Index | Sabab |
|-------|-------|-------|
| Live streams list | `idx_streams_live (status) WHERE status = 'live'` | Partial index, tez |
| Channel by slug | `idx_channels_slug` | Unique lookup |
| Chat history | `idx_chat_stream_time (stream_id, created_at DESC)` | Pagination |
| User followers | `idx_followers_channel (channel_id)` | Count & list |
| Stream by channel | `idx_streams_channel (channel_id)` | Channel page |
| Audit logs | `idx_audit_time (created_at DESC)` | Admin queries |

### Partitioning (scale uchun)

```sql
-- Chat messages: oylik partition (100M+ rows)
CREATE TABLE chat_messages (
    ...
) PARTITION BY RANGE (created_at);

-- Analytics events: ClickHouse da (PostgreSQL emas)
```

---

## Redis cache invalidation

| Event | Invalidation |
|-------|-------------|
| `user.updated` | `DEL user:{id}:*` |
| `channel.updated` | `DEL channel:{id}:*` |
| `stream.started` | `SET channel:{id}:live = stream_id` |
| `stream.ended` | `DEL channel:{id}:live`, `DEL stream:live:{id}` |
| `follow.created` | `DEL user:{id}:following` |

---

## Connection pooling

```go
// PostgreSQL — har bir service uchun
MaxConns: 25          // per service instance
MinConns: 5
MaxConnLifetime: 1h
MaxConnIdleTime: 30m

// Redis
PoolSize: 20
MinIdleConns: 5
```

---

*Batafsil schema: [ARCHITECTURE.md §7](./ARCHITECTURE.md#7-database-dizayn)*
