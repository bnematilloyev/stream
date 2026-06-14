package storage

import (
	"path/filepath"
	"strings"
)

// ContentType returns MIME type for HLS objects.
func ContentType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".m3u8":
		return "application/vnd.apple.mpegurl"
	case ".m4s", ".mp4":
		return "video/mp4"
	case ".ts":
		return "video/mp2t"
	default:
		return "application/octet-stream"
	}
}

func IsPlaylist(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".m3u8")
}

func IsSegment(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".ts" || ext == ".m4s" || ext == ".mp4"
}

func IsMediaFile(name string) bool {
	return IsPlaylist(name) || IsSegment(name)
}
