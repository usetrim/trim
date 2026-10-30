package platformadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type billingBackboneOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Model string `json:"model"`
}

func validateBackboneCatalog(options []billingBackboneOption, defaultID string) error {
	if len(options) == 0 {
		return fmt.Errorf("empty catalog")
	}
	defaultID = strings.TrimSpace(defaultID)
	if defaultID == "" {
		return fmt.Errorf("empty default")
	}
	seen := map[string]struct{}{}
	found := false
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
			found = true
		}
	}
	if !found {
		return fmt.Errorf("default not in catalog")
	}
	return nil
}

func (h *Handler) validateBillingBackboneFields(ctx context.Context, patch map[string]json.RawMessage) error {
	var options []billingBackboneOption
	var defaultID string

	if raw, ok := patch["deep_v2_backbone_options"]; ok {
		if err := json.Unmarshal(raw, &options); err != nil {
			return err
		}
	} else {
		var optionsRaw []byte
		if err := h.DB.QueryRow(ctx, `
			select coalesce(deep_v2_backbone_options, 'null'::jsonb)
			from public.billing_settings where id = 'default'
		`).Scan(&optionsRaw); err != nil {
			return err
		}
		if err := json.Unmarshal(optionsRaw, &options); err != nil {
			return err
		}
	}

	if raw, ok := patch["default_deep_v2_backbone"]; ok {
		if err := json.Unmarshal(raw, &defaultID); err != nil {
			return err
		}
	} else {
		if err := h.DB.QueryRow(ctx, `
			select coalesce(default_deep_v2_backbone, '')
			from public.billing_settings where id = 'default'
		`).Scan(&defaultID); err != nil {
			return err
		}
	}
	return validateBackboneCatalog(options, defaultID)
}
