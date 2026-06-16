# OBS Stable Stream Settings

These settings are tuned for Sahiy Stream RTMP ingest and HLS playback stability.

## Recommended OBS Output

Open `Settings -> Output -> Output Mode: Advanced -> Streaming`.

Use this as the first stable preset:

| Setting | Value |
|---|---|
| Encoder | `x264` or hardware H.264 (`Apple VT H264`, `NVENC H.264`) |
| Rate Control | `CBR` |
| Bitrate | `3500 Kbps` for 720p30, `4500 Kbps` for 1080p30 |
| Keyframe Interval | `2 s` |
| CPU Usage Preset | `veryfast` or `superfast` |
| Profile | `high` |
| Tune | `zerolatency` if available |
| B-frames | `0` |

## Video

Open `Settings -> Video`.

| Setting | Value |
|---|---|
| Base Canvas | Your monitor/camera size |
| Output Scaled Resolution | `1280x720` first; use `1920x1080` only after stable |
| FPS | `30` |
| Downscale Filter | `Bicubic` or `Lanczos` |

## Advanced

Open `Settings -> Advanced`.

| Setting | Value |
|---|---|
| Process Priority | `Above Normal` |
| Color Format | `NV12` |
| Color Space | `Rec. 709` |
| Color Range | `Partial` |
| Network | Enable `Dynamically change bitrate to manage congestion` |

## Quick Debug Checklist

- OBS bottom-right should stay green.
- `Dropped Frames (Network)` should stay near `0`.
- CPU usage should stay below `70%`.
- Upload speed should be at least `2x` your OBS bitrate.
- Start with `720p30 / 3500 Kbps`; only move to 1080p after 10 minutes without buffering.

For weak networks use `2500 Kbps`, `720p30`, `veryfast`, keyframe interval `2`.
