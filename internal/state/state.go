package state

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"monitor/internal/config"
	"monitor/internal/utils"
	"os"
)

// State stores the persisted monitoring data.
type State struct {
	Storage Storage
}

// Load loads persisted state from disk or returns an empty state when none exists.
func Load(cfg *config.Config) *State {
	if _, err := os.Stat(cfg.StatePath); err != nil {
		if os.IsNotExist(err) {
			return &State{
				Storage: *NewStorage(),
			}
		}

		panic("load state error: check state file error: " + err.Error())
	}

	file, err := os.Open(cfg.StatePath)

	if err != nil {
		panic("load state error: load state file error: " + err.Error())
	}

	defer file.Close()

	var state State

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&state)

	if err != nil {
		panic("load state error: decode state file error: " + err.Error())
	}

	return &state
}

// Save encodes the state and writes it to disk atomically.
func (s State) Save(cfg *config.Config) error {
	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	err := enc.Encode(s)

	if err != nil {
		return fmt.Errorf("save state error: encode state file error: %w", err)
	}

	err = utils.AtomicWrite(
		cfg.StatePath,
		cfg.StatePathTmp,
		buf.Bytes(),
		cfg.FilePerm,
	)

	if err != nil {
		return fmt.Errorf("save state error: write state file error: %w", err)
	}

	return nil
}
