package media

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Jovy-lqw/SanguoClip/internal/domain"
)

type Service struct {
	mediaDir  string
	exportDir string
}

type ProbeResult struct {
	Duration float64
	HasAudio bool
}

func NewService(dataDir string) (*Service, error) {
	service := &Service{
		mediaDir:  filepath.Join(dataDir, "media"),
		exportDir: filepath.Join(dataDir, "exports"),
	}
	for _, dir := range []string{service.mediaDir, service.exportDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create media directory: %w", err)
		}
	}
	return service, nil
}

func (s *Service) MediaPath(fileName string) string {
	return filepath.Join(s.mediaDir, filepath.Base(fileName))
}

func (s *Service) ExportPath(fileName string) string {
	return filepath.Join(s.exportDir, filepath.Base(fileName))
}

func (s *Service) Probe(path string) (ProbeResult, error) {
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return ProbeResult{}, errors.New("未找到 ffprobe，素材已保存但无法读取时长")
	}
	output, err := exec.Command(ffprobe,
		"-v", "error",
		"-show_entries", "format=duration:stream=codec_type",
		"-of", "json",
		path,
	).CombinedOutput()
	if err != nil {
		return ProbeResult{}, fmt.Errorf("读取视频信息失败: %s", strings.TrimSpace(string(output)))
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return ProbeResult{}, fmt.Errorf("解析视频信息失败: %w", err)
	}
	duration, _ := strconv.ParseFloat(result.Format.Duration, 64)
	probe := ProbeResult{Duration: duration}
	for _, stream := range result.Streams {
		if stream.CodecType == "audio" {
			probe.HasAudio = true
			break
		}
	}
	return probe, nil
}

func (s *Service) Render(inputPath, outputName string, segments []domain.Segment, hasAudio bool) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return errors.New("未找到 FFmpeg，请先安装后再渲染")
	}
	if len(segments) == 0 {
		return errors.New("至少需要一个有效片段")
	}

	args := buildRenderArgs(inputPath, s.ExportPath(outputName), segments, hasAudio)
	output, err := exec.Command(ffmpeg, args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 1200 {
			message = message[len(message)-1200:]
		}
		return fmt.Errorf("FFmpeg 渲染失败: %s", message)
	}
	return nil
}

func buildRenderArgs(inputPath, outputPath string, segments []domain.Segment, hasAudio bool) []string {
	args := []string{"-hide_banner", "-y", "-i", inputPath}
	filterParts := make([]string, 0, len(segments)*2+1)
	videoInputs := make([]string, 0, len(segments))
	audioInputs := make([]string, 0, len(segments))
	for index, segment := range segments {
		videoLabel := fmt.Sprintf("v%d", index)
		filterParts = append(filterParts, fmt.Sprintf(
			"[0:v]trim=start=%s:end=%s,setpts=PTS-STARTPTS[%s]",
			formatSeconds(segment.Start), formatSeconds(segment.End), videoLabel,
		))
		videoInputs = append(videoInputs, "["+videoLabel+"]")
		if hasAudio {
			audioLabel := fmt.Sprintf("a%d", index)
			filterParts = append(filterParts, fmt.Sprintf(
				"[0:a]atrim=start=%s:end=%s,asetpts=PTS-STARTPTS[%s]",
				formatSeconds(segment.Start), formatSeconds(segment.End), audioLabel,
			))
			audioInputs = append(audioInputs, "["+audioLabel+"]")
		}
	}

	var concatInputs strings.Builder
	if hasAudio {
		for index := range videoInputs {
			concatInputs.WriteString(videoInputs[index])
			concatInputs.WriteString(audioInputs[index])
		}
		filterParts = append(filterParts, fmt.Sprintf(
			"%sconcat=n=%d:v=1:a=1[outv][outa]", concatInputs.String(), len(segments),
		))
	} else {
		concatInputs.WriteString(strings.Join(videoInputs, ""))
		filterParts = append(filterParts, fmt.Sprintf(
			"%sconcat=n=%d:v=1:a=0[outv]", concatInputs.String(), len(segments),
		))
	}

	args = append(args, "-filter_complex", strings.Join(filterParts, ";"), "-map", "[outv]")
	if hasAudio {
		args = append(args, "-map", "[outa]", "-c:a", "aac", "-b:a", "160k")
	}
	args = append(args,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "20",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		outputPath,
	)
	return args
}

func formatSeconds(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}
