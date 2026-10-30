package account

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DeepV2BackboneOption is one Microsoft LLMLingua-2 Base Model catalog row.
type DeepV2BackboneOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Model string `json:"model"`
}

func parseBackboneOptions(raw []byte) ([]DeepV2BackboneOption, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("empty")
	}
	var options []DeepV2BackboneOption
	if err := json.Unmarshal(raw, &options); err != nil {
		return nil, err
	}
	return options, nil
}

func validateBackboneCatalog(options []DeepV2BackboneOption, defaultID string) error {
	if len(options) == 0 {
		return fmt.Errorf("empty catalog")
	}
	defaultID = strings.TrimSpace(defaultID)
	if defaultID == "" {
		return fmt.Errorf("empty default")
	}
	seen := map[string]struct{}{}
	foundDefault := false
	for _, o := range options {
		id := strings.TrimSpace(o.ID)
		label := strings.TrimSpace(o.Label)
		model := strings.TrimSpace(o.Model)
		if id == "" || label == "" || model == "" {
			return fmt.Errorf("incomplete entry")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate id")
		}
		seen[id] = struct{}{}
		if id == defaultID {
			foundDefault = true
		}
	}
	if !foundDefault {
		return fmt.Errorf("default missing from catalog")
	}
	return nil
}

func backboneInCatalog(options []DeepV2BackboneOption, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, o := range options {
		if strings.TrimSpace(o.ID) == id {
			return true
		}
	}
	return false
}

func resolveV2Model(options []DeepV2BackboneOption, selected string) (string, bool) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return "", false
	}
	for _, o := range options {
		if strings.TrimSpace(o.ID) == selected {
			m := strings.TrimSpace(o.Model)
			return m, m != ""
		}
	}
	return "", false
}

func defaultModelFromCatalog(options []DeepV2BackboneOption, defaultID string) (string, bool) {
	return resolveV2Model(options, defaultID)
}
