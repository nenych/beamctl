package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type preset struct {
	Brightness *int `json:"brightness"` // front, percent
	Temp       *int `json:"temp"`       // front, kelvin
	Back       *struct {
		Color      *string `json:"color"`      // RRGGBB
		Brightness *int    `json:"brightness"` // percent
	} `json:"back"`
}

func presetsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "beamctl", "presets.json"), nil
}

// loadPresets returns no presets when the file does not exist.
func loadPresets() (map[string]preset, error) {
	path, err := presetsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var presets map[string]preset
	if err := json.Unmarshal(data, &presets); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return presets, nil
}

// applyPreset sets the given values and turns the affected lights on.
func applyPreset(p preset) error {
	if p.Brightness != nil {
		if err := setFrontBrightness(*p.Brightness); err != nil {
			return err
		}
	}
	if p.Temp != nil {
		if err := setFrontTemp(*p.Temp); err != nil {
			return err
		}
	}
	if err := setPower(featFront, fnFrontSetPower, true); err != nil {
		return err
	}
	if p.Back == nil {
		return nil
	}
	if p.Back.Color != nil {
		r, g, b, err := parseColor(*p.Back.Color)
		if err != nil {
			return err
		}
		if err := setBackColor(r, g, b); err != nil {
			return err
		}
	}
	if p.Back.Brightness != nil {
		if err := setBackBrightness(*p.Back.Brightness); err != nil {
			return err
		}
	}
	return setPower(featBack, fnBackSetPower, true)
}
