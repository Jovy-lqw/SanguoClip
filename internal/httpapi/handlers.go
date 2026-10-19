package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Jovy-lqw/SanguoClip/internal/capture"
	"github.com/Jovy-lqw/SanguoClip/internal/domain"
)

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (s *Server) handleParseShare(response http.ResponseWriter, request *http.Request) {
	var input struct {
		URL string `json:"url"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	parsed, err := capture.ParseShareURL(input.URL)
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	writeJSON(response, http.StatusOK, parsed)
}

func (s *Server) handleDevices(response http.ResponseWriter, _ *http.Request) {
	devices, err := capture.ListDevices()
	if err != nil {
		writeError(response, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"devices": devices})
}

func (s *Server) handleStartCapture(response http.ResponseWriter, request *http.Request) {
	var input struct {
		ShareURL     string `json:"share_url"`
		DeviceSerial string `json:"device_serial"`
		PackageName  string `json:"package_name"`
		Label        string `json:"label"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	parsed, err := capture.ParseShareURL(input.ShareURL)
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(input.DeviceSerial) == "" {
		writeError(response, http.StatusBadRequest, errors.New("请选择模拟器或安卓设备"))
		return
	}

	id := newID("capture")
	fileName := id + ".mp4"
	now := time.Now()
	item := domain.Capture{
		ID:           id,
		Label:        strings.TrimSpace(input.Label),
		DeviceSerial: input.DeviceSerial,
		PackageName:  strings.TrimSpace(input.PackageName),
		ShareURL:     parsed.ShareURL,
		DeepLink:     parsed.DeepLink,
		Status:       "starting",
		StartedAt:    now,
	}
	if item.Label == "" {
		item.Label = "对局录制 " + now.Format("01-02 15:04")
	}
	if err := s.store.Update(func(state *domain.State) error {
		state.Captures = append([]domain.Capture{item}, state.Captures...)
		return nil
	}); err != nil {
		writeError(response, http.StatusInternalServerError, err)
		return
	}

	if err := s.capture.Start(id, input.DeviceSerial, item.PackageName, parsed.DeepLink, s.media.MediaPath(fileName)); err != nil {
		_ = s.updateCapture(id, func(current *domain.Capture) {
			current.Status = "failed"
			current.Error = err.Error()
			current.StoppedAt = time.Now()
		})
		writeError(response, http.StatusBadGateway, err)
		return
	}
	item.Status = "recording"
	_ = s.updateCapture(id, func(current *domain.Capture) {
		current.Status = "recording"
	})
	writeJSON(response, http.StatusCreated, item)
}

func (s *Server) handleStopCapture(response http.ResponseWriter, request *http.Request) {
	var input struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	if input.ID == "" {
		writeError(response, http.StatusBadRequest, errors.New("缺少录制任务 ID"))
		return
	}

	path, err := s.capture.Stop(input.ID)
	if err != nil {
		_ = s.updateCapture(input.ID, func(current *domain.Capture) {
			current.Status = "failed"
			current.Error = err.Error()
			current.StoppedAt = time.Now()
		})
		writeError(response, http.StatusBadGateway, err)
		return
	}

	info, statErr := os.Stat(path)
	if statErr != nil {
		writeError(response, http.StatusInternalServerError, fmt.Errorf("录制文件不存在: %w", statErr))
		return
	}
	probe, probeErr := s.media.Probe(path)
	asset := domain.Asset{
		ID:        newID("asset"),
		Name:      "设备录制 " + time.Now().Format("2006-01-02 15:04"),
		FileName:  filepath.Base(path),
		Source:    "capture",
		Size:      info.Size(),
		CaptureID: input.ID,
		CreatedAt: time.Now(),
	}
	if probeErr == nil {
		asset.Duration = probe.Duration
		asset.HasAudio = probe.HasAudio
	}
	if err := s.store.Update(func(state *domain.State) error {
		state.Assets = append([]domain.Asset{asset}, state.Assets...)
		for index := range state.Captures {
			if state.Captures[index].ID == input.ID {
				state.Captures[index].Status = "completed"
				state.Captures[index].AssetID = asset.ID
				state.Captures[index].Error = ""
				state.Captures[index].StoppedAt = time.Now()
				return nil
			}
		}
		return errors.New("录制任务不存在")
	}); err != nil {
		writeError(response, http.StatusInternalServerError, err)
		return
	}
	result := map[string]any{"asset": asset}
	if probeErr != nil {
		result["warning"] = probeErr.Error()
	}
	writeJSON(response, http.StatusOK, result)
}

func (s *Server) updateCapture(id string, update func(*domain.Capture)) error {
	return s.store.Update(func(state *domain.State) error {
		for index := range state.Captures {
			if state.Captures[index].ID == id {
				update(&state.Captures[index])
				return nil
			}
		}
		return errors.New("录制任务不存在")
	})
}

func (s *Server) handleUpload(response http.ResponseWriter, request *http.Request) {
	reader, err := request.MultipartReader()
	if err != nil {
		writeError(response, http.StatusBadRequest, errors.New("请选择要上传的视频文件"))
		return
	}

	var source io.Reader
	var originalName string
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			writeError(response, http.StatusBadRequest, fmt.Errorf("读取上传内容失败: %w", nextErr))
			return
		}
		if part.FormName() == "file" && part.FileName() != "" {
			source = part
			originalName = cleanName(part.FileName())
			break
		}
		_ = part.Close()
	}
	if source == nil {
		writeError(response, http.StatusBadRequest, errors.New("上传内容中没有视频文件"))
		return
	}

	extension := strings.ToLower(filepath.Ext(originalName))
	switch extension {
	case ".mp4", ".mov", ".mkv", ".webm", ".m4v":
	default:
		writeError(response, http.StatusBadRequest, errors.New("仅支持 MP4、MOV、MKV、WebM 或 M4V 视频"))
		return
	}

	id := newID("asset")
	fileName := id + extension
	path := s.media.MediaPath(fileName)
	target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		writeError(response, http.StatusInternalServerError, fmt.Errorf("创建素材文件失败: %w", err))
		return
	}
	size, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		writeError(response, http.StatusInternalServerError, errors.New("保存上传视频失败"))
		return
	}

	probe, probeErr := s.media.Probe(path)
	asset := domain.Asset{
		ID:        id,
		Name:      originalName,
		FileName:  fileName,
		Source:    "upload",
		Duration:  probe.Duration,
		Size:      size,
		HasAudio:  probe.HasAudio,
		CreatedAt: time.Now(),
	}
	if err := s.store.Update(func(state *domain.State) error {
		state.Assets = append([]domain.Asset{asset}, state.Assets...)
		return nil
	}); err != nil {
		_ = os.Remove(path)
		writeError(response, http.StatusInternalServerError, err)
		return
	}

	result := map[string]any{"asset": asset}
	if probeErr != nil {
		result["warning"] = probeErr.Error()
	}
	writeJSON(response, http.StatusCreated, result)
}

