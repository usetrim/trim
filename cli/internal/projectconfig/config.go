package projectconfig

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Compression modes from the product bible (Mode 1-4).
const (
	ModeMild       = "mild"
	ModeBalanced   = "balanced"
	ModeAggressive = "aggressive"
	ModeCustom     = "custom"
)

// Config is project-local Trim rules loaded from .trimrc and .tokenignore.
type Config struct {
	IgnorePaths          []string
	NeverTrimPaths       []string
	ActiveFileProtection bool
	// ActiveFileProtectionSet is true only when .trimrc explicitly set the key (no invent default).
	ActiveFileProtectionSet bool
	MaxLogBytes             int
	LogCompactMinBytes      int
	LogCompactMaxLines      int
	// LogNoiseSubstrings: case-insensitive substrings dropped by CompactLogs (no invent).
	LogNoiseSubstrings []string
	// Mode: mild | balanced | aggressive | custom. Empty until .trimrc sets it (no invent).
	Mode string
	// CustomMinLines applies when Mode is custom (and aggressive uses 5).
	CustomMinLines int
	// CustomLogsOnly skips AST skeletonization when Mode is custom.
	CustomLogsOnly bool
	// CustomQueries optional Tree-sitter / rule snippets for custom mode tooling.
	CustomQueries []string
	// HistoryKeepTurns: last N non-system chat turns kept; older turns stubbed. 0 = off.
	HistoryKeepTurns int
}

// Load walks from startDir upward looking for .trimrc and .tokenignore.
// Missing files are fine. Mode stays empty until .trimrc sets it (fail closed).
func Load(startDir string) (Config, error) {
	// Fail-closed: ActiveFileProtection stays false until .trimrc or site_messages chrome sets it.
	cfg := Config{}
	if startDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return cfg, err
		}
		startDir = wd
	}

	dir := startDir
	for {
		trimrc := filepath.Join(dir, ".trimrc")
		if data, err := os.ReadFile(trimrc); err == nil {
			parseTrimrc(string(data), &cfg)
		}
		tokenignore := filepath.Join(dir, ".tokenignore")
		if f, err := os.Open(tokenignore); err == nil {
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				cfg.NeverTrimPaths = append(cfg.NeverTrimPaths, line)
			}
			_ = f.Close()
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cfg, nil
}

func parseTrimrc(raw string, cfg *Config) {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(parts[0]))
		val := strings.TrimSpace(parts[1])
		switch key {
		case "ignore_paths", "ignore":
			for _, p := range strings.Split(val, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					cfg.IgnorePaths = append(cfg.IgnorePaths, p)
				}
			}
		case "never_trim", "never_trim_paths":
			for _, p := range strings.Split(val, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					cfg.NeverTrimPaths = append(cfg.NeverTrimPaths, p)
				}
			}
		case "active_file_protection":
			cfg.ActiveFileProtection = val == "true" || val == "1" || val == "yes"
			cfg.ActiveFileProtectionSet = true
		case "max_log_bytes":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				cfg.MaxLogBytes = n
			}
		case "log_compact_min_bytes", "compact_min_bytes":
			if n, err := strconv.Atoi(val); err == nil && n >= 0 {
				cfg.LogCompactMinBytes = n
			}
		case "log_compact_max_lines", "compact_max_lines":
			if n, err := strconv.Atoi(val); err == nil && n >= 0 {
				cfg.LogCompactMaxLines = n
			}
		case "log_noise_substrings", "noise_substrings", "log_noise":
			for _, p := range strings.Split(val, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					cfg.LogNoiseSubstrings = append(cfg.LogNoiseSubstrings, p)
				}
			}
		case "mode", "compression_mode", "trim_mode":
			cfg.Mode = NormalizeMode(val)
		case "custom_min_lines", "min_lines":
			if n, err := strconv.Atoi(val); err == nil && n >= 0 {
				cfg.CustomMinLines = n
			}
		case "custom_logs_only", "logs_only":
			cfg.CustomLogsOnly = val == "true" || val == "1" || val == "yes"
		case "custom_query", "query":
			if val != "" {
				cfg.CustomQueries = append(cfg.CustomQueries, val)
			}
		case "history_keep_turns", "keep_turns", "history_turns":
			if n, err := strconv.Atoi(val); err == nil && n >= 0 {
				cfg.HistoryKeepTurns = n
			}
		}
	}
}

// NormalizeMode maps bible aliases (1-4, names) to canonical mode strings.
// Empty or unknown returns "" (no silent invent of balanced).
func NormalizeMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "mild", "1", "mode1", "mode_1":
		return ModeMild
	case "balanced", "2", "mode2", "mode_2":
		return ModeBalanced
	case "aggressive", "3", "mode3", "mode_3":
		return ModeAggressive
	case "custom", "4", "mode4", "mode_4":
		return ModeCustom
	default:
		return ""
	}
}

// MinLinesForMode returns the small-file exemption threshold for skeletonization.
// Returns 0 when mode is unset (caller / trimmer applies mode defaults only after mode is set).
func (c Config) MinLinesForMode() int {
	switch NormalizeMode(c.Mode) {
	case ModeAggressive:
		return 5
	case ModeCustom:
		if c.CustomMinLines > 0 {
			return c.CustomMinLines
		}
		return 15
	case ModeMild, ModeBalanced:
		return 15
	default:
		return 0
	}
}

// SkeletonizeEnabled is false for mild (logs only), custom+logs_only, or unset mode.
func (c Config) SkeletonizeEnabled() bool {
	switch NormalizeMode(c.Mode) {
	case ModeMild:
		return false
	case ModeCustom:
		return !c.CustomLogsOnly
	case ModeBalanced, ModeAggressive:
		return true
	default:
		return false
	}
}

// AlwaysCompactLogs is true for mild and aggressive (and custom when logs-focused).
func (c Config) AlwaysCompactLogs() bool {
	switch NormalizeMode(c.Mode) {
	case ModeMild, ModeAggressive:
		return true
	case ModeCustom:
		return c.CustomLogsOnly
	default:
		return false
	}
}

// MustNeverTrim returns true when path matches a never-trim or ignore pattern.
// Supports *, ?, and ** (any directory depth) globs.
func (c Config) MustNeverTrim(filePath string) bool {
	norm := strings.ReplaceAll(strings.ToLower(filePath), "\\", "/")
	base := filepath.Base(norm)
	for _, pat := range append(append([]string{}, c.NeverTrimPaths...), c.IgnorePaths...) {
		pat = strings.ToLower(strings.TrimSpace(pat))
		if pat == "" {
			continue
		}
		if matchGlob(pat, base) || matchGlob(pat, norm) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, name string) bool {
	pattern = strings.ReplaceAll(pattern, "\\", "/")
	name = strings.ReplaceAll(name, "\\", "/")
	if pattern == name {
		return true
	}
	if !strings.Contains(pattern, "**") {
		ok, _ := filepath.Match(pattern, name)
		return ok
	}
	re := globToRegexp(pattern)
	return re.MatchString(name)
}

func globToRegexp(pattern string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); {
		if i+1 < len(pattern) && pattern[i] == '*' && pattern[i+1] == '*' {
			if i+2 < len(pattern) && pattern[i+2] == '/' {
				b.WriteString("(.*/)?")
				i += 3
				continue
			}
			b.WriteString(".*")
			i += 2
			continue
		}
		switch c := pattern[i]; c {
		case '*':
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
		i++
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return regexp.MustCompile("^$")
	}
	return re
}
