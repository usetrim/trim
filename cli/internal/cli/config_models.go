package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/cliout"
	"github.com/usetrim/trim/cli/internal/deepopt"
)

var configModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		p, err := deepopt.LoadPreferences()
		if err != nil {
			return err
		}
		return printDeepV2ModelsFlow(p, chrome)
	},
}

// printDeepV2ModelsFlow mirrors trim config rates coloring for Microsoft Base Model catalog.
func printDeepV2ModelsFlow(p deepopt.Preferences, chrome cliChrome) error {
	title := strings.TrimSpace(chrome.ConfigModelsTitle)
	if title == "" {
		return chrome.fail("")
	}
	if !deepopt.RequireLiveDeepPolicy(p) ||
		deepopt.ValidateBackboneCatalog(p.DeepV2BackboneOptions, p.DefaultDeepV2Backbone) != nil {
		unsynced := strings.TrimSpace(chrome.ConfigModelsUnsynced)
		if unsynced != "" {
			cliout.Warn(unsynced)
			return nil
		}
		return chrome.fail("")
	}

	cliout.Success(title)

	defaultLab := strings.TrimSpace(chrome.ConfigModelsDefaultLabel)
	selectedLab := strings.TrimSpace(chrome.ConfigModelsSelectedLabel)
	selectedMark := strings.TrimSpace(chrome.ConfigModelsSelectedMark)
	if selectedMark == "" {
		selectedMark = "selected"
	}

	rows := make([]cliout.Row, 0, 4)
	if defaultLab != "" && strings.TrimSpace(p.DefaultDeepV2Backbone) != "" {
		rows = append(rows, cliout.Row{
			Label: defaultLab,
			Value: deepopt.BackboneLabel(p.DeepV2BackboneOptions, p.DefaultDeepV2Backbone),
			Kind:  "accent",
		})
	}
	if selectedLab != "" && strings.TrimSpace(p.DeepV2Backbone) != "" {
		rows = append(rows, cliout.Row{
			Label: selectedLab,
			Value: deepopt.BackboneLabel(p.DeepV2BackboneOptions, p.DeepV2Backbone),
			Kind:  "ok",
		})
	}
	if model, ok := deepopt.ResolveV2Model(p.DeepV2BackboneOptions, p.DeepV2Backbone); ok {
		hfLab := strings.TrimSpace(chrome.ConfigModelsHfLabel)
		if hfLab == "" {
			return chrome.fail("")
		}
		rows = append(rows, cliout.Row{Label: hfLab, Value: model})
	}
	cliout.Table(rows)

	listRows := make([]cliout.Row, 0, len(p.DeepV2BackboneOptions))
	for _, o := range p.DeepV2BackboneOptions {
		id := strings.TrimSpace(o.ID)
		label := strings.TrimSpace(o.Label)
		if label == "" {
			label = id
		}
		value := ""
		kind := ""
		isSelected := id == strings.TrimSpace(p.DeepV2Backbone)
		isDefault := id == strings.TrimSpace(p.DefaultDeepV2Backbone)
		switch {
		case isSelected:
			value = selectedMark
			kind = "ok"
		case isDefault:
			value = defaultLab
			kind = "accent"
		}
		listRows = append(listRows, cliout.Row{Label: label, Value: value, Kind: kind})
	}
	cliout.Table(listRows)

	if tip := strings.TrimSpace(chrome.ConfigModelsSyncHint); tip != "" {
		cliout.Muted(tip)
	}
	return nil
}

func init() {
	configCmd.AddCommand(configModelsCmd)
}
