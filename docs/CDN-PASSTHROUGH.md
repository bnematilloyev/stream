# CDN + Passthrough HLS

Use this mode when OBS sends a stable H.264 video stream and you do not need adaptive bitrate transcoding.

## Runtime Mode

```env
TRANSCODE_MODE=passthrough
HLS_STORAGE_BACKEND=local
PLAYBACK_BASE_URL=https://api.stream.vibrant.uz
```

In this setup:

- OBS publishes RTMP to the ingest server.
- `media-orchestrator` starts FFmpeg in remux mode.
- FFmpeg copies video (`-c:v copy`) and writes HLS.
- `stream-service` serves signed `/playback/{streamID}/master.m3u8`.
- A CDN should cache `/playback/*` in front of `stream-service`.

## CDN Origin

Point CDN origin to:

```text
https://api.stream.vibrant.uz
```

Cache behavior:

| Path | Cache |
|---|---|
| `/playback/*.m3u8` | 1 second, stale while revalidate |
| `/playback/*.(ts|m4s|mp4)` | long cache, immutable |
| `/v1/streams/*/playback` | 5 seconds |
| `/v1/streams/live` | 2 seconds |

## When To Use GPU Again

Use `TRANSCODE_MODE=local` or `queue` only when you need:

- Multiple qualities: 1080p/720p/480p.
- Codec repair for bad OBS input.
- Strict bitrate/fps normalization.
- Large public streams with mixed viewer network quality.

For private/small streams with good OBS settings, passthrough is cheaper and more stable.
