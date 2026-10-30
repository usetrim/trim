package deepopt

import "testing"

func TestValidateBackboneCatalogMicrosoft(t *testing.T) {
	opts := []V2BackboneOption{
		{
			ID:    "mbert-base",
			Label: "mbert-base",
			Model: "microsoft/llmlingua-2-bert-base-multilingual-cased-meetingbank",
		},
		{
			ID:    "xlm-roberta-large",
			Label: "xlm-roberta-large",
			Model: "microsoft/llmlingua-2-xlm-roberta-large-meetingbank",
		},
	}
	if err := ValidateBackboneCatalog(opts, "mbert-base"); err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if !BackboneInCatalog(opts, "xlm-roberta-large") {
		t.Fatal("expected xlm in catalog")
	}
	m, ok := ResolveV2Model(opts, "mbert-base")
	if !ok || m != "microsoft/llmlingua-2-bert-base-multilingual-cased-meetingbank" {
		t.Fatalf("resolve mbert: %q %v", m, ok)
	}
	m2, ok2 := ResolveV2Model(opts, "xlm-roberta-large")
	if !ok2 || m2 != "microsoft/llmlingua-2-xlm-roberta-large-meetingbank" {
		t.Fatalf("resolve xlm: %q %v", m2, ok2)
	}
}
