package cli

import "strings"

// applyHumanChromeFallbacks fills empty human-UX chrome fields with the same
// defaults as server MessageForCode (site_messages). Used when the CLI is newer
// than a deployed API catalog so status/login/start stay usable worldwide.
func applyHumanChromeFallbacks(ch *cliChrome) {
	if ch == nil {
		return
	}
	set := func(dst *string, def string) {
		if strings.TrimSpace(*dst) == "" {
			*dst = def
		}
	}
	set(&ch.StatusLoggedIn, "Logged in")
	set(&ch.StatusLabelPlan, "Plan")
	set(&ch.StatusLabelCredits, "Credits")
	set(&ch.StatusLabelRemaining, "Remaining")
	set(&ch.StatusLabelDashboard, "Dashboard")
	set(&ch.StatusLabelUpgrade, "Upgrade")
	set(&ch.StatusRemainingFmt, "%d / %d")
	set(&ch.StatusChecking, "Checking account…")
	set(&ch.StatusHTTPErrorFmt, "Could not reach Trim Cloud (%s). Check your network and try again.")
	set(&ch.StartBaseURLLabel, "IDE Base URL")
	set(&ch.LoginURLLabel, "Sign-in URL")
	set(&ch.StatusNotLoggedIn, "Not logged in. Run: trim login")
}