func (s *Server) handleMedia(response http.ResponseWriter, request *http.Request) {
	asset, err := findAsset(s.store.Snapshot(), request.PathValue("id"))
	if err != nil {
		writeError(response, http.StatusNotFound, err)
		return
	}
	response.Header().Set("Content-Disposition", `inline; filename="`+safeHeaderName(asset.Name)+`"`)
	http.ServeFile(response, request, s.media.MediaPath(asset.FileName))
}

func (s *Server) handleCreateEdit(response http.ResponseWriter, request *http.Request) {
	var input struct {
		Title        string           `json:"title"`
		AssetID      string           `json:"asset_id"`
		StyleID      string           `json:"style_id"`
		Segments     []domain.Segment `json:"segments"`
		ModelSummary string           `json:"model_summary"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	state := s.store.Snapshot()
	asset, err := findAsset(state, input.AssetID)
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	if input.StyleID != "" {
		if _, err := findStyle(state, input.StyleID); err != nil {
			writeError(response, http.StatusBadRequest, err)
			return
		}
	}
	if err := validateSegments(input.Segments, asset.Duration); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}

	now := time.Now()
	edit := domain.Edit{
		ID:           newID("edit"),
		Title:        strings.TrimSpace(input.Title),
		AssetID:      input.AssetID,
		StyleID:      input.StyleID,
		Segments:     input.Segments,
		Status:       "draft",
		ModelSummary: strings.TrimSpace(input.ModelSummary),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if edit.Title == "" {
		edit.Title = strings.TrimSuffix(asset.Name, filepath.Ext(asset.Name)) + " 剪辑"
	}
	if err := s.store.Update(func(state *domain.State) error {
		state.Edits = append([]domain.Edit{edit}, state.Edits...)
		return nil
	}); err != nil {
		writeError(response, http.StatusInternalServerError, err)
		return
	}
	writeJSON(response, http.StatusCreated, edit)
}

func (s *Server) handleSuggestEdit(response http.ResponseWriter, request *http.Request) {
	var input struct {
		AssetID string `json:"asset_id"`
		StyleID string `json:"style_id"`
		Notes   string `json:"notes"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	state := s.store.Snapshot()
	asset, err := findAsset(state, input.AssetID)
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	if asset.Duration <= 0 {
		writeError(response, http.StatusBadRequest, errors.New("素材时长未知，请安装 FFmpeg 后重新上传素材"))
		return
	}
	style, err := findStyle(state, input.StyleID)
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(input.Notes) == "" {
		writeError(response, http.StatusBadRequest, errors.New("请填写关键事件或希望保留的时间点"))
		return
	}
	suggestion, err := s.model.Suggest(request.Context(), state.ModelConfig, style, asset.Duration, input.Notes)
	if err != nil {
		writeError(response, http.StatusBadGateway, err)
		return
	}
	writeJSON(response, http.StatusOK, suggestion)
}

func (s *Server) handleRenderEdit(response http.ResponseWriter, request *http.Request) {
	var input struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	state := s.store.Snapshot()
	edit, err := findEdit(state, input.ID)
	if err != nil {
		writeError(response, http.StatusNotFound, err)
		return
	}
	if edit.Status == "rendering" {
		writeError(response, http.StatusConflict, errors.New("该方案正在渲染"))
		return
	}
	asset, err := findAsset(state, edit.AssetID)
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	outputName := edit.ID + ".mp4"
	if err := s.updateEdit(edit.ID, func(current *domain.Edit) {
		current.Status = "rendering"
		current.Error = ""
		current.OutputName = ""
		current.UpdatedAt = time.Now()
	}); err != nil {
		writeError(response, http.StatusInternalServerError, err)
		return
	}

	go func() {
		renderErr := s.media.Render(
			s.media.MediaPath(asset.FileName),
			outputName,
			edit.Segments,
			asset.HasAudio,
		)
		if renderErr != nil {
			s.logger.Error("render failed", "edit_id", edit.ID, "error", renderErr)
			_ = s.updateEdit(edit.ID, func(current *domain.Edit) {
				current.Status = "failed"
				current.Error = renderErr.Error()
				current.UpdatedAt = time.Now()
			})
			return
		}
		_ = s.updateEdit(edit.ID, func(current *domain.Edit) {
			current.Status = "completed"
			current.OutputName = outputName
			current.Error = ""
			current.UpdatedAt = time.Now()
		})
	}()
	writeJSON(response, http.StatusAccepted, map[string]string{"id": edit.ID, "status": "rendering"})
}

func (s *Server) handleExport(response http.ResponseWriter, request *http.Request) {
	edit, err := findEdit(s.store.Snapshot(), request.PathValue("id"))
	if err != nil {
		writeError(response, http.StatusNotFound, err)
		return
	}
	if edit.Status != "completed" || edit.OutputName == "" {
		writeError(response, http.StatusConflict, errors.New("视频尚未渲染完成"))
		return
	}
	disposition := "inline"
	if request.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	response.Header().Set("Content-Disposition", disposition+`; filename="`+safeHeaderName(edit.Title)+`.mp4"`)
	http.ServeFile(response, request, s.media.ExportPath(edit.OutputName))
}

func (s *Server) updateEdit(id string, update func(*domain.Edit)) error {
	return s.store.Update(func(state *domain.State) error {
		for index := range state.Edits {
			if state.Edits[index].ID == id {
				update(&state.Edits[index])
				return nil
			}
		}
		return errors.New("剪辑方案不存在")
	})
}

func (s *Server) handleCreateStyle(response http.ResponseWriter, request *http.Request) {
	var input domain.Style
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	if err := validateStyle(&input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	now := time.Now()
	input.ID = newID("style")
	input.CreatedAt = now
	input.UpdatedAt = now
	if err := s.store.Update(func(state *domain.State) error {
		if err := validateExampleAssets(*state, input.ExampleAssetIDs); err != nil {
			return err
		}
		state.Styles = append([]domain.Style{input}, state.Styles...)
		return nil
	}); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	writeJSON(response, http.StatusCreated, input)
}

func (s *Server) handleUpdateStyle(response http.ResponseWriter, request *http.Request) {
	var input domain.Style
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	if err := validateStyle(&input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	id := request.PathValue("id")
	var updated domain.Style
	if err := s.store.Update(func(state *domain.State) error {
		if err := validateExampleAssets(*state, input.ExampleAssetIDs); err != nil {
			return err
		}
		for index := range state.Styles {
			if state.Styles[index].ID == id {
				createdAt := state.Styles[index].CreatedAt
				input.ID = id
				input.CreatedAt = createdAt
				input.UpdatedAt = time.Now()
				state.Styles[index] = input
				updated = input
				return nil
			}
		}
		return errors.New("剪辑风格不存在")
	}); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	writeJSON(response, http.StatusOK, updated)
}

func (s *Server) handleUpdateModel(response http.ResponseWriter, request *http.Request) {
	var input domain.ModelConfig
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	parsed, err := url.Parse(strings.TrimSpace(input.BaseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		writeError(response, http.StatusBadRequest, errors.New("模型接口地址无效"))
		return
	}
	if strings.TrimSpace(input.Model) == "" {
		writeError(response, http.StatusBadRequest, errors.New("模型名称不能为空"))
		return
	}
	if input.APIKeyEnv != "" && !envNamePattern.MatchString(input.APIKeyEnv) {
		writeError(response, http.StatusBadRequest, errors.New("API Key 环境变量名无效"))
		return
	}
	if input.Temperature < 0 || input.Temperature > 2 {
		writeError(response, http.StatusBadRequest, errors.New("温度必须在 0 到 2 之间"))
		return
	}
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	input.Model = strings.TrimSpace(input.Model)
	input.APIKeyEnv = strings.TrimSpace(input.APIKeyEnv)
	input.UpdatedAt = time.Now()
	if err := s.store.Update(func(state *domain.State) error {
		state.ModelConfig = input
		return nil
	}); err != nil {
		writeError(response, http.StatusInternalServerError, err)
		return
	}
	writeJSON(response, http.StatusOK, input)
}

func validateStyle(style *domain.Style) error {
	style.Name = strings.TrimSpace(style.Name)
	style.Description = strings.TrimSpace(style.Description)
	style.Prompt = strings.TrimSpace(style.Prompt)
	if style.Name == "" {
		return errors.New("风格名称不能为空")
	}
	if style.Prompt == "" {
		return errors.New("风格要求不能为空")
	}
	style.ExampleAssetIDs = uniqueStrings(style.ExampleAssetIDs)
	return nil
}

func validateExampleAssets(state domain.State, ids []string) error {
	for _, id := range ids {
		if _, err := findAsset(state, id); err != nil {
			return fmt.Errorf("示例素材 %s 不存在", id)
		}
	}
	return nil
}

func validateSegments(segments []domain.Segment, duration float64) error {
	if len(segments) == 0 {
		return errors.New("至少添加一个保留片段")
	}
	sort.SliceStable(segments, func(i, j int) bool {
		return segments[i].Start < segments[j].Start
	})
	var previousEnd float64
	for index := range segments {
		segment := &segments[index]
		segment.Label = strings.TrimSpace(segment.Label)
		if segment.Start < 0 || segment.End <= segment.Start {
			return fmt.Errorf("第 %d 个片段的开始或结束时间无效", index+1)
		}
		if duration > 0 && segment.End > duration+0.001 {
			return fmt.Errorf("第 %d 个片段超过素材时长", index+1)
		}
		if index > 0 && segment.Start < previousEnd {
			return fmt.Errorf("第 %d 个片段与前一片段重叠", index+1)
		}
		previousEnd = segment.End
	}
	return nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func safeHeaderName(value string) string {
	replacer := strings.NewReplacer("\r", "", "\n", "", `"`, "'")
	return replacer.Replace(cleanName(value))
}
