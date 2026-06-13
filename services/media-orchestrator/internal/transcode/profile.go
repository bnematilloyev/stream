package transcode

// Profile controls encoder packaging — latency vs compatibility.
type Profile struct {
	SegmentSec  float64
	PartSec     float64
	UseLLHLS    bool
	UseFMP4     bool
	Preset      string
	GOP         int
	AudioRate   int
}

func LLHLSProfile() Profile {
	return Profile{
		SegmentSec: 2.0,
		PartSec:    0.33, // ~330ms partials → 2–4s end-to-end with player
		UseLLHLS:   true,
		UseFMP4:    true,
		Preset:     "veryfast",
		GOP:        60,
		AudioRate:  48000,
	}
}

func StandardProfile() Profile {
	return Profile{
		SegmentSec: 4.0,
		PartSec:    0,
		UseLLHLS:   false,
		UseFMP4:    false,
		Preset:     "fast",
		GOP:        120,
		AudioRate:  48000,
	}
}
