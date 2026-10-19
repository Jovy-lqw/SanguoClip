package media

import (
	"strings"
	"testing"

	"github.com/Jovy-lqw/SanguoClip/internal/domain"
)

func TestBuildRenderArgsInterleavesVideoAndAudio(t *testing.T) {
	args := buildRenderArgs("input.mp4", "output.mp4", []domain.Segment{
		{Start: 1, End: 3},
		{Start: 8, End: 12},
	}, true)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "[v0][a0][v1][a1]concat=n=2:v=1:a=1") {
		t.Fatalf("filter does not interleave video and audio inputs: %s", joined)
	}
	if !strings.Contains(joined, "-map [outv] -map [outa]") {
		t.Fatalf("audio render maps are missing: %s", joined)
	}
}

func TestBuildRenderArgsWithoutAudio(t *testing.T) {
	args := buildRenderArgs("input.mp4", "output.mp4", []domain.Segment{{Start: 0, End: 5}}, false)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "[v0]concat=n=1:v=1:a=0") {
		t.Fatalf("video-only concat filter is invalid: %s", joined)
	}
	if strings.Contains(joined, "[outa]") {
		t.Fatalf("video-only render unexpectedly includes audio map: %s", joined)
	}
}
