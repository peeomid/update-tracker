package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	Items map[string]Item `json:"items"`
}

type Item struct {
	LastCheckedAt time.Time `json:"lastCheckedAt"`
	LastSeen      string    `json:"lastSeen"`
	LastStatus    string    `json:"lastStatus"`
	LastError     string    `json:"lastError,omitempty"`
}

func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{Items: map[string]Item{}}, nil
		}
		return State{}, fmt.Errorf("read state: %w", err)
	}
	if len(data) == 0 {
		return State{Items: map[string]Item{}}, nil
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, fmt.Errorf("parse state: %w", err)
	}
	if st.Items == nil {
		st.Items = map[string]Item{}
	}
	return st, nil
}

func Save(path string, st State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}
