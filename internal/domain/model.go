package domain

import "time"

type State struct {
	Assets      []Asset     `json:"assets"`
	Captures    []Capture   `json:"captures"`
	Edits       []Edit      `json:"edits"`
	Styles      []Style     `json:"styles"`
	ModelConfig ModelConfig `json:"model_config"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Asset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	FileName  string    `json:"file_name"`
	Source    string    `json:"source"`
	Duration  float64   `json:"duration"`
	Size      int64     `json:"size"`
	HasAudio  bool      `json:"has_audio"`
	CaptureID string    `json:"capture_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Capture struct {
	ID           string    `json:"id"`
	Label        string    `json:"label"`
	DeviceSerial string    `json:"device_serial"`
	PackageName  string    `json:"package_name,omitempty"`
	ShareURL     string    `json:"share_url"`
	DeepLink     string    `json:"deep_link"`
	Status       string    `json:"status"`
	Error        string    `json:"error,omitempty"`
	AssetID      string    `json:"asset_id,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	StoppedAt    time.Time `json:"stopped_at"`
}

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Label string  `json:"label,omitempty"`
}

type Edit struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	AssetID      string    `json:"asset_id"`
	StyleID      string    `json:"style_id,omitempty"`
	Segments     []Segment `json:"segments"`
	Status       string    `json:"status"`
	OutputName   string    `json:"output_name,omitempty"`
	Error        string    `json:"error,omitempty"`
	ModelSummary string    `json:"model_summary,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Style struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Prompt          string    `json:"prompt"`
	ExampleAssetIDs []string  `json:"example_asset_ids"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ModelConfig struct {
	BaseURL     string    `json:"base_url"`
	Model       string    `json:"model"`
	APIKeyEnv   string    `json:"api_key_env"`
	Temperature float64   `json:"temperature"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Device struct {
	Serial  string       `json:"serial"`
	State   string       `json:"state"`
	Model   string       `json:"model,omitempty"`
	Clients []GameClient `json:"clients,omitempty"`
}

type GameClient struct {
	PackageName string `json:"package_name"`
	Activity    string `json:"activity"`
	Channel     string `json:"channel"`
}

type Environment struct {
	ADB      Dependency `json:"adb"`
	FFmpeg   Dependency `json:"ffmpeg"`
	FFprobe  Dependency `json:"ffprobe"`
	DataDir  string     `json:"data_dir"`
	Ready    bool       `json:"ready"`
	Warnings []string   `json:"warnings"`
}

type Dependency struct {
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
}
