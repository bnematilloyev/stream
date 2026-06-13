# Sahiy Stream — Ideal Live Streaming Platform
## To'liq System Design & Implementation Blueprint

> **Maqsad:** YouTube Live / Twitch darajasidagi, past latency, ko'p sifatli (ABR), xavfsiz, millionlab concurrent viewerlarga chidamli live streaming platforma.
>
> **Stack:** Go (backend) · Next.js + TypeScript (frontend) · PostgreSQL · Redis · Kubernetes · CDN
>
> **Prinsip:** MVP yo'q — har bir komponent production-grade, scale-ready va xavfsizlik birinchi o'rinda.

---

## Mundarija

1. [Platforma ko'rinishi](#1-platforma-korinishi)
2. [Yuqori darajadagi arxitektura](#2-yuqori-darajadagi-arxitektura)
3. [Microservices (Go)](#3-microservices-go)
4. [Media Pipeline](#4-media-pipeline)
5. [Latency strategiyasi](#5-latency-strategiyasi)
6. [Video sifat & ABR](#6-video-sifat--abr)
7. [Database dizayn](#7-database-dizayn)
8. [API dizayn](#8-api-dizayn)
9. [Real-time tizimlar](#9-real-time-tizimlar)
10. [Frontend arxitektura (Next.js)](#10-frontend-arxitektura-nextjs)
11. [Xavfsizlik](#11-xavfsizlik)
12. [Infratuzilma & DevOps](#12-infratuzilma--devops)
13. [Monitoring & Observability](#13-monitoring--observability)
14. [Disaster Recovery & HA](#14-disaster-recovery--ha)
15. [Loyiha strukturasi](#15-loyiha-strukturasi)
16. [Implementation tartibi](#16-implementation-tartibi)

---

## 1. Platforma ko'rinishi

### 1.1 Asosiy funksiyalar

| Modul | Funksiyalar |
|-------|-------------|
| **Auth & Identity** | Email/phone, OAuth2 (Google, Apple), 2FA/TOTP, session management, device trust |
| **Channels** | Channel yaratish, branding, banner/avatar, custom URL, verification badge |
| **Live Streaming** | RTMP/SRT/WebRTC ingest, multi-bitrate ABR, DVR, stream keys, scheduled streams |
| **VOD** | Avtomatik yozib olish, chapter/markers, thumbnail generation, transcoding |
| **Chat** | Real-time chat, emotes, badges, slow mode, subscriber-only mode |
| **Moderation** | Auto-mod (AI), word filter, timeout/ban, mod queue, report system |
| **Monetization** | Subscriptions, super chat, ads (VAST), revenue dashboard |
| **Discovery** | Home feed, categories, search (Elasticsearch), recommendations, trending |
| **Analytics** | Real-time viewer count, watch time, retention, geographic, device breakdown |
| **Notifications** | Push (FCM/APNs), email, in-app, "channel went live" alerts |
| **Admin** | Platform admin panel, content policy, appeals, audit logs |

### 1.2 No-functional talablar (SLO/SLA)

| Metrika | Maqsad |
|---------|--------|
| API latency (p99) | < 100ms |
| Stream start latency (ingest → playable) | < 3 soniya |
| Playback latency (LL-HLS) | 2–5 soniya |
| Playback latency (WebRTC mode) | 0.5–2 soniya |
| Uptime | 99.95% |
| Concurrent viewers (per stream) | 500K+ |
| Concurrent streams (platform) | 50K+ |
| Chat messages/sec (global) | 100K+ |
| Data durability | 99.999999999% (11 nines) |

---

## 2. Yuqori darajadagi arxitektura

```
                                    ┌─────────────────────────────────────────┐
                                    │              GLOBAL CDN LAYER            │
                                    │   (Cloudflare / Multi-CDN strategy)      │
                                    └──────────────────┬──────────────────────┘
                                                       │
          ┌────────────────────────────────────────────┼────────────────────────────────────────────┐
          │                                            │                                            │
          ▼                                            ▼                                            ▼
   ┌──────────────┐                          ┌──────────────────┐                          ┌──────────────┐
   │  Next.js App │                          │   API Gateway    │                          │  WHIP/WHEP   │
   │  (Frontend)  │◄──── REST/GraphQL ──────►│  (Go + Envoy)    │                          │  Edge Nodes  │
   └──────────────┘                          └────────┬─────────┘                          └──────┬───────┘
                                                       │                                            │
                              ┌─────────────────────────┼─────────────────────────┐                  │
                              │                         │                         │                  │
                              ▼                         ▼                         ▼                  ▼
                    ┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐   ┌──────────────┐
                    │  Auth Service   │      │ Stream Service  │      │  Chat Service   │   │ WebRTC SFU   │
                    └────────┬────────┘      └────────┬────────┘      └────────┬────────┘   └──────────────┘
                             │                          │                          │
                    ┌────────┴────────┐      ┌──────────┴──────────┐      ┌────────┴────────┐
                    │  User Service   │      │ Media Orchestrator  │      │ Notification    │
                    └────────┬────────┘      └──────────┬──────────┘      └─────────────────┘
                             │                          │
                    ┌────────┴────────┐      ┌──────────┴──────────┐
                    │ Moderation Svc  │      │ Transcode Workers   │
                    └────────┬────────┘      │ (FFmpeg + GPU)      │
                             │               └──────────┬──────────┘
                    ┌────────┴────────┐                 │
                    │ Analytics Svc   │      ┌──────────┴──────────┐
                    └────────┬────────┘      │  Packager (HLS/    │
                             │               │  DASH/CMAF)         │
                             │               └──────────┬──────────┘
                             │                          │
          ┌──────────────────┴──────────────────────────┴──────────────────┐
          │                         DATA LAYER                              │
          │  PostgreSQL  │  Redis Cluster  │  S3/MinIO  │  Elasticsearch   │
          │              │  NATS JetStream │  ClickHouse│  (search/logs)   │
          └──────────────────────────────────────────────────────────────────┘
```

### 2.1 Arxitektura prinsiplari

- **Event-driven:** Servislar NATS JetStream orqali bog'lanadi (loose coupling)
- **CQRS:** Write (PostgreSQL) va read-heavy analytics (ClickHouse) ajratilgan
- **Multi-region:** Active-active yoki active-passive geo-replication
- **Zero-trust security:** Har bir servis o'z mTLS sertifikati bilan gaplashadi
- **Idempotency:** Barcha kritik operatsiyalar idempotent key bilan
- **Graceful degradation:** CDN fallback, chat throttle, quality downgrade

---

## 3. Microservices (Go)

### 3.1 Servislar ro'yxati

| Servis | Vazifasi | Protokol | Scale |
|--------|----------|----------|-------|
| `api-gateway` | Routing, rate limit, auth middleware, request validation | HTTP/gRPC | Horizontal |
| `auth-service` | Login, register, JWT, OAuth2, 2FA, session | gRPC | Horizontal |
| `user-service` | Profile, channel, followers, subscriptions | gRPC | Horizontal |
| `stream-service` | Stream lifecycle, keys, schedule, status | gRPC | Horizontal |
| `media-orchestrator` | Transcode job dispatch, ingest routing, health | gRPC + NATS | Horizontal |
| `chat-service` | WebSocket hub, message routing, moderation hooks | WebSocket + gRPC | Sharded |
| `moderation-service` | Auto-mod, reports, bans, appeals | gRPC + NATS | Horizontal |
| `notification-service` | Push, email, SMS, in-app | gRPC + NATS | Horizontal |
| `analytics-service` | Event ingestion, aggregation, dashboards API | gRPC + NATS | Horizontal |
| `search-service` | Full-text search, recommendations | gRPC | Horizontal |
| `billing-service` | Payments, subscriptions, payouts | gRPC | Horizontal |
| `admin-service` | Platform admin operations | gRPC | Limited |

### 3.2 Har bir servis ichki strukturasi (Clean Architecture)

```
service-name/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── domain/                  # Entities, value objects, domain errors
│   │   ├── entity.go
│   │   └── repository.go        # Interface
│   ├── usecase/                 # Business logic
│   │   └── *.go
│   ├── adapter/
│   │   ├── handler/             # HTTP/gRPC handlers
│   │   │   ├── grpc/
│   │   │   └── http/
│   │   ├── repository/          # PostgreSQL, Redis implementations
│   │   └── publisher/           # NATS event publisher
│   └── config/
│       └── config.go
├── proto/                       # gRPC definitions (shared)
├── migrations/
├── Dockerfile
└── go.mod
```

### 3.3 Shared packages (`/pkg`)

```
pkg/
├── auth/           # JWT middleware, RBAC helpers
├── database/       # PostgreSQL connection pool (pgx)
├── redis/          # Redis client wrapper
├── nats/           # NATS JetStream helpers
├── logger/         # Structured logging (zap)
├── metrics/        # Prometheus metrics
├── tracing/        # OpenTelemetry
├── errors/         # Standardized error codes
├── validator/      # Input validation
├── crypto/         # Encryption, hashing utilities
└── grpc/           # gRPC interceptors (auth, logging, recovery)
```

### 3.4 Texnologiya tanlovlari (Go)

| Komponent | Tanlov | Sabab |
|-----------|--------|-------|
| HTTP framework | **Fiber** yoki **chi** | Tez, middleware ekosistema |
| gRPC | **google.golang.org/grpc** | Inter-service communication |
| ORM | **sqlc** (code generation) | Type-safe SQL, performance |
| Migrations | **golang-migrate** | Version-controlled schema |
| Config | **viper** + env | 12-factor app |
| Validation | **go-playground/validator** | Struct validation |
| Logging | **zap** | Structured, fast |
| Testing | **testify** + **testcontainers** | Integration tests |

---

## 4. Media Pipeline

### 4.1 To'liq media oqimi

```
Streamer (OBS/Mobile)
        │
        ├── RTMP ──────────► Ingest Server (nginx-rtmp / custom Go)
        ├── SRT ───────────► SRT Listener (Haivision libsrt)
        └── WebRTC/WHIP ───► WHIP Endpoint → SFU (ion-sfu / livekit)
                │
                ▼
        ┌───────────────────┐
        │  Ingest Router    │  ← stream key validation, region routing
        │  (Go service)     │
        └────────┬──────────┘
                 │
                 ▼
        ┌───────────────────┐
        │ Transcode Cluster │  ← FFmpeg + NVIDIA NVENC (GPU nodes)
        │                   │
        │  Input: 1080p60   │
        │  Outputs:         │
        │   - 1080p60 6Mbps │
        │   - 720p60  3Mbps │
        │   - 480p30  1.5Mbps│
        │   - 360p30  800Kbps│
        │   - 240p30  400Kbps│
        │   - 144p30  200Kbps│ (mobil fallback)
        └────────┬──────────┘
                 │
                 ▼
        ┌───────────────────┐
        │  Packager         │  ← CMAF/fMP4 segments
        │                   │
        │  - LL-HLS (.m3u8) │  ← primary delivery
        │  - DASH (.mpd)    │  ← fallback
        │  - WebRTC out     │  ← ultra-low latency mode
        └────────┬──────────┘
                 │
        ┌────────┴────────┐
        ▼                 ▼
   Origin Storage     Live DVR Buffer
   (S3/MinIO)        (Redis + local SSD)
        │
        ▼
   CDN Distribution (signed URLs, geo-routing)
        │
        ▼
   Player (hls.js / Shaka / WebRTC WHEP)
```

### 4.2 Ingest server

**RTMP Ingest (custom Go yoki nginx-rtmp module):**

```
rtmp://ingest.sahiy.stream/live/{stream_key}
```

- Stream key validation (Redis cache → PostgreSQL fallback)
- Region-based routing (eng yaqin ingest node)
- Connection limit per key
- Automatic reconnect handling
- Health heartbeat har 5 soniyada

**WHIP (WebRTC HTTP Ingest Protocol):**

```
POST https://ingest.sahiy.stream/whip/{stream_key}
Content-Type: application/sdp
```

- SFU: **LiveKit** (Go-native) yoki **ion-sfu**
- Simulcast support (bir nechta sifat bir vaqtda)
- TURN server (NAT traversal uchun)

### 4.3 Transcoding

**Worker architecture:**

```go
// media-orchestrator dispatches jobs
type TranscodeJob struct {
    StreamID    string
    InputURL    string            // rtmp://internal/...
    Outputs     []TranscodeOutput
    Priority    int               // live > VOD re-encode
    GPURequired bool
}

type TranscodeOutput struct {
    Resolution  string            // "1920x1080"
    Bitrate     int               // kbps
    Codec       string            // h264, h265, av1
    FPS         int
    Profile     string            // main, high
}
```

**FFmpeg command template (1080p output):**

```bash
ffmpeg -hwaccel cuda -i rtmp://input \
  -c:v h264_nvenc -preset p4 -profile:v high \
  -b:v 6000k -maxrate 6500k -bufsize 12000k \
  -s 1920x1080 -r 60 -g 120 \
  -c:a aac -b:a 192k -ar 48000 \
  -f hls -hls_time 2 -hls_list_size 10 \
  -hls_flags independent_segments+program_date_time \
  output_1080p.m3u8
```

**GPU cluster:**
- NVIDIA A10 / L4 instances
- Har bir GPU: ~10 concurrent 1080p transcode
- Auto-scaling: stream count bo'yicha worker spawn
- Spot instances + on-demand fallback

### 4.4 Packaging (LL-HLS)

**CMAF segment strukturasi:**

```
/stream/{stream_id}/
├── master.m3u8              # Master playlist (barcha bitratelar)
├── 1080p/
│   ├── playlist.m3u8        # Media playlist
│   ├── init.mp4             # Initialization segment
│   └── seg_00001.m4s        # Media segments (2s each)
├── 720p/
│   └── ...
└── audio/
    └── ...
```

**LL-HLS xususiyatlari:**
- Partial segments (200ms chunks)
- Blocking playlist reload
- Preload hints
- Delta updates
- `EXT-X-PART` tags

### 4.5 DVR (Digital Video Recording)

- Live stream real-time yozib olinadi
- Oxirgi 2 soat buffer (seek back during live)
- Stream tugagach → to'liq VOD sifatida saqlanadi
- Thumbnail har 10 soniyada (sprite sheet)
- Chapter markers (streamer belgilaydi)

### 4.6 VOD Processing Pipeline

```
Stream ends
    │
    ▼
Finalize recording (concat segments)
    │
    ▼
Generate thumbnails + preview GIF
    │
    ▼
Transcode all qualities (agar live da barcha sifat bo'lmasa)
    │
    ▼
Upload to S3 (multi-region replication)
    │
    ▼
Update database (vod_recordings table)
    │
    ▼
Index in Elasticsearch (searchable)
    │
    ▼
Publish event: vod.ready
```

---

## 5. Latency strategiyasi

### 5.1 Ikki rejim

| Rejim | Texnologiya | Latency | Foydalanish |
|-------|-------------|---------|-------------|
| **Standard** | LL-HLS (CMAF) | 3–5s | Default — ko'pchilik streamlar |
| **Ultra-Low** | WebRTC (WHEP) | 0.5–2s | Gaming, auction, interactive |

### 5.2 LL-HLS latency optimizatsiyasi

```
Encoder (OBS)          ~0ms (real-time)
    ↓ RTMP
Ingest server          ~50ms
    ↓
Transcode (GPU)        ~500ms–1s
    ↓
Packager (2s segments) ~2s
    ↓
CDN propagation        ~200–500ms
    ↓
Player buffer          ~1–2s
    ↓
TOTAL                  ~3–5s
```

**Optimizatsiya:**
- 2 soniyalik segmentlar (4s emas)
- Partial segments (200ms)
- CDN edge caching (stale-while-revalidate)
- Player: `liveSyncDurationCount: 2` (hls.js)
- Preconnect CDN origin

### 5.3 WebRTC ultra-low latency

```
Streamer → WHIP → SFU → WHEP → Viewer
Total: 500ms–2s
```

- SFU cluster (region-based)
- Simulcast: viewer bandwidth ga qarab layer tanlash
- TURN server (coturn) — firewall ortidagi userlar uchun
- Max 10K concurrent per SFU node

### 5.4 Latency monitoring

- End-to-end latency o'lchash (encoder timestamp → player display)
- SRT/RTMP ingest delay metrics
- CDN segment age tracking
- Alert: latency > 10s bo'lsa

---

## 6. Video sifat & ABR

### 6.1 Adaptive Bitrate ladder

| Tier | Resolution | FPS | Video Bitrate | Audio | Codec |
|------|-----------|-----|---------------|-------|-------|
| 0 | 3840x2160 (4K) | 60 | 15 Mbps | 256k AAC | H.265/AV1 |
| 1 | 1920x1080 | 60 | 6 Mbps | 192k AAC | H.264 |
| 2 | 1920x1080 | 30 | 4 Mbps | 192k AAC | H.264 |
| 3 | 1280x720 | 60 | 3 Mbps | 128k AAC | H.264 |
| 4 | 1280x720 | 30 | 2 Mbps | 128k AAC | H.264 |
| 5 | 854x480 | 30 | 1.5 Mbps | 128k AAC | H.264 |
| 6 | 640x360 | 30 | 800 Kbps | 96k AAC | H.264 |
| 7 | 426x240 | 30 | 400 Kbps | 96k AAC | H.264 |
| 8 | 256x144 | 30 | 200 Kbps | 64k AAC | H.264 |

### 6.2 ABR algoritmi (player-side)

- **Buffer-based** (BOLA) — default
- **Throughput-based** fallback
- Quality switch: seamless (CMAF aligned segments)
- Bandwidth estimate: exponential moving average
- Min 3 segment buffer before quality upgrade
- Immediate downgrade on buffer starvation

### 6.3 Codec strategiyasi

| Codec | Qachon | Sabab |
|-------|--------|-------|
| H.264 (AVC) | Default, barcha tierlar | Universal support |
| H.265 (HEVC) | 4K, premium | 40% bandwidth tejash |
| AV1 | Kelajak (6 oy+) | 50% bandwidth tejash, royalty-free |
| VP9 | Fallback | Browser support |

### 6.4 Audio

- AAC-LC 48kHz stereo (default)
- Opus (WebRTC mode)
- Audio-only stream support (podcast/radio)

---

## 7. Database dizayn

### 7.1 PostgreSQL — asosiy schema

```sql
-- ============================================
-- USERS & AUTH
-- ============================================

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE,
    phone           VARCHAR(20) UNIQUE,
    username        VARCHAR(50) UNIQUE NOT NULL,
    display_name    VARCHAR(100) NOT NULL,
    avatar_url      TEXT,
    password_hash   TEXT,                    -- nullable (OAuth users)
    email_verified  BOOLEAN DEFAULT FALSE,
    phone_verified  BOOLEAN DEFAULT FALSE,
    totp_secret     TEXT,                    -- 2FA
    totp_enabled    BOOLEAN DEFAULT FALSE,
    role            VARCHAR(20) DEFAULT 'user',  -- user, moderator, admin
    status          VARCHAR(20) DEFAULT 'active', -- active, suspended, banned
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE oauth_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        VARCHAR(20) NOT NULL,     -- google, apple, github
    provider_id     VARCHAR(255) NOT NULL,
    access_token    TEXT,
    refresh_token   TEXT,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_id)
);

CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token   TEXT UNIQUE NOT NULL,
    device_info     JSONB,
    ip_address      INET,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- ============================================
-- CHANNELS
-- ============================================

CREATE TABLE channels (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slug            VARCHAR(50) UNIQUE NOT NULL,
    title           VARCHAR(100) NOT NULL,
    description     TEXT,
    banner_url      TEXT,
    avatar_url      TEXT,
    category_id     UUID REFERENCES categories(id),
    is_verified     BOOLEAN DEFAULT FALSE,
    is_live         BOOLEAN DEFAULT FALSE,
    follower_count  INTEGER DEFAULT 0,
    settings        JSONB DEFAULT '{}',        -- chat settings, etc.
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_channels_user ON channels(user_id);
CREATE INDEX idx_channels_slug ON channels(slug);
CREATE INDEX idx_channels_live ON channels(is_live) WHERE is_live = TRUE;

CREATE TABLE categories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(100) UNIQUE NOT NULL,
    icon_url        TEXT,
    parent_id       UUID REFERENCES categories(id),
    sort_order      INTEGER DEFAULT 0
);

-- ============================================
-- STREAMS
-- ============================================

CREATE TABLE streams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    thumbnail_url   TEXT,
    status          VARCHAR(20) DEFAULT 'scheduled',
                    -- scheduled, live, ended, processing, ready
    stream_key      VARCHAR(64) UNIQUE NOT NULL,
    ingest_protocol VARCHAR(10) DEFAULT 'rtmp',  -- rtmp, srt, whip
    latency_mode    VARCHAR(20) DEFAULT 'standard', -- standard, ultra-low
    visibility      VARCHAR(20) DEFAULT 'public', -- public, unlisted, private
    category_id     UUID REFERENCES categories(id),
    tags            TEXT[],
    scheduled_at    TIMESTAMPTZ,
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,
    viewer_count    INTEGER DEFAULT 0,
    peak_viewers    INTEGER DEFAULT 0,
    duration_sec    INTEGER,
    ingest_region   VARCHAR(20),
    settings        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_streams_channel ON streams(channel_id);
CREATE INDEX idx_streams_status ON streams(status);
CREATE INDEX idx_streams_live ON streams(status) WHERE status = 'live';
CREATE INDEX idx_streams_scheduled ON streams(scheduled_at) WHERE status = 'scheduled';

CREATE TABLE stream_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    key_hash        VARCHAR(128) NOT NULL,      -- bcrypt hash
    label           VARCHAR(50),
    is_active       BOOLEAN DEFAULT TRUE,
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- VOD (Video on Demand)
-- ============================================

CREATE TABLE vod_recordings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id       UUID REFERENCES streams(id),
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    thumbnail_url   TEXT,
    duration_sec    INTEGER NOT NULL,
    status          VARCHAR(20) DEFAULT 'processing', -- processing, ready, failed
    storage_path    TEXT NOT NULL,
    qualities       JSONB,                     -- available quality tiers
    view_count      INTEGER DEFAULT 0,
    visibility      VARCHAR(20) DEFAULT 'public',
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_vod_channel ON vod_recordings(channel_id);
CREATE INDEX idx_vod_status ON vod_recordings(status);

-- ============================================
-- SOCIAL
-- ============================================

CREATE TABLE followers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    follower_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    notifications   BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(follower_id, channel_id)
);

CREATE INDEX idx_followers_channel ON followers(channel_id);
CREATE INDEX idx_followers_user ON followers(follower_id);

CREATE TABLE subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscriber_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    tier            VARCHAR(20) DEFAULT 'basic',  -- basic, premium
    status          VARCHAR(20) DEFAULT 'active',
    stripe_sub_id   VARCHAR(255),
    current_period_start TIMESTAMPTZ,
    current_period_end   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(subscriber_id, channel_id)
);

-- ============================================
-- CHAT
-- ============================================

CREATE TABLE chat_messages (
    id              BIGSERIAL PRIMARY KEY,
    stream_id       UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id),
    content         TEXT NOT NULL,
    type            VARCHAR(20) DEFAULT 'text',  -- text, emote, system, super_chat
    metadata        JSONB DEFAULT '{}',          -- super_chat amount, emote id
    is_deleted      BOOLEAN DEFAULT FALSE,
    deleted_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_chat_stream_time ON chat_messages(stream_id, created_at DESC);

-- Partition by month for chat_messages at scale:
-- CREATE TABLE chat_messages_2026_06 PARTITION OF chat_messages
--     FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE TABLE chat_bans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id),
    banned_by       UUID NOT NULL REFERENCES users(id),
    reason          TEXT,
    expires_at      TIMESTAMPTZ,               -- NULL = permanent
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- MODERATION
-- ============================================

CREATE TABLE reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id     UUID NOT NULL REFERENCES users(id),
    target_type     VARCHAR(20) NOT NULL,       -- stream, user, message, vod
    target_id       UUID NOT NULL,
    reason          VARCHAR(50) NOT NULL,
    description     TEXT,
    status          VARCHAR(20) DEFAULT 'pending', -- pending, reviewed, actioned, dismissed
    reviewed_by     UUID REFERENCES users(id),
    action_taken    TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE moderation_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id      UUID REFERENCES channels(id), -- NULL = global
    type            VARCHAR(20) NOT NULL,        -- word_filter, link_block, caps_limit
    pattern         TEXT NOT NULL,
    action          VARCHAR(20) DEFAULT 'block', -- block, flag, timeout
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- BILLING
-- ============================================

CREATE TABLE payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    type            VARCHAR(20) NOT NULL,         -- subscription, super_chat, tip
    amount_cents    INTEGER NOT NULL,
    currency        VARCHAR(3) DEFAULT 'USD',
    stripe_payment_id VARCHAR(255),
    status          VARCHAR(20) DEFAULT 'pending',
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- AUDIT
-- ============================================

CREATE TABLE audit_logs (
    id              BIGSERIAL PRIMARY KEY,
    actor_id        UUID REFERENCES users(id),
    action          VARCHAR(50) NOT NULL,
    resource_type   VARCHAR(50) NOT NULL,
    resource_id     UUID,
    details         JSONB,
    ip_address      INET,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_time ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_actor ON audit_logs(actor_id);
```

### 7.2 Redis data structures

| Key pattern | Type | TTL | Maqsad |
|-------------|------|-----|--------|
| `session:{id}` | Hash | 7d | Active sessions |
| `stream:live:{id}` | Hash | - | Live stream metadata |
| `stream:viewers:{id}` | HyperLogLog | - | Viewer count (approximate) |
| `stream:viewers:{id}:exact` | SET | - | Exact viewer set (small streams) |
| `chat:{stream_id}:recent` | LIST | 1h | Recent messages cache |
| `ratelimit:{ip}:{endpoint}` | String (counter) | 1m | Rate limiting |
| `stream:key:{hash}` | String | - | Stream key → channel mapping |
| `channel:{id}:live` | String | - | Channel → active stream ID |
| `user:{id}:following` | SET | 5m | Following list cache |
| `trending:streams` | ZSET | 5m | Trending score |
| `pubsub:chat:{stream_id}` | Pub/Sub | - | Chat message distribution |

### 7.3 ClickHouse — analytics

```sql
CREATE TABLE stream_events (
    event_time      DateTime64(3),
    stream_id       UUID,
    channel_id      UUID,
    user_id         Nullable(UUID),
    event_type      LowCardinality(String),  -- join, leave, quality_change, buffer
    quality         LowCardinality(String),
    region          LowCardinality(String),
    device          LowCardinality(String),
    buffer_duration Float32,
    bandwidth       UInt32
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_time)
ORDER BY (stream_id, event_time);

CREATE TABLE chat_stats (
    event_time      DateTime64(3),
    stream_id       UUID,
    message_count   UInt32,
    unique_chatters UInt32,
    emote_count     UInt32
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(event_time)
ORDER BY (stream_id, event_time);
```

### 7.4 Elasticsearch — search index

```json
{
  "channels": {
    "mappings": {
      "properties": {
        "title":       { "type": "text", "analyzer": "standard" },
        "slug":        { "type": "keyword" },
        "description": { "type": "text" },
        "category":    { "type": "keyword" },
        "follower_count": { "type": "integer" },
        "is_live":     { "type": "boolean" },
        "tags":        { "type": "keyword" }
      }
    }
  },
  "streams": {
    "mappings": {
      "properties": {
        "title":       { "type": "text" },
        "channel_title": { "type": "text" },
        "tags":        { "type": "keyword" },
        "category":    { "type": "keyword" },
        "status":      { "type": "keyword" },
        "started_at":  { "type": "date" },
        "viewer_count": { "type": "integer" }
      }
    }
  }
}
```

---

## 8. API dizayn

### 8.1 API Gateway routing

```
Base URL: https://api.sahiy.stream/v1

Authentication: Bearer JWT (access token, 15min)
Refresh: POST /auth/refresh (refresh token, 7d, httpOnly cookie)
```

### 8.2 REST API endpoints

```
── AUTH ──────────────────────────────────────────
POST   /auth/register
POST   /auth/login
POST   /auth/logout
POST   /auth/refresh
POST   /auth/forgot-password
POST   /auth/reset-password
POST   /auth/verify-email
POST   /auth/2fa/setup
POST   /auth/2fa/verify
GET    /auth/oauth/{provider}
GET    /auth/oauth/{provider}/callback

── USERS ─────────────────────────────────────────
GET    /users/me
PATCH  /users/me
DELETE /users/me
GET    /users/{username}
POST   /users/me/avatar

── CHANNELS ──────────────────────────────────────
POST   /channels
GET    /channels/{slug}
PATCH  /channels/{slug}
GET    /channels/{slug}/streams
GET    /channels/{slug}/vods
GET    /channels/{slug}/followers
POST   /channels/{slug}/follow
DELETE /channels/{slug}/follow

── STREAMS ───────────────────────────────────────
POST   /streams                          # schedule/create
GET    /streams/{id}
PATCH  /streams/{id}
DELETE /streams/{id}
POST   /streams/{id}/start
POST   /streams/{id}/end
GET    /streams/{id}/playback              # signed playback URL
GET    /streams/{id}/ingest                # ingest URL + stream key
POST   /streams/{id}/key/rotate
GET    /streams/live                       # all live streams
GET    /streams/trending

── CHAT ──────────────────────────────────────────
WS     /chat/{stream_id}                   # WebSocket
GET    /chat/{stream_id}/history?cursor=
DELETE /chat/{stream_id}/messages/{id}   # mod delete

── VOD ───────────────────────────────────────────
GET    /vods/{id}
GET    /vods/{id}/playback
PATCH  /vods/{id}
DELETE /vods/{id}

── SEARCH ────────────────────────────────────────
GET    /search?q=&type=channel|stream|vod&page=

── SUBSCRIPTIONS ─────────────────────────────────
POST   /subscriptions/{channel_id}
DELETE /subscriptions/{channel_id}
GET    /subscriptions/me

── MODERATION ────────────────────────────────────
POST   /reports
GET    /moderation/queue                    # mod only
POST   /moderation/actions
POST   /channels/{slug}/bans
DELETE /channels/{slug}/bans/{user_id}

── ANALYTICS ─────────────────────────────────────
GET    /analytics/stream/{id}               # streamer dashboard
GET    /analytics/channel/{slug}
GET    /analytics/realtime/{stream_id}      # live viewer count

── ADMIN ─────────────────────────────────────────
GET    /admin/users?status=&page=
PATCH  /admin/users/{id}/status
GET    /admin/audit-logs
GET    /admin/platform-stats
```

### 8.3 WebSocket protokoli (Chat)

```json
// Client → Server
{ "type": "message", "content": "Hello!" }
{ "type": "delete", "message_id": 12345 }
{ "type": "pin", "message_id": 12345 }
{ "type": "slow_mode", "seconds": 5 }

// Server → Client
{ "type": "message", "id": 12345, "user": {...}, "content": "Hello!", "ts": "..." }
{ "type": "delete", "message_id": 12345 }
{ "type": "viewer_count", "count": 1523 }
{ "type": "stream_ended" }
{ "type": "user_banned", "user_id": "..." }
{ "type": "system", "content": "Slow mode enabled (5s)" }
```

### 8.4 gRPC inter-service

```protobuf
// proto/stream/v1/stream.proto
service StreamService {
  rpc CreateStream(CreateStreamRequest) returns (Stream);
  rpc GetStream(GetStreamRequest) returns (Stream);
  rpc StartStream(StartStreamRequest) returns (Stream);
  rpc EndStream(EndStreamRequest) returns (Stream);
  rpc GetLiveStreams(GetLiveStreamsRequest) returns (stream Stream);
  rpc ValidateStreamKey(ValidateStreamKeyRequest) returns (ValidateStreamKeyResponse);
  rpc GetPlaybackURL(GetPlaybackURLRequest) returns (PlaybackURL);
}

// proto/chat/v1/chat.proto
service ChatService {
  rpc SendMessage(SendMessageRequest) returns (ChatMessage);
  rpc GetHistory(GetHistoryRequest) returns (ChatHistory);
  rpc DeleteMessage(DeleteMessageRequest) returns (Empty);
  rpc BanUser(BanUserRequest) returns (Empty);
}

// proto/media/v1/media.proto
service MediaOrchestrator {
  rpc StartIngest(StartIngestRequest) returns (IngestInfo);
  rpc StopIngest(StopIngestRequest) returns (Empty);
  rpc GetTranscodeStatus(TranscodeStatusRequest) returns (TranscodeStatus);
  rpc StartRecording(StartRecordingRequest) returns (Recording);
  rpc FinalizeVOD(FinalizeVODRequest) returns (VOD);
}
```

---

## 9. Real-time tizimlar

### 9.1 Chat architecture

```
Viewer WebSocket ──► Chat Gateway (Go, sharded by stream_id)
                           │
                           ├── Validate auth (JWT)
                           ├── Rate limit (5 msg/sec per user)
                           ├── Content filter (moderation-service)
                           ├── Publish to NATS: chat.{stream_id}
                           │
                     NATS JetStream
                           │
                     All Chat Gateway nodes
                           │
                     Broadcast to connected viewers
                           │
                     Async: persist to PostgreSQL
                     Async: publish analytics event
```

**Sharding strategiyasi:**
- `stream_id` hash → gateway node
- Consistent hashing (ketama)
- Har bir node: 10K concurrent WebSocket
- Redis Pub/Sub cross-node message relay

### 9.2 Viewer count

```
Player heartbeat (har 30s) ──► API Gateway
                                    │
                              Redis HyperLogLog (approximate)
                              + Redis SET (exact, < 1000 viewers)
                                    │
                              NATS: viewer.count.{stream_id}
                                    │
                              Chat WebSocket broadcast
```

### 9.3 Live notifications ("channel went live")

```
stream.started event (NATS)
    │
    ▼
Notification Service
    │
    ├── Query: followers WHERE notifications = true
    ├── Batch: 1000 followers per batch
    ├── FCM push (mobile)
    ├── Email (if enabled)
    └── In-app notification (WebSocket)
```

---

## 10. Frontend arxitektura (Next.js)

### 10.1 Loyiha strukturasi

```
frontend/
├── app/
│   ├── (auth)/
│   │   ├── login/page.tsx
│   │   ├── register/page.tsx
│   │   └── layout.tsx
│   ├── (main)/
│   │   ├── page.tsx                    # Home feed
│   │   ├── browse/page.tsx             # Categories
│   │   ├── search/page.tsx
│   │   ├── trending/page.tsx
│   │   └── layout.tsx                  # Navbar, sidebar
│   ├── (watch)/
│   │   ├── live/[id]/page.tsx          # Live stream viewer
│   │   ├── vod/[id]/page.tsx           # VOD viewer
│   │   └── layout.tsx                  # Minimal chrome
│   ├── (creator)/
│   │   ├── studio/page.tsx             # Creator dashboard
│   │   ├── studio/stream/page.tsx      # Go live / stream settings
│   │   ├── studio/analytics/page.tsx
│   │   ├── studio/community/page.tsx   # Mod tools
│   │   ├── studio/settings/page.tsx
│   │   └── layout.tsx
│   ├── channel/[slug]/
│   │   ├── page.tsx                    # Channel home
│   │   ├── videos/page.tsx
│   │   └── about/page.tsx
│   ├── settings/
│   │   ├── profile/page.tsx
│   │   ├── security/page.tsx
│   │   └── notifications/page.tsx
│   ├── admin/                          # Admin panel
│   ├── api/                            # BFF routes (optional)
│   ├── layout.tsx                      # Root layout
│   └── globals.css
├── components/
│   ├── player/
│   │   ├── LivePlayer.tsx              # hls.js + LL-HLS
│   │   ├── VodPlayer.tsx
│   │   ├── PlayerControls.tsx
│   │   ├── QualitySelector.tsx
│   │   └── TheaterMode.tsx
│   ├── chat/
│   │   ├── ChatPanel.tsx
│   │   ├── ChatMessage.tsx
│   │   ├── ChatInput.tsx
│   │   ├── EmotePicker.tsx
│   │   └── ModActions.tsx
│   ├── stream/
│   │   ├── StreamCard.tsx
│   │   ├── StreamGrid.tsx
│   │   ├── LiveBadge.tsx
│   │   └── ViewerCount.tsx
│   ├── channel/
│   │   ├── ChannelHeader.tsx
│   │   ├── FollowButton.tsx
│   │   └── SubscribeButton.tsx
│   └── ui/                             # shadcn/ui components
├── lib/
│   ├── api/                            # API client (fetch wrapper)
│   ├── auth/                           # Auth context, hooks
│   ├── websocket/                      # WebSocket manager
│   ├── player/                         # Player utilities
│   └── utils/
├── hooks/
│   ├── useAuth.ts
│   ├── useChat.ts
│   ├── usePlayer.ts
│   ├── useStream.ts
│   └── useWebSocket.ts
├── stores/                             # Zustand stores
│   ├── authStore.ts
│   ├── chatStore.ts
│   └── playerStore.ts
├── types/
│   └── index.ts
├── next.config.ts
├── tailwind.config.ts
├── tsconfig.json
└── package.json
```

### 10.2 Video Player (ideal)

```typescript
// LivePlayer.tsx — asosiy player komponenti
// Texnologiyalar:
// - hls.js (LL-HLS support)
// - Media Source Extensions (MSE)
// - WebRTC WHEP (ultra-low latency mode)

// Xususiyatlar:
// - Adaptive bitrate (avtomatik + manual quality)
// - DVR seek (live streamda orqaga qaytish)
// - Theater mode / fullscreen / PiP
// - Keyboard shortcuts (space, f, m, arrows)
// - Buffer health indicator
// - Latency indicator
// - Volume persistence (localStorage)
// - Auto-quality based on bandwidth
// - Error recovery (auto-reconnect, fallback CDN)
// - Captions support (future)
```

### 10.3 State management

| State | Tool | Sabab |
|-------|------|-------|
| Server state | **TanStack Query** | Cache, refetch, optimistic updates |
| Client state | **Zustand** | Lightweight, no boilerplate |
| Real-time | **WebSocket + Zustand** | Chat, viewer count |
| Auth | **Context + httpOnly cookies** | Secure token handling |
| Forms | **React Hook Form + Zod** | Validation |

### 10.4 Performance

- **SSR:** Channel pages, stream metadata (SEO)
- **CSR:** Player, chat (client-only, dynamic import)
- **ISR:** Home page, trending (revalidate: 60s)
- **Image optimization:** Next.js Image + CDN
- **Code splitting:** Player lazy loaded
- **Prefetch:** Channel links on hover
- **Service Worker:** Offline channel cache (future)

---

## 11. Xavfsizlik

### 11.1 Authentication & Authorization

```
┌─────────────────────────────────────────────┐
│              Security Layers                 │
├─────────────────────────────────────────────┤
│ 1. CDN/WAF (Cloudflare)                     │
│    - DDoS protection                        │
│    - Bot detection                          │
│    - Geo-blocking (if needed)               │
├─────────────────────────────────────────────┤
│ 2. API Gateway                              │
│    - Rate limiting (per IP, per user)       │
│    - JWT validation                         │
│    - Request size limits                    │
│    - CORS policy                            │
├─────────────────────────────────────────────┤
│ 3. Service Mesh (mTLS)                      │
│    - Service-to-service encryption          │
│    - Identity verification                  │
├─────────────────────────────────────────────┤
│ 4. Application Layer                        │
│    - RBAC (role-based access control)       │
│    - Input validation & sanitization        │
│    - SQL injection prevention (sqlc)        │
│    - XSS prevention (CSP headers)           │
├─────────────────────────────────────────────┤
│ 5. Data Layer                               │
│    - Encryption at rest (AES-256)           │
│    - Encryption in transit (TLS 1.3)        │
│    - PII masking in logs                    │
│    - Database access controls               │
└─────────────────────────────────────────────┘
```

### 11.2 Stream security

| Xavfsizlik | Implementatsiya |
|------------|-----------------|
| Stream key | 32-byte random, bcrypt hash stored, rotation support |
| Playback URL | Signed URL (HMAC-SHA256), 4 soat TTL, IP-bound optional |
| Ingest auth | Stream key validation har connection da |
| Hotlink prevention | Referer check + signed URLs |
| DRM (kelajak) | Widevine/FairPlay (premium content uchun) |
| Content encryption | AES-128 HLS encryption (per-segment keys) |

### 11.3 Rate limiting

| Endpoint | Limit |
|----------|-------|
| Auth (login) | 5/min per IP |
| Auth (register) | 3/min per IP |
| API (authenticated) | 100/min per user |
| API (public) | 30/min per IP |
| Chat messages | 5/sec per user |
| Stream creation | 10/hour per user |
| Search | 20/min per user |
| File upload | 10/hour per user |

### 11.4 Content moderation

- **Auto-mod pipeline:** Chat message → word filter → ML toxicity check → action
- **Report system:** User report → mod queue → review → action
- **Strike system:** 3 strike = temp ban → permanent ban
- **Appeal process:** Ban appeal → admin review
- **Audit trail:** Har bir mod action loglanadi

---

## 12. Infratuzilma & DevOps

### 12.1 Kubernetes architecture

```
┌─── Kubernetes Cluster (per region) ──────────────────────────────┐
│                                                                   │
│  Namespace: gateway                                               │
│  ├── api-gateway (3 replicas, HPA)                               │
│  └── envoy-proxy (ingress)                                       │
│                                                                   │
│  Namespace: services                                              │
│  ├── auth-service (2 replicas)                                   │
│  ├── user-service (2 replicas)                                   │
│  ├── stream-service (3 replicas)                                 │
│  ├── chat-service (5 replicas, sharded)                          │
│  ├── media-orchestrator (2 replicas)                             │
│  ├── moderation-service (2 replicas)                             │
│  ├── notification-service (2 replicas)                           │
│  ├── analytics-service (2 replicas)                              │
│  └── search-service (2 replicas)                                 │
│                                                                   │
│  Namespace: media                                                 │
│  ├── ingest-rtmp (2 replicas, hostNetwork)                     │
│  ├── ingest-whip (2 replicas)                                    │
│  ├── transcode-workers (HPA: 0-50, GPU nodes)                    │
│  ├── packager (2 replicas)                                       │
│  └── vod-processor (HPA: 0-10)                                   │
│                                                                   │
│  Namespace: data                                                  │
│  ├── postgresql (StatefulSet, 3 nodes HA)                        │
│  ├── redis-cluster (6 nodes)                                     │
│  ├── nats-jetstream (3 nodes)                                    │
│  └── clickhouse (3 nodes)                                        │
│                                                                   │
│  Namespace: monitoring                                            │
│  ├── prometheus                                                   │
│  ├── grafana                                                      │
│  ├── alertmanager                                                 │
│  └── jaeger (tracing)                                            │
└───────────────────────────────────────────────────────────────────┘
```

### 12.2 CI/CD Pipeline

```
Developer push
    │
    ▼
GitHub Actions
    ├── Lint (golangci-lint, eslint)
    ├── Unit tests
    ├── Integration tests (testcontainers)
    ├── Security scan (gosec, trivy)
    ├── Build Docker images
    ├── Push to container registry
    │
    ▼
Staging deploy (automatic)
    ├── Smoke tests
    ├── Load tests (k6)
    │
    ▼
Production deploy (manual approval)
    ├── Rolling update
    ├── Health check verification
    ├── Auto-rollback on error
```

### 12.3 Environment strategiya

| Environment | Maqsad | Infratuzilma |
|-------------|--------|--------------|
| `dev` | Local development | Docker Compose |
| `staging` | Pre-production testing | K8s (1 region, minimal) |
| `production` | Live platform | K8s (multi-region) |

### 12.4 Docker Compose (local dev)

```yaml
# docker-compose.yml — local development
services:
  postgres:
    image: postgres:16
  redis:
    image: redis:7-alpine
  nats:
    image: nats:2.10-alpine
  minio:
    image: minio/minio         # S3-compatible storage
  elasticsearch:
    image: elasticsearch:8.12
  # Services built from Dockerfile
  api-gateway:
    build: ./services/api-gateway
  auth-service:
    build: ./services/auth-service
  # ... etc
```

---

## 13. Monitoring & Observability

### 13.1 Metrics (Prometheus)

| Metric | Alert threshold |
|--------|-------------------|
| `api_request_duration_p99` | > 200ms |
| `stream_ingest_latency` | > 2s |
| `transcode_queue_depth` | > 50 jobs |
| `chat_message_latency_p99` | > 500ms |
| `websocket_connections` | Per-node capacity 80% |
| `cdn_segment_age` | > 10s |
| `error_rate_5xx` | > 1% |
| `db_connection_pool_usage` | > 80% |
| `redis_memory_usage` | > 80% |

### 13.2 Logging

```go
// Structured logging format (zap)
{
  "level": "info",
  "ts": "2026-06-12T10:30:00Z",
  "service": "stream-service",
  "trace_id": "abc123",
  "span_id": "def456",
  "user_id": "uuid",
  "stream_id": "uuid",
  "action": "stream.started",
  "duration_ms": 45
}
```

- **Centralized:** Loki yoki ELK stack
- **PII masking:** Email, phone loglarda maskelanadi
- **Retention:** 30 kun hot, 1 yil cold (S3)

### 13.3 Tracing (OpenTelemetry)

- Distributed tracing across all services
- Critical paths: stream start, playback URL generation, chat message
- Jaeger UI for trace visualization

### 13.4 Alerting

| Severity | Channel | Response time |
|----------|---------|---------------|
| Critical (P1) | PagerDuty + Slack | 15 min |
| High (P2) | Slack | 1 hour |
| Medium (P3) | Slack | 4 hours |
| Low (P4) | Ticket | Next business day |

---

## 14. Disaster Recovery & HA

### 14.1 High Availability

| Komponent | HA strategiya |
|-----------|---------------|
| API Gateway | 3+ replicas, multi-AZ |
| Services | 2+ replicas, pod anti-affinity |
| PostgreSQL | Primary + 2 replicas (streaming replication) |
| Redis | Cluster mode, 6 nodes (3 master + 3 replica) |
| NATS | JetStream cluster, 3 nodes, R3 replication |
| CDN | Multi-CDN (primary + fallback) |
| Object Storage | Cross-region replication |

### 14.2 Backup

| Data | Frequency | Retention | Method |
|------|-----------|-----------|--------|
| PostgreSQL | Har 1 soat (WAL) + kunlik full | 30 kun | pgBackRest → S3 |
| Redis | Kunlik RDB snapshot | 7 kun | S3 |
| Object storage | Cross-region replication | Forever | S3 CRR |
| Config/secrets | Git (encrypted) | Forever | SOPS + age |

### 14.3 Disaster Recovery

- **RTO (Recovery Time Objective):** 30 daqiqa
- **RPO (Recovery Point Objective):** 1 soat
- **Failover:** DNS-based (Route53 health checks)
- **Runbook:** Har bir failure scenario uchun documented procedure
- **DR drill:** Har chorakda bir marta

---

## 15. Loyiha strukturasi

```
sahiy-stream/
├── docs/
│   ├── ARCHITECTURE.md              # Bu hujjat
│   ├── API.md                       # API documentation
│   ├── DEPLOYMENT.md                # Deploy qo'llanmasi
│   └── RUNBOOKS.md                    # Incident response
│
├── proto/                           # Shared gRPC definitions
│   ├── auth/v1/
│   ├── stream/v1/
│   ├── chat/v1/
│   ├── user/v1/
│   ├── media/v1/
│   └── common/v1/
│
├── pkg/                             # Shared Go packages
│   ├── auth/
│   ├── database/
│   ├── redis/
│   ├── nats/
│   ├── logger/
│   ├── metrics/
│   ├── tracing/
│   ├── errors/
│   ├── validator/
│   ├── crypto/
│   └── grpc/
│
├── services/                        # Go microservices
│   ├── api-gateway/
│   ├── auth-service/
│   ├── user-service/
│   ├── stream-service/
│   ├── media-orchestrator/
│   ├── chat-service/
│   ├── moderation-service/
│   ├── notification-service/
│   ├── analytics-service/
│   ├── search-service/
│   ├── billing-service/
│   └── admin-service/
│
├── media/                           # Media processing
│   ├── ingest/                      # RTMP/SRT ingest server
│   ├── transcode/                   # FFmpeg worker
│   ├── packager/                    # HLS/DASH packager
│   └── vod-processor/               # VOD post-processing
│
├── frontend/                        # Next.js application
│   ├── app/
│   ├── components/
│   ├── lib/
│   ├── hooks/
│   ├── stores/
│   └── types/
│
├── infra/                           # Infrastructure as Code
│   ├── terraform/                   # Cloud resources
│   ├── kubernetes/                  # K8s manifests
│   │   ├── base/
│   │   ├── staging/
│   │   └── production/
│   ├── docker/
│   │   └── docker-compose.yml       # Local dev
│   └── helm/                        # Helm charts
│
├── scripts/
│   ├── migrate.sh
│   ├── seed.sh
│   └── load-test/
│
├── .github/
│   └── workflows/
│       ├── ci.yml
│       ├── deploy-staging.yml
│       └── deploy-production.yml
│
├── go.work                          # Go workspace
├── Makefile                         # Build automation
└── README.md
```

---

## 16. Implementation tartibi

> MVP yo'q — lekin qurish tartibi muhim. Har bir bosqich production-ready.

### Bosqich 1: Foundation (Asos)

| # | Vazifa | Natija |
|---|--------|--------|
| 1.1 | Monorepo struktura, Go workspace, Makefile | Loyiha skeleton |
| 1.2 | Docker Compose (PostgreSQL, Redis, NATS, MinIO) | Local dev environment |
| 1.3 | `pkg/` shared packages (logger, database, redis, errors) | Shared foundation |
| 1.4 | Proto definitions (auth, user, stream) | gRPC contracts |
| 1.5 | Database migrations (users, channels, streams) | Schema ready |
| 1.6 | `auth-service` (register, login, JWT, refresh) | Authentication works |
| 1.7 | `api-gateway` (routing, auth middleware, rate limit) | API accessible |
| 1.8 | CI pipeline (lint, test, build) | Automated quality |

### Bosqich 2: Core Platform (Yadro)

| # | Vazifa | Natija |
|---|--------|--------|
| 2.1 | `user-service` (profile, channel CRUD) | Users & channels |
| 2.2 | `stream-service` (create, schedule, lifecycle) | Stream management |
| 2.3 | Stream key generation & validation | Secure ingest |
| 2.4 | Frontend: auth pages, layout, routing | UI foundation |
| 2.5 | Frontend: channel page, profile settings | Channel UI |
| 2.6 | Frontend: creator studio (basic) | Streamer dashboard |

### Bosqich 3: Media Pipeline (Eng muhim)

| # | Vazifa | Natija |
|---|--------|--------|
| 3.1 | RTMP ingest server | Stream qabul qilish |
| 3.2 | `media-orchestrator` (job dispatch, health) | Transcode boshqaruvi |
| 3.3 | Transcode workers (FFmpeg, GPU support) | Multi-bitrate output |
| 3.4 | HLS packager (CMAF, LL-HLS) | Playlist generation |
| 3.5 | Origin storage (MinIO/S3) | Segment storage |
| 3.6 | CDN integration (signed URLs) | Global delivery |
| 3.7 | Frontend: LivePlayer (hls.js, ABR) | Playback works |
| 3.8 | Frontend: watch page (live/[id]) | Stream ko'rish |
| 3.9 | DVR buffer (live rewind) | Seek during live |
| 3.10 | VOD processing (auto-record, finalize) | VOD ready |

### Bosqich 4: Real-time (Jonli o'zaro ta'sir)

| # | Vazifa | Natija |
|---|--------|--------|
| 4.1 | `chat-service` (WebSocket, sharding) | Real-time chat |
| 4.2 | Chat moderation (word filter, rate limit) | Safe chat |
| 4.3 | Viewer count (HyperLogLog + broadcast) | Live viewer count |
| 4.4 | Frontend: ChatPanel, emotes, mod tools | Chat UI |
| 4.5 | Follow/notification system | "Went live" alerts |
| 4.6 | `notification-service` (push, email) | Multi-channel notify |

### Bosqich 5: Discovery & Social

| # | Vazifa | Natija |
|---|--------|--------|
| 5.1 | `search-service` (Elasticsearch) | Full-text search |
| 5.2 | Home feed, categories, trending | Content discovery |
| 5.3 | Frontend: browse, search, trending pages | Discovery UI |
| 5.4 | Followers, subscriptions | Social features |
| 5.5 | Frontend: home feed, subscription feed | Personalized feed |

### Bosqich 6: Quality & Performance

| # | Vazifa | Natija |
|---|--------|--------|
| 6.1 | LL-HLS optimization (partial segments) | 3-5s latency |
| 6.2 | WebRTC/WHIP ingest (ultra-low mode) | <2s latency option |
| 6.3 | 4K/H.265 transcoding tier | Premium quality |
| 6.4 | Multi-CDN strategy | Global performance |
| 6.5 | Load testing (k6, 100K concurrent) | Proven scale |

### Bosqich 7: Moderation & Safety

| # | Vazifa | Natija |
|---|--------|--------|
| 7.1 | `moderation-service` (auto-mod, ML) | Content safety |
| 7.2 | Report system, mod queue | Community reporting |
| 7.3 | Ban/timeout/strike system | Enforcement |
| 7.4 | Admin panel (user management, audit) | Platform governance |

### Bosqich 8: Analytics & Monetization

| # | Vazifa | Natija |
|---|--------|--------|
| 8.1 | `analytics-service` (ClickHouse) | Event tracking |
| 8.2 | Streamer analytics dashboard | Creator insights |
| 8.3 | `billing-service` (Stripe integration) | Payments |
| 8.4 | Subscriptions, super chat | Monetization |
| 8.5 | Frontend: analytics dashboard, revenue | Creator earnings |

### Bosqich 9: Production Hardening

| # | Vazifa | Natija |
|---|--------|--------|
| 9.1 | Kubernetes production manifests | K8s deployment |
| 9.2 | Monitoring (Prometheus, Grafana, alerts) | Full observability |
| 9.3 | Security audit (penetration test) | Verified security |
| 9.4 | Disaster recovery setup | HA validated |
| 9.5 | Performance optimization (p99 < 100ms) | SLO met |
| 9.6 | Documentation (API docs, runbooks) | Ops ready |

---

## Texnologiya xulosasi

| Qatlam | Texnologiya |
|--------|-------------|
| **Backend** | Go 1.22+, Fiber/chi, gRPC, sqlc |
| **Frontend** | Next.js 15, TypeScript, Tailwind, shadcn/ui |
| **Player** | hls.js (LL-HLS), WebRTC WHEP |
| **Database** | PostgreSQL 16, Redis 7, ClickHouse |
| **Search** | Elasticsearch 8 |
| **Queue** | NATS JetStream |
| **Storage** | S3/MinIO |
| **Media** | FFmpeg, NVIDIA NVENC, LiveKit SFU |
| **CDN** | Cloudflare / multi-CDN |
| **Auth** | Custom JWT + OAuth2 + 2FA |
| **Payments** | Stripe |
| **Infra** | Docker, Kubernetes, Terraform |
| **CI/CD** | GitHub Actions |
| **Monitoring** | Prometheus, Grafana, Jaeger, Loki |
| **Security** | mTLS, WAF, signed URLs, encryption |

---

*Bu hujjat Sahiy Stream platformasining to'liq ideal arxitektura rejasidir. Har bir bosqich production-grade va shu rejaga qat'iy amal qilinadi.*
