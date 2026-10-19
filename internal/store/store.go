package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Jovy-lqw/SanguoClip/internal/domain"
)

type Store struct {
	mu       sync.RWMutex
	filePath string
	state    domain.State
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	s := &Store{filePath: filepath.Join(dataDir, "state.json")}
	data, err := os.ReadFile(s.filePath)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &s.state); err != nil {
			return nil, fmt.Errorf("decode state: %w", err)
		}
	case errors.Is(err, os.ErrNotExist):
		now := time.Now()
		s.state = domain.State{
			Styles: []domain.Style{{
				ID:          "default-highlight",
				Name:        "高光速剪",
				Description: "保留关键决策和胜负转折，去除等待与重复操作。",
				Prompt:      "优先保留出牌决策、身份判断、关键技能发动和胜负转折。节奏紧凑，不改变事件先后顺序。",
				CreatedAt:   now,
				UpdatedAt:   now,
			}},
			ModelConfig: domain.ModelConfig{
				BaseURL:     "https://api.openai.com/v1",
				APIKeyEnv:   "SANGUOCLIP_MODEL_API_KEY",
				Temperature: 0.2,
				UpdatedAt:   now,
			},
			UpdatedAt: now,
		}
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("read state: %w", err)
	}
	if normalize(&s.state) {
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) Snapshot() domain.State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clone(s.state)
}

func (s *Store) Update(fn func(*domain.State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := clone(s.state)
	if err := fn(&next); err != nil {
		return err
	}
	next.UpdatedAt = time.Now()
	previous := s.state
	s.state = next
	if err := s.saveLocked(); err != nil {
		s.state = previous
		return err
	}
	return nil
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.filePath), ".state-*.json")
	if err != nil {
		return fmt.Errorf("create temporary state: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("protect temporary state: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporary state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary state: %w", err)
	}
	if err := replaceFile(tmpName, s.filePath); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

func clone(value domain.State) domain.State {
	data, _ := json.Marshal(value)
	var copied domain.State
	_ = json.Unmarshal(data, &copied)
	return copied
}

func normalize(state *domain.State) bool {
	changed := false
	if state.Assets == nil {
		state.Assets = []domain.Asset{}
		changed = true
	}
	if state.Captures == nil {
		state.Captures = []domain.Capture{}
		changed = true
	}
	if state.Edits == nil {
		state.Edits = []domain.Edit{}
		changed = true
	}
	if state.Styles == nil {
		state.Styles = []domain.Style{}
		changed = true
	}
	for index := range state.Styles {
		if state.Styles[index].ExampleAssetIDs == nil {
			state.Styles[index].ExampleAssetIDs = []string{}
			changed = true
		}
	}
	return changed
}
