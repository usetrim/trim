package proxy

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/usetrim/trim/server/pkg/localchrome"
	"github.com/usetrim/trim/server/pkg/trimmer"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	st := s.Stats()
	ch := s.opts.Chrome
	daysStr := fmt.Sprintf("%d", s.opts.DefaultSeriesDays)
	if daysStr == "0" {
		daysStr = ""
	}
	seriesLead := localchrome.ReplaceDays(ch.SeriesLead, daysStr)
	savedPct := 0.0
	if st.TokensBefore > 0 {
		savedPct = float64(st.TokensBefore-st.TokensAfter) / float64(st.TokensBefore) * 100
	}
	lastPct := 0.0
	if st.LastBeforeTokens > 0 {
		lastPct = float64(st.LastBeforeTokens-st.LastAfterTokens) / float64(st.LastBeforeTokens) * 100
	}
	beforeEsc := dashboardEscape(st.LastBeforePreview, ch.NoRequestYet)
	afterEsc := dashboardEscape(st.LastAfterPreview, ch.NoRequestYet)
	mode := strings.TrimSpace(s.opts.LiveModeLabel)
	if mode == "" {
		mode = trimmer.NormalizeMode(s.opts.CompressionMode)
	}
	chromeJSON, _ := json.Marshal(ch)
	leadHTML := ""
	if ch.LeadFmt != "" {
		// Escape the formatted lead, but re-inject a bold mode without breaking <<<MODE>>> /
		// %s substitution (html.EscapeString would turn <<<MODE>>> into &lt;&lt;&lt;MODE&gt;&gt;&gt;).
		const modeToken = "\x00TRIM_MODE\x00"
		rawLead := fmt.Sprintf(ch.LeadFmt, modeToken)
		if !strings.Contains(rawLead, modeToken) {
			rawLead = strings.Replace(ch.LeadFmt, "<<<MODE>>>", modeToken, 1)
		}
		leadHTML = html.EscapeString(rawLead)
		leadHTML = strings.ReplaceAll(leadHTML, html.EscapeString(modeToken), "<strong>"+html.EscapeString(mode)+"</strong>")
	}

	seriesDays := s.opts.DefaultSeriesDays
	if seriesDays < 1 {
		seriesDays = 0
	}
	seriesHref := "/v1/stats/series"
	if seriesDays > 0 {
		seriesHref = fmt.Sprintf("/v1/stats/series?days=%d", seriesDays)
	}
	skelRows := seriesDays
	if skelRows < 1 {
		skelRows = 0
	}
	if skelRows > 31 {
		skelRows = 31
	}
	var skelBuilder strings.Builder
	skelBuilder.WriteString(`<table class="series-skel-table" role="presentation">
        <tr>
          <td style="width:18%"><div class="shimmer w80"></div></td>
          <td style="width:14%"><div class="shimmer w60"></div></td>
          <td style="width:14%"><div class="shimmer w60"></div></td>
          <td style="width:14%"><div class="shimmer w60"></div></td>
          <td style="width:16%"><div class="shimmer w40"></div></td>
          <td style="width:14%"><div class="shimmer w40"></div></td>
        </tr>`)
	for i := 0; i < skelRows; i++ {
		wClass := "w80"
		switch i % 3 {
		case 1:
			wClass = "w60"
		case 2:
			wClass = "w40"
		}
		skelBuilder.WriteString(fmt.Sprintf(`
        <tr><td colspan="6"><div class="shimmer %s"></div></td></tr>`, wClass))
	}
	skelBuilder.WriteString(`
      </table>`)
	// Fprintf treats % as verbs; double any literal percent in injected HTML.
	seriesSkeletonHTML := strings.ReplaceAll(skelBuilder.String(), "%", "%%")

	estSavedHTML := ""
	if ch.TUIEstSavedFmt != "" {
		estSavedHTML = html.EscapeString(fmt.Sprintf(ch.TUIEstSavedFmt, st.SavedUSD))
	}
	latencyHTML := ""
	if ch.TUILatencyFmt != "" {
		latencyHTML = html.EscapeString(fmt.Sprintf(ch.TUILatencyFmt, st.LastLatency))
	}
	lastReqHTML := ""
	if ch.LastRequestValueFmt != "" {
		lastReqHTML = html.EscapeString(fmt.Sprintf(ch.LastRequestValueFmt, st.LastBeforeTokens, st.LastAfterTokens))
	}
	deepStageHTML := ""
	if ch.LastRequestValueFmt != "" {
		deepStageHTML = html.EscapeString(fmt.Sprintf(ch.LastRequestValueFmt, st.LastDeepStageBefore, st.LastDeepStageAfter))
	}
	deepStatusLabel := st.LastDeepStatus
	if ch.DeepStatusLabels != nil {
		if mapped := strings.TrimSpace(ch.DeepStatusLabels[st.LastDeepStatus]); mapped != "" {
			deepStatusLabel = mapped
		}
	}
	deepStatusEsc := html.EscapeString(deepStatusLabel)
	wireDetail := strings.TrimSpace(ch.SavingsDetailWireFmt)
	wireDetail = strings.ReplaceAll(wireDetail, "{before}", fmt.Sprintf("%d", st.LastBeforeTokens))
	wireDetail = strings.ReplaceAll(wireDetail, "{after}", fmt.Sprintf("%d", st.LastAfterTokens))
	wireDetail = strings.ReplaceAll(wireDetail, "{pct}", fmt.Sprintf("%.1f", lastPct))
	stageDetail := strings.TrimSpace(ch.SavingsDetailStageFmt)
	stageDetail = strings.ReplaceAll(stageDetail, "{before}", fmt.Sprintf("%d", st.LastDeepStageBefore))
	stageDetail = strings.ReplaceAll(stageDetail, "{after}", fmt.Sprintf("%d", st.LastDeepStageAfter))
	stageDetail = strings.ReplaceAll(stageDetail, "{pct}", fmt.Sprintf("%.1f", st.LastDeepStageSavedPct))
	statusDetail := strings.TrimSpace(ch.SavingsDetailStatusFmt)
	statusDetail = strings.ReplaceAll(statusDetail, "{status}", deepStatusLabel)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	langAttr := ""
	if strings.TrimSpace(ch.HtmlLang) != "" {
		langAttr = ` lang="` + html.EscapeString(strings.TrimSpace(ch.HtmlLang)) + `"`
	}
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html%s>
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>%s</title>
<style>
  :root { color-scheme: dark; --bg:#09090b; --card:#18181b; --fg:#fafafa; --muted:#a1a1aa; --accent:#e4e4e7; --border:#27272a; }
  * { box-sizing: border-box; }
  body { margin:0; font-family: ui-sans-serif, system-ui, sans-serif; background: radial-gradient(1200px 600px at 10%% -10%%, #27272a 0%%, var(--bg) 55%%); color: var(--fg); min-height: 100vh; }
  main { max-width: 1400px; margin: 0 auto; padding: 48px 24px; }
  h1.brand { font-size: 0; letter-spacing: 0; margin: 0 0 12px; line-height: 0; }
  h1.brand .brand-mark {
    display: block; height: 36px; width: auto;
    /* Aspect matches apps web TrimWordmark (~0.988). */
  }
  h2 { font-size: 1rem; font-weight: 600; margin: 0 0 12px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.06em; }
  p.lead { color: var(--muted); margin: 0 0 32px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 16px; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 12px; padding: 20px; min-width: 0; }
  .card-span-2 { grid-column: span 2; }
  @media (max-width: 520px) { .card-span-2 { grid-column: span 1; } }
  .label { color: var(--muted); font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.06em; }
  .value { font-size: 1.75rem; font-weight: 600; margin-top: 8px; line-height: 1.2; }
  /* Prose / status strings: stay readable inside narrow cards without overflowing. */
  .value-text {
    font-size: clamp(0.8125rem, 0.7rem + 0.55vw, 1.05rem);
    font-weight: 600;
    line-height: 1.35;
    overflow-wrap: anywhere;
    word-break: break-word;
    hyphens: auto;
  }
  .diff { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-top: 32px; }
  @media (max-width: 800px) { .diff { grid-template-columns: 1fr; } }
  pre { margin: 0; padding: 16px; background: #09090b; border: 1px solid var(--border); border-radius: 10px; font-size: 0.75rem; line-height: 1.45; overflow: auto; max-height: 360px; white-space: pre-wrap; word-break: break-word; color: #d4d4d8; }
  .foot { margin-top: 28px; color: var(--muted); font-size: 0.875rem; }
  a { color: var(--accent); }
  .playground { margin-top: 40px; }
  .playground .row { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; margin-bottom: 12px; }
  textarea, select, input[type="text"] {
    width: 100%%; background: #09090b; color: var(--fg); border: 1px solid var(--border);
    border-radius: 10px; padding: 12px; font-family: ui-monospace, monospace; font-size: 0.8rem;
  }
  select { width: auto; min-width: 160px; }
  button {
    background: var(--fg); color: var(--bg); border: none; border-radius: 8px;
    padding: 10px 16px; font-weight: 600; cursor: pointer;
  }
  button:disabled { opacity: 0.5; cursor: wait; }
  .meta { color: var(--muted); font-size: 0.875rem; margin: 8px 0 0; }
  .shimmer {
    position: relative; overflow: hidden; border-radius: 8px;
    background: linear-gradient(90deg, #18181b 0%%, #27272a 40%%, #18181b 80%%);
    background-size: 200%% 100%%; animation: shimmer 1.4s ease-in-out infinite;
    height: 12px; margin: 8px 0;
  }
  .shimmer.w60 { width: 60%%; } .shimmer.w80 { width: 80%%; } .shimmer.w40 { width: 40%%; }
  .series-skel-table { width: 100%%; border-collapse: collapse; font-family: ui-monospace, monospace; font-size: 0.8rem; }
  .series-skel-table td { padding: 8px 10px 8px 0; vertical-align: middle; }
  .series-skel-table .shimmer { margin: 0; height: 10px; }
  @keyframes shimmer { 0%% { background-position: 100%% 0; } 100%% { background-position: -100%% 0; } }
  #seriesSkeleton { padding: 8px 0; }
  #seriesSkeleton.hidden { display: none; }
  #seriesOut.hidden { display: none; }
  #savingsDetail.hidden { display: none; }
  .detail-box { margin-top: 24px; background: var(--card); border: 1px solid var(--border); border-radius: 12px; padding: 20px; }
  .detail-box p { color: var(--muted); margin: 0 0 10px; font-size: 0.9rem; }
  .detail-box ul { margin: 0; padding-left: 1.2rem; color: var(--fg); }
  .detail-box li { margin: 6px 0; }
  .detail-toggle { margin-top: 20px; }
</style>
</head>
<body>
<main>
  <h1 class="brand"><img class="brand-mark" src="%s" width="36" height="36" alt="%s"/></h1>
  <p class="lead">%s</p>
  <div class="grid">
    <div class="card"><div class="label">%s</div><div class="value">%d</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%d</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%d</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%.1f%%</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%s</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%s</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%s</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%.1f%%</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%d</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%s</div></div>
    <div class="card card-span-2"><div class="label">%s</div><div class="value value-text" title="%s">%s</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%s</div></div>
    <div class="card"><div class="label">%s</div><div class="value">%.1f%%</div></div>
  </div>
  <div class="detail-toggle">
    <button type="button" id="savingsDetailBtn">%s</button>
  </div>
  <div class="detail-box hidden" id="savingsDetail">
    <h2>%s</h2>
    <p>%s</p>
    <ul>
      <li>%s</li>
      <li>%s</li>
      <li>%s</li>
    </ul>
    <p style="margin-top:14px">%s</p>
  </div>
  <div class="diff">
    <div>
      <h2>%s</h2>
      <pre>%s</pre>
    </div>
    <div>
      <h2>%s</h2>
      <pre>%s</pre>
    </div>
  </div>
  <section class="playground">
    <h2>%s</h2>
    <p class="lead" style="margin-bottom:16px">%s</p>
    <div class="row">
      <label class="label">%s
        <select id="mode">
          <option value="mild">%s</option>
          <option value="balanced">%s</option>
          <option value="aggressive">%s</option>
          <option value="custom">%s</option>
        </select>
      </label>
      <label class="label">%s
        <input type="text" id="userPrompt" placeholder="%s" style="width:280px"/>
      </label>
      <button type="button" id="previewBtn">%s</button>
    </div>
    <textarea id="input" rows="10" placeholder="%s"></textarea>
    <p class="meta" id="previewMeta"></p>
    <div class="diff">
      <div>
        <h2>%s</h2>
        <pre id="previewIn">%s</pre>
      </div>
      <div>
        <h2>%s</h2>
        <pre id="previewOut">%s</pre>
      </div>
    </div>
  </section>
  <p class="foot">%s: <a href="/v1/stats">/v1/stats</a> · %s: <a href="%s">%s</a> · %s: <a href="/v1/trim/preview">POST /v1/trim/preview</a> · %s: <a href="/metrics">/metrics</a> · %s: <a href="/health">/health</a></p>
  <section class="playground" id="seriesSection" style="margin-top:32px">
    <h2>%s</h2>
    <p class="lead" style="margin-bottom:12px">%s</p>
    <div id="seriesSkeleton" aria-hidden="true">
      %s
    </div>
    <pre id="seriesOut" class="hidden"></pre>
  </section>
</main>
<script>
var CHROME = %s;
(function(){
  function showSeries(text) {
    var sk = document.getElementById("seriesSkeleton");
    var out = document.getElementById("seriesOut");
    if (sk) sk.classList.add("hidden");
    if (out) {
      out.classList.remove("hidden");
      out.textContent = text;
    }
  }
  var seriesHref = %q;
  fetch(seriesHref).then(function(r){ return r.json(); }).then(function(data){
    var items = data.items || [];
    var c = data.chrome || CHROME;
    if (!items.length) {
      showSeries(data.message || c.series_empty || "");
      return;
    }
    var lines = [c.series_header || ""];
    function applyGoFloatFmt(fmt, n) {
      if (!fmt) return "";
      // %% becomes a single percent for JS; avoids Go printf verbs in this template.
      var reSrc = "%%" + "\\.(\\d+)f";
      var m = fmt.match(new RegExp(reSrc));
      var digits = m ? parseInt(m[1], 10) : 2;
      var num = Number(n || 0).toFixed(digits);
      return fmt.replace(new RegExp("%%" + "\\.\\d+f"), num);
    }
    for (var i = 0; i < items.length; i++) {
      var p = items[i];
      var usd = applyGoFloatFmt(c.tui_est_saved_fmt || "", p.saved_usd_est);
      var lat = applyGoFloatFmt(c.tui_latency_fmt || "", p.avg_latency_ms);
      lines.push(
        (p.day || "") + " | " + (p.requests || 0) + " | " + (p.tokens_before || 0) + " | " +
        (p.tokens_after || 0) + " | " + usd + " | " + lat
      );
    }
    showSeries(lines.join("\n"));
  }).catch(function(e){
    var fmt = CHROME.series_unavailable_fmt || "";
    showSeries(fmt.split("{error}").join(String(e)));
  });
  var modeEl = document.getElementById("mode");
  modeEl.value = %q;
  (function(){
    var btn = document.getElementById("savingsDetailBtn");
    var box = document.getElementById("savingsDetail");
    if (!btn || !box) return;
    var show = CHROME.savings_detail_show || "";
    var hide = CHROME.savings_detail_hide || "";
    btn.addEventListener("click", function() {
      var open = !box.classList.contains("hidden");
      if (open) {
        box.classList.add("hidden");
        btn.textContent = show;
      } else {
        box.classList.remove("hidden");
        btn.textContent = hide;
      }
    });
  })();
  document.getElementById("previewBtn").addEventListener("click", async function() {
    var btn = this;
    btn.disabled = true;
    btn.textContent = CHROME.preview_pending || "";
    var text = document.getElementById("input").value;
    var body = {
      text: text,
      user_prompt: document.getElementById("userPrompt").value,
      mode: modeEl.value,
      logs_only: modeEl.value === "custom" && false
    };
    try {
      var res = await fetch("/v1/trim/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body)
      });
      var data = await res.json();
      var empty = CHROME.empty_preview || "";
      document.getElementById("previewIn").textContent = text || empty;
      document.getElementById("previewOut").textContent = data.output || empty;
      var meta = CHROME.preview_meta_fmt || "";
      meta = meta.split("{mode}").join(String(data.mode || ""));
      meta = meta.split("{before}").join(String(data.before_tokens || 0));
      meta = meta.split("{after}").join(String(data.after_tokens || 0));
      meta = meta.split("{pct}").join(Number(data.saved_percent || 0).toFixed(1));
      document.getElementById("previewMeta").textContent = meta;
    } catch (e) {
      var failFmt = CHROME.preview_failed_fmt || "";
      document.getElementById("previewMeta").textContent = failFmt.split("{error}").join(String(e));
    } finally {
      btn.disabled = false;
      btn.textContent = CHROME.preview_action || "";
    }
  });
})();
</script>
</body>
</html>`,
		langAttr,
		html.EscapeString(ch.DocumentTitle),
		brandMarkWhiteURL,
		html.EscapeString(ch.Brand),
		leadHTML,
		html.EscapeString(ch.LabelRequests), st.Requests,
		html.EscapeString(ch.LabelTokensIn), st.TokensBefore,
		html.EscapeString(ch.LabelTokensOut), st.TokensAfter,
		html.EscapeString(ch.LabelSaved), savedPct,
		html.EscapeString(ch.LabelEstUSD), estSavedHTML,
		html.EscapeString(ch.LabelLastLatency), latencyHTML,
		html.EscapeString(ch.LabelLastRequest), lastReqHTML,
		html.EscapeString(ch.LabelLastSaved), lastPct,
		html.EscapeString(ch.LabelFallbacks), st.Fallbacks,
		html.EscapeString(ch.LabelLastDoor), html.EscapeString(st.LastDoor),
		html.EscapeString(ch.LabelDeepStatus), deepStatusEsc, deepStatusEsc,
		html.EscapeString(ch.LabelDeepStage), deepStageHTML,
		html.EscapeString(ch.LabelDeepStageSaved), st.LastDeepStageSavedPct,
		html.EscapeString(ch.SavingsDetailShow),
		html.EscapeString(ch.SavingsDetailTitle),
		html.EscapeString(ch.SavingsDetailLead),
		html.EscapeString(wireDetail),
		html.EscapeString(stageDetail),
		html.EscapeString(statusDetail),
		html.EscapeString(ch.SavingsDetailChromeTip),
		html.EscapeString(ch.HeadingBefore), beforeEsc,
		html.EscapeString(ch.HeadingAfter), afterEsc,
		html.EscapeString(ch.PlaygroundTitle),
		html.EscapeString(ch.PlaygroundLead),
		html.EscapeString(ch.LabelMode),
		html.EscapeString(ch.ModeMild),
		html.EscapeString(ch.ModeBalanced),
		html.EscapeString(ch.ModeAggressive),
		html.EscapeString(ch.ModeCustom),
		html.EscapeString(ch.LabelUserPrompt),
		html.EscapeString(ch.UserPromptPlaceholder),
		html.EscapeString(ch.PreviewAction),
		html.EscapeString(ch.InputPlaceholder),
		html.EscapeString(ch.HeadingPreviewIn),
		html.EscapeString(ch.PreviewIdle),
		html.EscapeString(ch.HeadingPreviewOut),
		html.EscapeString(ch.PreviewIdle),
		html.EscapeString(ch.FootJSON),
		html.EscapeString(ch.FootSeries),
		html.EscapeString(seriesHref),
		html.EscapeString(seriesHref),
		html.EscapeString(ch.FootPreview),
		html.EscapeString(ch.FootMetrics),
		html.EscapeString(ch.FootHealth),
		html.EscapeString(ch.SeriesTitle),
		html.EscapeString(seriesLead),
		seriesSkeletonHTML,
		string(chromeJSON),
		seriesHref,
		mode,
	)
}

func dashboardEscape(s, empty string) string {
	if s == "" {
		return html.EscapeString(empty)
	}
	return html.EscapeString(s)
}
