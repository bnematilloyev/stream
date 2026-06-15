package playback

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

var tagURIAttr = regexp.MustCompile(`URI="([^"]+)"`)

// QueryForResource returns signed query params for an HLS resource at a fixed expiry.
func (s *Signer) QueryForResource(streamID, resource string, exp int64) string {
	return "exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.compute(streamID, resource, exp)
}

// RewriteManifest appends signed query params to relative URIs inside an HLS playlist.
func RewriteManifest(body []byte, baseResource string, queryFor func(resource string) string) []byte {
	lines := strings.Split(string(body), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			out = append(out, rewriteTagLine(line, baseResource, queryFor))
			continue
		}
		out = append(out, appendSignedQuery(trimmed, queryFor(resolveHLSResource(baseResource, trimmed))))
	}
	return []byte(strings.Join(out, "\n"))
}

func rewriteTagLine(line, baseResource string, queryFor func(resource string) string) string {
	return tagURIAttr.ReplaceAllStringFunc(line, func(match string) string {
		sub := tagURIAttr.FindStringSubmatch(match)
		if len(sub) != 2 {
			return match
		}
		uri := sub[1]
		if isAbsoluteURI(uri) {
			return match
		}
		res := resolveHLSResource(baseResource, uri)
		return `URI="` + appendSignedQuery(uri, queryFor(res)) + `"`
	})
}

func resolveHLSResource(baseResource, uri string) string {
	baseDir := path.Dir(normalizeResource(baseResource))
	if baseDir == "." {
		baseDir = ""
	}
	joined := path.Clean(path.Join(baseDir, uri))
	return strings.TrimPrefix(joined, "./")
}

func appendSignedQuery(uri, query string) string {
	if uri == "" || query == "" || isAbsoluteURI(uri) {
		return uri
	}
	if strings.Contains(uri, "?") {
		return uri + "&" + query
	}
	return uri + "?" + query
}

func isAbsoluteURI(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}
