package transcode

import "strconv"

// VideoEncoder selects the FFmpeg video codec pipeline.
type VideoEncoder struct {
	Codec string
}

func (e VideoEncoder) BaseArgs(profile Profile) []string {
	switch e.Codec {
	case "h264_nvenc":
		return []string{
			"-c:v", "h264_nvenc",
			"-preset", "p4",
			"-tune", "ll",
			"-profile:v", "high",
			"-pix_fmt", "yuv420p",
			"-g", strconv.Itoa(profile.GOP),
			"-bf", "0",
		}
	default:
		x264 := "force-cfr=1"
		if profile.HighQuality {
			x264 += ":aq-mode=3:psy-rd=1.0,0.15:deblock=1,1"
		}
		return []string{
			"-c:v", "libx264",
			"-preset", profile.Preset,
			"-tune", "zerolatency",
			"-profile:v", "high",
			"-pix_fmt", "yuv420p",
			"-g", strconv.Itoa(profile.GOP),
			"-keyint_min", strconv.Itoa(profile.GOP),
			"-sc_threshold", "0",
			"-bf", "0",
			"-x264-params", x264,
		}
	}
}
