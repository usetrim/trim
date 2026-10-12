package cliout

import (
	"bytes"
	"strings"
	"testing"
)

func TestTableNoColor(t *testing.T) {
	var buf bytes.Buffer
	prev := Writer
	Writer = &buf
	t.Cleanup(func() { Writer = prev })
	t.Setenv("NO_COLOR", "1")

	Table([]Row{
		{Label: "Plan", Value: "Team", Kind: "accent"},
		{Label: "Remaining", Value: "10 / 100", Kind: "bad"},
	})
	out := buf.String()
	if !strings.Contains(out, "Plan") || !strings.Contains(out, "Team") {
		t.Fatal(out)
	}
}

func TestTitleCaseWord(t *testing.T) {
	if TitleCaseWord("team") != "Team" {
		t.Fatal(TitleCaseWord("team"))
	}
}
