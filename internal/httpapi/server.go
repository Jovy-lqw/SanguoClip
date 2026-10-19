package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Jovy-lqw/SanguoClip/internal/capture"
	"github.com/Jovy-lqw/SanguoClip/internal/domain"
	"github.com/Jovy-lqw/SanguoClip/internal/media"
	modelclient "github.com/Jovy-lqw/SanguoClip/internal/model"
	"github.com/Jovy-lqw/SanguoClip/internal/store"
	webassets "github.com/Jovy-lqw/SanguoClip/web"
)

type Server struct {
	store    *store.Store
	capture  *capture.Manager
	media    *media.Service
	model    *modelclient.Client
	dataDir  string
	logger   *slog.Logger
	staticFS fs.FS
}

func New(dataDir string, logger *slog.Logger) (*Server, error) {
	stateStore, err := store.Open(dataDir)
	if err != nil {
		return nil, err
	}
	mediaService, err := media.NewService(dataDir)
	if err != nil {
		return nil, err
	}
	staticFS, err := fs.Sub(webassets.Content, ".")
	if err != nil {
		return nil, fmt.Errorf("prepare web assets: %w", err)
	}
	return &Server{
		store:    stateStore,
		capture:  capture.NewManager(),
		media:    mediaService,
		model:    modelclient.NewClient(),
		dataDir:  dataDir,
		logger:   logger,
		staticFS: staticFS,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("GET /api/environment", s.handleEnvironment)
	mux.HandleFunc("POST /api/share/parse", s.handleParseShare)
	mux.HandleFunc("GET /api/devices", s.handleDevices)
	mux.HandleFunc("POST /api/captures/start", s.handleStartCapture)
	mux.HandleFunc("POST /api/captures/stop", s.handleStopCapture)
	mux.HandleFunc("POST /api/media/upload", s.handleUpload)
	mux.HandleFunc("GET /api/media/{id}", s.handleMedia)
	mux.HandleFunc("POST /api/edits", s.handleCreateEdit)
	mux.HandleFunc("POST /api/edits/suggest", s.handleSuggestEdit)
	mux.HandleFunc("POST /api/edits/render", s.handleRenderEdit)
	mux.HandleFunc("GET /api/exports/{id}", s.handleExport)
	mux.HandleFunc("POST /api/styles", s.handleCreateStyle)
	mux.HandleFunc("PUT /api/styles/{id}", s.handleUpdateStyle)
	mux.HandleFunc("PUT /api/model", s.handleUpdateModel)
	mux.Handle("/", http.FileServer(http.FS(s.staticFS)))
	return s.withMiddleware(mux)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := time.Now()
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' https://unpkg.com; style-src 'self'; img-src 'self' data:; media-src 'self' blob:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		next.ServeHTTP(response, request)
		s.logger.Info("request",
			"method", request.Method,
			"path", request.URL.Path,
			"duration", time.Since(started).Round(time.Millisecond),
		)
	})
}

func (s *Server) handleHealth(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "SanguoClip",
		"time":    time.Now(),
	})
}

func (s *Server) handleState(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, s.store.Snapshot())
}

func (s *Server) handleEnvironment(response http.ResponseWriter, _ *http.Request) {
	environment := domain.Environment{
		ADB:     inspectDependency("adb"),
		FFmpeg:  inspectDependency("ffmpeg"),
		FFprobe: inspectDependency("ffprobe"),
		DataDir: s.dataDir,
	}
	if !environment.ADB.Available {
		environment.Warnings = append(environment.Warnings, "未安装 ADB：不能从模拟器自动采集，可先手动上传录像。")
	}
	if !environment.FFmpeg.Available || !environment.FFprobe.Available {
		environment.Warnings = append(environment.Warnings, "未完整安装 FFmpeg：可以编辑时间段，但暂时不能读取时长或导出视频。")
	}
	environment.Ready = environment.FFmpeg.Available && environment.FFprobe.Available
	writeJSON(response, http.StatusOK, environment)
}

func inspectDependency(name string) domain.Dependency {
	path, err := exec.LookPath(name)
	if err != nil {
		return domain.Dependency{}
	}
	versionArg := "-version"
	if name == "adb" {
		versionArg = "version"
	}
	output, _ := exec.Command(path, versionArg).CombinedOutput()
	version := strings.SplitN(strings.TrimSpace(string(output)), "\n", 2)[0]
	return domain.Dependency{Available: true, Path: path, Version: version}
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeError(response http.ResponseWriter, status int, err error) {
	writeJSON(response, status, map[string]string{"error": err.Error()})
}

func decodeJSON(request *http.Request, target any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("请求数据无效: %w", err)
	}
	return nil
}

func newID(prefix string) string {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(random)
}

func findAsset(state domain.State, id string) (domain.Asset, error) {
	for _, asset := range state.Assets {
		if asset.ID == id {
			return asset, nil
		}
	}
	return domain.Asset{}, errors.New("素材不存在")
}

func findStyle(state domain.State, id string) (domain.Style, error) {
	for _, style := range state.Styles {
		if style.ID == id {
			return style, nil
		}
	}
	return domain.Style{}, errors.New("剪辑风格不存在")
}

func findEdit(state domain.State, id string) (domain.Edit, error) {
	for _, edit := range state.Edits {
		if edit.ID == id {
			return edit, nil
		}
	}
	return domain.Edit{}, errors.New("剪辑方案不存在")
}

func cleanName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "." || name == "" {
		return "未命名素材"
	}
	return name
}
