package deepopt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/usetrim/trim/cli/internal/clierr"
)

// Request is the JSON stdin payload for optimizer.py.
type Request struct {
	Text         string   `json:"text"`
	Question     string   `json:"question,omitempty"`
	TargetToken  int      `json:"target_token"`
	Engine       string   `json:"engine"`
	Rate         *float64 `json:"rate,omitempty"`
	ModelName    string   `json:"model_name,omitempty"`
	DeviceMap    string   `json:"device_map,omitempty"`
	ForceTokens  []string `json:"force_tokens,omitempty"`
	UseLLMLingua2 bool    `json:"use_llmlingua2,omitempty"`
}

// Runtime holds fail-closed Deep engine knobs from CLI env (no invent model ids).
type Runtime struct {
	DeviceMap     string
	V1Model       string
	V2Model       string
	LongModel     string
	V2ForceTokens []string
}

// codingAgentForceTokens are always merged into Deep force_tokens so LLMLingua
// is less likely to drop path/punctuation characters when Deep does run.
// Official LLMLingua-2 examples force \n . ! ? and related punctuation.
var codingAgentForceTokens = []string{"\n", "/", "\\", ".", "-", "_", ":", "?", "!", ",", "=", "+", "#", "@"}

func mergeForceTokens(base, extra []string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, s := range append(append([]string{}, base...), extra...) {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func (rt Runtime) ApplyEngine(req *Request, engine Engine) error {
	if req == nil {
		return fmt.Errorf("deep request nil")
	}
	dm := strings.TrimSpace(rt.DeviceMap)
	if dm == "" {
		return fmt.Errorf("TRIM_DEEP_DEVICE_MAP required")
	}
	req.DeviceMap = dm
	// Path/punctuation keepers for every engine (v1/long/v2) when Deep runs.
	req.ForceTokens = mergeForceTokens(rt.V2ForceTokens, codingAgentForceTokens)
	switch engine {
	case EngineV2:
		m := strings.TrimSpace(rt.V2Model)
		if m == "" {
			return fmt.Errorf("deep_v2_model required (trim config sync from billing_settings)")
		}
		req.ModelName = m
		req.UseLLMLingua2 = true
	case EngineLong:
		m := strings.TrimSpace(rt.LongModel)
		if m == "" {
			return fmt.Errorf("deep_long_model required (trim config sync from billing_settings)")
		}
		req.ModelName = m
	case EngineV1:
		m := strings.TrimSpace(rt.V1Model)
		if m == "" {
			return fmt.Errorf("deep_v1_model required (trim config sync from billing_settings)")
		}
		req.ModelName = m
	default:
		return fmt.Errorf("deep_engine must be v1, long, or v2")
	}
	return nil
}

// Response is the JSON stdout payload from optimizer.py.
type Response struct {
	CompressedPrompt string `json:"compressed_prompt"`
	OriginTokens     int    `json:"origin_tokens"`
	CompressedTokens int    `json:"compressed_tokens"`
	SavingRate       string `json:"saving_rate"`
	Engine           string `json:"engine"`
}

// UXChrome holds API-driven copy for Deep Mode. Empty strings mean fail-closed (no invent English).
type UXChrome struct {
	Unavailable         string
	BootstrapReqs       string
	BootstrapPip        string
	RequirementsMissing string
	AutoInstall         string
	BinEnvMissingFmt    string
	BinMissing          string
	PyEnvMissingFmt     string
	PyMissing           string
	LLMMissingFmt       string
	PipFailed           string
	TargetRequired      string
	EngineRequired      string
	QuestionRequired    string
	OptimizeFmt         string
	Timeout             string
	ParseFmt            string
}

func (ux UXChrome) err(msg string) error {
	if msg != "" {
		return fmt.Errorf("%s", msg)
	}
	if ux.Unavailable != "" {
		return fmt.Errorf("%s", ux.Unavailable)
	}
	return clierr.ErrChromeUnavailable
}

func (ux UXChrome) errFmt(fmtStr string, args ...any) error {
	if fmtStr != "" {
		return fmt.Errorf(fmtStr, args...)
	}
	return ux.err("")
}

// OptimizerBin resolves an optional frozen Deep Mode binary (PyInstaller trim-deep).
// Prefer TRIM_OPTIMIZER_BIN, then trim-deep / optimizer next to the CLI.
func OptimizerBin(ux UXChrome) (string, error) {
	if v := strings.TrimSpace(os.Getenv("TRIM_OPTIMIZER_BIN")); v != "" {
		if st, err := os.Stat(v); err == nil && !st.IsDir() {
			return v, nil
		}
		return "", ux.errFmt(ux.BinEnvMissingFmt, v)
	}
	names := []string{"trim-deep", "optimizer"}
	if runtime.GOOS == "windows" {
		names = []string{"trim-deep.exe", "optimizer.exe", "trim-deep", "optimizer"}
	}
	dirs := []string{}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd, filepath.Join(wd, "optimizer"), filepath.Join(wd, "cli", "optimizer", "dist"))
	}
	for _, dir := range dirs {
		for _, name := range names {
			p := filepath.Join(dir, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	return "", ux.err(ux.BinMissing)
}

// ScriptPath resolves optimizer.py next to the trim binary, or from TRIM_OPTIMIZER_PY,
// or from the source tree during local development.
func ScriptPath(ux UXChrome) (string, error) {
	if v := strings.TrimSpace(os.Getenv("TRIM_OPTIMIZER_PY")); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v, nil
		}
		return "", ux.errFmt(ux.PyEnvMissingFmt, v)
	}

	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "optimizer.py"),
			filepath.Join(dir, "optimizer", "optimizer.py"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "optimizer", "optimizer.py"),
			filepath.Join(wd, "cli", "optimizer", "optimizer.py"),
		)
	}
	candidates = append(candidates,
		"/usr/share/trim/optimizer.py",
		"/usr/local/share/trim/optimizer.py",
	)
	_, thisFile, _, _ := runtime.Caller(0)
	pkgDir := filepath.Dir(thisFile)
	candidates = append(candidates,
		filepath.Clean(filepath.Join(pkgDir, "..", "..", "optimizer", "optimizer.py")),
	)

	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", ux.err(ux.PyMissing)
}

