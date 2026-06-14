package transcode

const (
	QualityBalanced   = "balanced"
	QualityProduction = "production"
)

// ProductionLadder — 3-tier high-bitrate ladder for single-origin VPS (fewer viewers, better quality).
var ProductionLadder = []Tier{
	{Name: "1080p", Width: 1920, Height: 1080, Bitrate: "6500k", Maxrate: "7000k", Bufsize: "10500k", AudioBR: "192k"},
	{Name: "720p", Width: 1280, Height: 720, Bitrate: "4500k", Maxrate: "4800k", Bufsize: "6750k", AudioBR: "160k"},
	{Name: "480p", Width: 854, Height: 480, Bitrate: "2500k", Maxrate: "2700k", Bufsize: "3750k", AudioBR: "128k"},
}

// LadderForQuality picks the ABR ladder for the configured quality mode.
func LadderForQuality(quality string) []Tier {
	if quality == QualityProduction {
		return ProductionLadder
	}
	return DefaultLadder
}

// ResolvePipeline returns encoder profile and ladder for quality + latency mode.
func ResolvePipeline(quality, latencyMode string) (Profile, []Tier) {
	profile := profileForLatency(latencyMode)
	if quality == QualityProduction {
		profile.HighQuality = true
		if latencyMode != "standard" {
			profile.Preset = "fast"
		}
	}
	return profile, LadderForQuality(quality)
}

func profileForLatency(latencyMode string) Profile {
	if latencyMode == "standard" {
		return StandardProfile()
	}
	return LLHLSProfile()
}

// NormalizeQuality returns a known quality mode (default: production for VPS deploys).
func NormalizeQuality(quality string) string {
	switch quality {
	case QualityBalanced, QualityProduction:
		return quality
	default:
		return QualityProduction
	}
}
