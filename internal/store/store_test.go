package store

import (
	"errors"
	"testing"

	"github.com/Jovy-lqw/SanguoClip/internal/domain"
)

func TestStorePersistsUpdates(t *testing.T) {
	dataDir := t.TempDir()
	first, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if len(first.Snapshot().Styles) != 1 {
		t.Fatalf("default styles = %d, want 1", len(first.Snapshot().Styles))
	}

	err = first.Update(func(state *domain.State) error {
		state.Assets = append(state.Assets, domain.Asset{ID: "asset-test", Name: "test.mp4"})
		return nil
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	reopened, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open() after update error = %v", err)
	}
	if got := reopened.Snapshot().Assets; len(got) != 1 || got[0].ID != "asset-test" {
		t.Fatalf("assets = %#v, want persisted asset", got)
	}
}

func TestStoreRollsBackRejectedUpdate(t *testing.T) {
	stateStore, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	rejected := errors.New("reject")
	err = stateStore.Update(func(state *domain.State) error {
		state.Assets = append(state.Assets, domain.Asset{ID: "should-not-exist"})
		return rejected
	})
	if !errors.Is(err, rejected) {
		t.Fatalf("Update() error = %v, want %v", err, rejected)
	}
	if len(stateStore.Snapshot().Assets) != 0 {
		t.Fatal("rejected update modified the store")
	}
}
