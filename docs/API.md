# Sahiy Stream — API Reference

> Base URL: `https://api.sahiy.stream/v1`
> WebSocket: `wss://ws.sahiy.stream/v1`

---

## Authentication

### Token flow

```
1. POST /auth/login → { access_token, refresh_token }
2. access_token: Bearer header, 15 min TTL
3. refresh_token: httpOnly cookie, 7 day TTL
4. POST /auth/refresh → new access_token (rotation)
```

### Error format

```json
{
  "error": {
    "code": "STREAM_NOT_FOUND",
    "message": "Stream with id xxx not found",
    "details": {}
  }
}
```

### Standard error codes

| HTTP | Code | Ma'nosi |
|------|------|---------|
| 400 | `VALIDATION_ERROR` | Input noto'g'ri |
| 401 | `UNAUTHORIZED` | Token yo'q yoki expired |
| 403 | `FORBIDDEN` | Ruxsat yo'q |
| 404 | `NOT_FOUND` | Resource topilmadi |
| 409 | `CONFLICT` | Duplicate (username, email) |
| 429 | `RATE_LIMITED` | Juda ko'p so'rov |
| 500 | `INTERNAL_ERROR` | Server xatosi |

---

## Pagination

```
GET /streams/live?page=1&limit=20&sort=viewers&order=desc

Response:
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1523,
    "has_next": true
  }
}
```

### Cursor-based (chat history)

```
GET /chat/{stream_id}/history?cursor=12345&limit=50

Response:
{
  "messages": [...],
  "next_cursor": "12295",
  "has_more": true
}
```

---

## Stream playback flow

```
1. GET /streams/{id}/playback
   → { "url": "https://cdn.../master.m3u8?token=...", "expires_at": "..." }

2. Player loads master.m3u8
3. Player selects quality based on bandwidth
4. Segments served from CDN (signed URL)
```

## Ingest flow

```
1. GET /streams/{id}/ingest (authenticated, channel owner)
   → {
       "rtmp_url": "rtmp://ingest.sahiy.stream/live",
       "stream_key": "sk_live_xxxx",
       "srt_url": "srt://ingest.sahiy.stream:9000",
       "whip_url": "https://ingest.sahiy.stream/whip/{key}"
     }

2. OBS configured with URL + key
3. On connect: ingest validates key → media-orchestrator starts transcode
4. Stream status: scheduled → live (automatic)
```

---

*Batafsil endpointlar: [ARCHITECTURE.md §8](./ARCHITECTURE.md#8-api-dizayn)*
