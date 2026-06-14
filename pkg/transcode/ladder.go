package transcode

// Tier defines one rung on the adaptive bitrate ladder (YouTube/Twitch-grade).
type Tier struct {
	Name    string
	Width   int
	Height  int
	Bitrate string
	Maxrate string
	Bufsize string
	AudioBR string
}

// DefaultLadder — 1080p → 360p, 30fps, tuned for live ABR + LL-HLS.
var DefaultLadder = []Tier{
	{Name: "1080p", Width: 1920, Height: 1080, Bitrate: "5500k", Maxrate: "5800k", Bufsize: "8250k", AudioBR: "160k"},
	{Name: "720p", Width: 1280, Height: 720, Bitrate: "3200k", Maxrate: "3400k", Bufsize: "4800k", AudioBR: "128k"},
	{Name: "480p", Width: 854, Height: 480, Bitrate: "1600k", Maxrate: "1700k", Bufsize: "2400k", AudioBR: "128k"},
	{Name: "360p", Width: 640, Height: 360, Bitrate: "800k", Maxrate: "850k", Bufsize: "1200k", AudioBR: "96k"},
}
