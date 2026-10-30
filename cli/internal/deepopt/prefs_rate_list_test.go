package deepopt

import "testing"

func TestListCompressionRatesMicrosoftStyle(t *testing.T) {
	got := ListCompressionRates(0.3, 1.0, 0.1)
	if len(got) < 8 {
		t.Fatalf("expected ladder, got %v", got)
	}
	if got[0] != 0.3 {
		t.Fatalf("min=%g", got[0])
	}
	if got[len(got)-1] != 1.0 {
		t.Fatalf("max=%g", got[len(got)-1])
	}
	for i := 1; i < len(got); i++ {
		if got[i] <= got[i-1] {
			t.Fatalf("not increasing: %v", got)
		}
	}
}

func TestListCompressionRatesRejectsBadBounds(t *testing.T) {
	if ListCompressionRates(0, 1, 0.1) != nil {
		t.Fatal("expected nil for min<=0")
	}
	if ListCompressionRates(0.3, 0.2, 0.1) != nil {
		t.Fatal("expected nil for max<min")
	}
}
