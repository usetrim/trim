package deepopt

import (
	"fmt"
	"strings"
)

// V2BackboneOption is one Microsoft LLMLingua-2 Base Model catalog row from billing.
type V2BackboneOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Model string `json:"model"`
}

// ValidateBackboneCatalog fail-closes when the billing catalog is incomplete.
func ValidateBackboneCatalog(options []V2BackboneOption, defaultID string) error {
	if len(options) == 0 {
		return fmt.Errorf("deep_v2_backbone_options empty")
	}
	defaultID = strings.TrimSpace(defaultID)
	if defaultID == "" {
		return fmt.Errorf("default_deep_v2_backbone empty")
	}
	seen := map[string]struct{}{}
	foundDefault := false
	for _, o := range options {
		id := strings.TrimSpace(o.ID)
		label := strings.TrimSpace(o.Label)
		model := strings.TrimSpace(o.Model)
		if id == "" || label == "" || model == "" {
			return fmt.Errorf("deep_v2_backbone_options entry incomplete")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("deep_v2_backbone_options duplicate id %q", id)
		}
		seen[id] = struct{}{}
		if id == defaultID {
			foundDefault = true
		}
	}
	if !foundDefault {
		return fmt.Errorf("default_deep_v2_backbone %q not in catalog", defaultID)
	}
	return nil
}

// BackboneInCatalog reports whether id is present in the catalog.
func BackboneInCatalog(options []V2BackboneOption, id string) bool {
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

// ResolveV2Model returns the Hugging Face model id for the selected backbone.
func ResolveV2Model(options []V2BackboneOption, selected string) (string, bool) {
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

// BackboneLabel returns the display label for id, or id when unknown.
func BackboneLabel(options []V2BackboneOption, id string) string {
	id = strings.TrimSpace(id)
	for _, o := range options {
		if strings.TrimSpace(o.ID) == id {
			if lab := strings.TrimSpace(o.Label); lab != "" {
				return lab
			}
			return id
		}
	}
	return id
}
