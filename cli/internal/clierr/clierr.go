package clierr

import "errors"

// ErrChromeUnavailable is returned when auth-providers CLI chrome is missing
// and no ChromeUnavailable copy was loaded from the API. Machine-readable only;
// do not invent English marketing copy here.
var ErrChromeUnavailable = errors.New("CLI_CHROME_UNAVAILABLE")
