package transcode_test

import (
	"testing"

	"github.com/sahiy/sahiy-stream/pkg/transcode"
)

func TestResolvePipelineProduction(t *testing.T) {
	profile, ladder := transcode.ResolvePipeline(transcode.QualityProduction, "ultra-low")
	if !profile.HighQuality {
		t.Fatal("production profile should be high quality")
	}
	if profile.Preset != "fast" {
		t.Fatalf("production LL-HLS preset want fast got %s", profile.Preset)
	}
	if len(ladder) != 3 {
		t.Fatalf("production ladder want 3 tiers got %d", len(ladder))
	}
	if ladder[0].Bitrate != "4500k" {
		t.Fatalf("1080p bitrate got %s", ladder[0].Bitrate)
	}
	if profile.PartSec != 0 {
		t.Fatalf("OBS-stable LL-HLS should not emit partial segment hints, got %f", profile.PartSec)
	}
}

func TestResolvePipelineBalanced(t *testing.T) {
	profile, ladder := transcode.ResolvePipeline(transcode.QualityBalanced, "ultra-low")
	if profile.HighQuality {
		t.Fatal("balanced profile should not set high quality")
	}
	if len(ladder) != 4 {
		t.Fatalf("balanced ladder want 4 tiers got %d", len(ladder))
	}
}

func TestNormalizeQuality(t *testing.T) {
	if transcode.NormalizeQuality("") != transcode.QualityProduction {
		t.Fatal("empty quality should default to production")
	}
	if transcode.NormalizeQuality("unknown") != transcode.QualityProduction {
		t.Fatal("unknown quality should default to production")
	}
}