func pythonBin() string {
	if v := strings.TrimSpace(os.Getenv("TRIM_PYTHON")); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("py"); err == nil {
			return "py"
		}
	}
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}
	return "python"
}

// Available reports whether Deep Mode can run (frozen binary, or script + llmlingua).
func Available(ux UXChrome) error {
	if _, err := OptimizerBin(ux); err == nil {
		return nil
	}
	script, err := ScriptPath(ux)
	if err != nil {
		return err
	}
	cmd := exec.Command(pythonBin(), "-c", "import llmlingua")
	if out, err := cmd.CombinedOutput(); err != nil {
		return ux.errFmt(ux.LLMMissingFmt, err, strings.TrimSpace(string(out)))
	}
	_ = script
	return nil
}

func requirementsPath(ux UXChrome) string {
	script, err := ScriptPath(ux)
	if err != nil {
		return ""
	}
	dir := filepath.Dir(script)
	candidates := []string{
		filepath.Join(dir, "requirements-deep.txt"),
		filepath.Join(dir, "requirements.txt"),
		filepath.Join(dir, "optimizer", "requirements-deep.txt"),
		filepath.Join(dir, "optimizer", "requirements.txt"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// Bootstrap installs Deep Mode deps from requirements-deep.txt beside optimizer.py (local only).
func Bootstrap(ux UXChrome) error {
	if _, err := ScriptPath(ux); err != nil {
		return err
	}
	req := requirementsPath(ux)
	if req == "" {
		return ux.err(ux.RequirementsMissing)
	}
	args := []string{"-m", "pip", "install", "-r", req, "--user", "--quiet"}
	if ux.BootstrapReqs != "" {
		fmt.Println(ux.BootstrapReqs)
	} else if ux.BootstrapPip != "" {
		fmt.Println(ux.BootstrapPip)
	}
	if runtime.GOOS == "windows" && pythonBin() == "py" {
		args = append([]string{"-3"}, args...)
	}
	cmd := exec.Command(pythonBin(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ux.PipFailed != "" {
			return fmt.Errorf("%s: %w", ux.PipFailed, err)
		}
		return err
	}
	return Available(ux)
}

// Compress runs Microsoft Deep Mode via frozen binary (preferred) or local Python bridge.
// timeoutSec is the wall-clock kill from TRIM_DEEP_TIMEOUT_SEC (no invent 600s).
func Compress(req Request, ux UXChrome, timeoutSec int) (*Response, error) {
	if req.TargetToken <= 0 {
		return nil, ux.err(ux.TargetRequired)
	}
	if strings.TrimSpace(req.Engine) == "" {
		return nil, ux.err(ux.EngineRequired)
	}
	if strings.TrimSpace(req.ModelName) == "" || strings.TrimSpace(req.DeviceMap) == "" {
		return nil, ux.errFmt(ux.OptimizeFmt, "model_name and device_map required (sync billing deep_*_model; set TRIM_DEEP_DEVICE_MAP)")
	}
	if strings.EqualFold(strings.TrimSpace(req.Engine), string(EngineLong)) &&
		strings.TrimSpace(req.Question) == "" {
		return nil, ux.err(ux.QuestionRequired)
	}

	if bin, err := OptimizerBin(ux); err == nil {
		return runOptimizerCmd(exec.Command(bin), req, ux, timeoutSec)
	}

	if err := Available(ux); err != nil {
		if ux.LLMMissingFmt != "" && strings.Contains(err.Error(), "llmlingua") {
			if ux.AutoInstall != "" {
				fmt.Println(ux.AutoInstall)
			}
			if bootErr := Bootstrap(ux); bootErr != nil {
				return nil, bootErr
			}
		} else {
			return nil, err
		}
	}
	script, err := ScriptPath(ux)
	if err != nil {
		return nil, err
	}
	return runOptimizerCmd(exec.Command(pythonBin(), script), req, ux, timeoutSec)
}

func runOptimizerCmd(cmd *exec.Cmd, req Request, ux UXChrome, timeoutSec int) (*Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	cmd.Stdin = bytes.NewReader(body)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	if timeoutSec < 1 {
		timeoutSec = 1
	}
	select {
	case err := <-done:
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return nil, ux.errFmt(ux.OptimizeFmt, msg)
		}
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, ux.err(ux.Timeout)
	}

	var res Response
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		return nil, ux.errFmt(ux.ParseFmt, err, strings.TrimSpace(stderr.String()))
	}
	// Source of truth: token counts. LLMLingua often returns saving_rate 0 / 0.0 / missing.
	res.SavingRate = FormatSavingPercent(res.OriginTokens, res.CompressedTokens)
	if strings.TrimSpace(res.Engine) == "" {
		res.Engine = strings.TrimSpace(req.Engine)
	}
	return &res, nil
}

// FormatSavingPercent is (origin - compressed) / origin as a one-decimal percent label.
// Negative when compression grew the payload (honest; do not clamp to fake 0%).
func FormatSavingPercent(origin, compressed int) string {
	if origin <= 0 {
		return "0.0%"
	}
	rate := (float64(origin) - float64(compressed)) / float64(origin) * 100.0
	return fmt.Sprintf("%.1f%%", rate)
}

// FastHeuristic is structural cleanup used when tier=fast (no ML).
func FastHeuristic(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	prevBlank := false
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if strings.TrimSpace(trimmed) == "" {
			if prevBlank {
				continue
			}
			prevBlank = true
			out = append(out, "")
			continue
		}
		prevBlank = false
		t := strings.TrimSpace(trimmed)
		if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "*") && !strings.HasPrefix(t, "*/") {
			if !strings.Contains(t, "go:build") && !strings.Contains(t, "eslint") && !strings.Contains(t, "prettier") {
				continue
			}
		}
		out = append(out, strings.ReplaceAll(trimmed, "\t", " "))
	}
	return strings.Join(out, "\n")
}
