package proxy

import (
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strings"
)

type cursorLinkDashState struct {
	Enabled       bool   `json:"enabled"`
	PublicBaseURL string `json:"public_base_url"`
	PublicOrigin  string `json:"public_origin"`
	AccessToken   string `json:"access_token"`
}

func loadCursorLinkDashboardHTML() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(home, ".trim", "cursor-link.json"))
	if err != nil {
		return cursorLinkDashOffHTML()
	}
	var st cursorLinkDashState
	if err := json.Unmarshal(b, &st); err != nil {
		return cursorLinkDashOffHTML()
	}
	if !st.Enabled {
		return cursorLinkDashOffHTML()
	}
	url := strings.TrimSpace(st.PublicBaseURL)
	if url == "" {
		origin := strings.TrimRight(strings.TrimSpace(st.PublicOrigin), "/")
		tok := strings.TrimSpace(st.AccessToken)
		if origin != "" && tok != "" {
			url = origin + "/cl/" + tok + "/v1"
		}
	}
	if url == "" {
		return `
  <section class="detail-box" style="margin-top:24px">
    <h2>Connect Cursor (Agent)</h2>
    <p>Cursor Link is <strong>on</strong>, but no public URL yet. Run <code>trim start</code>, then <code>trim cursor-link url</code>.</p>
  </section>`
	}
	esc := html.EscapeString(url)
	return `
  <section class="detail-box" style="margin-top:24px">
    <h2>Connect Cursor (Agent)</h2>
    <p>Paste this Base URL into Cursor → Settings → Models → Override OpenAI Base URL. Compression stays on this PC.</p>
    <p class="meta" style="margin:12px 0 8px">Cursor Agent Base URL</p>
    <pre id="cursorLinkUrl">` + esc + `</pre>
    <div class="row" style="margin-top:12px">
      <button type="button" id="cursorLinkCopyBtn">Copy Base URL</button>
    </div>
    <p class="meta" style="margin-top:12px">CLI: <code>trim cursor-link status</code> · rotate: <code>trim cursor-link rotate</code> · off: <code>trim cursor-link disable</code></p>
    <script>
      (function(){
        var btn = document.getElementById('cursorLinkCopyBtn');
        var el = document.getElementById('cursorLinkUrl');
        if (!btn || !el) return;
        btn.addEventListener('click', function(){
          var t = el.textContent || '';
          if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(t).then(function(){ btn.textContent = 'Copied'; setTimeout(function(){ btn.textContent = 'Copy Base URL'; }, 1600); });
          }
        });
      })();
    </script>
  </section>`
}

func cursorLinkDashOffHTML() string {
	return `
  <section class="detail-box" style="margin-top:24px">
    <h2>Connect Cursor (Agent)</h2>
    <p>Off. Cursor Agent cannot use <code>127.0.0.1</code> (private networks forbidden). Enable with:</p>
    <pre>trim cursor-link enable
trim start
trim cursor-link url</pre>
    <p class="meta" style="margin-top:12px">Local tools / Claude Code keep using localhost. Docs: Trim Cursor Link.</p>
  </section>`
}
