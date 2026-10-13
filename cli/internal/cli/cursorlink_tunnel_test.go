package cli

import "testing"

func TestParseTryCloudflareURL(t *testing.T) {
	t.Parallel()
	line := `2026-04-12T12:00:00Z INF  |  https://random-words-here.trycloudflare.com`
	got := parseTryCloudflareURL(line)
	want := "https://random-words-here.trycloudflare.com"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if parseTryCloudflareURL("no url here") != "" {
		t.Fatal("expected empty")
	}
}

func TestCursorLinkPublicBaseURL(t *testing.T) {
	t.Parallel()
	st := cursorLinkState{PublicOrigin: "https://abc.trycloudflare.com", AccessToken: "deadbeef"}
	if got := cursorLinkPublicBaseURL(st); got != "https://abc.trycloudflare.com/cl/deadbeef/v1" {
		t.Fatalf("got %q", got)
	}
	st.PublicBaseURL = "https://abc.trycloudflare.com/cl/deadbeef/v1"
	if got := cursorLinkPublicBaseURL(st); got != "https://abc.trycloudflare.com/cl/deadbeef/v1" {
		t.Fatalf("got %q", got)
	}
}
