package hlsrecord

import (
	"os"
	"strings"
)

// FinalizePlaylist marks an HLS event playlist as complete VOD (#EXT-X-ENDLIST).
func FinalizePlaylist(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	if strings.Contains(content, "#EXT-X-ENDLIST") {
		return nil
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "#EXT-X-ENDLIST\n"
	return os.WriteFile(path, []byte(content), 0o644)
}
