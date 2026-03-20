package diagnostic

import (
	"fmt"
	"net/url"
	"strings"
)

// URLProblem describes a detected issue with an environment URL
// and suggests a correction.
type URLProblem struct {
	// Message describes the problem in human-readable form.
	Message string
	// SuggestedURL is the corrected URL, if one can be derived.
	SuggestedURL string
}

// CheckEnvironmentURL inspects an environment URL for common mistakes.
// It returns a list of problems found (empty if the URL looks correct).
func CheckEnvironmentURL(environmentURL string) []URLProblem {
	if environmentURL == "" {
		return nil
	}

	var problems []URLProblem

	lower := strings.ToLower(environmentURL)

	// --- SaaS production: <tenant>.live.dynatrace.com instead of <tenant>.apps.dynatrace.com ---
	if strings.Contains(lower, ".live.dynatrace.com") {
		suggested := fixDomain(environmentURL, ".live.dynatrace.com", ".apps.dynatrace.com")
		problems = append(problems, URLProblem{
			Message:      "Environment URL uses 'live.dynatrace.com' which is the classic API domain. dtctl requires the platform API at 'apps.dynatrace.com'",
			SuggestedURL: suggested,
		})
	}

	// --- SaaS production: <tenant>.dynatrace.com (bare domain without .apps.) ---
	// Match "xyz123.dynatrace.com" but NOT "*.apps.dynatrace.com" or "*.live.dynatrace.com"
	// or other known subdomains
	if !strings.Contains(lower, ".apps.dynatrace.com") &&
		!strings.Contains(lower, ".live.dynatrace.com") &&
		matchesBareProductionDomain(lower) {
		suggested := fixDomain(environmentURL, ".dynatrace.com", ".apps.dynatrace.com")
		problems = append(problems, URLProblem{
			Message:      "Environment URL uses the bare 'dynatrace.com' domain. dtctl requires the platform API at 'apps.dynatrace.com'",
			SuggestedURL: suggested,
		})
	}

	// --- DEV: <tenant>.dev.dynatracelabs.com instead of <tenant>.dev.apps.dynatracelabs.com ---
	if strings.Contains(lower, ".dev.dynatracelabs.com") &&
		!strings.Contains(lower, ".dev.apps.dynatracelabs.com") {
		suggested := fixDomain(environmentURL, ".dev.dynatracelabs.com", ".dev.apps.dynatracelabs.com")
		problems = append(problems, URLProblem{
			Message:      "Environment URL uses 'dev.dynatracelabs.com' without '.apps.' in the domain. dtctl requires the platform API at 'dev.apps.dynatracelabs.com'",
			SuggestedURL: suggested,
		})
	}

	// --- SPRINT/HARD: <tenant>.sprint.dynatracelabs.com instead of <tenant>.sprint.apps.dynatracelabs.com ---
	if strings.Contains(lower, ".sprint.dynatracelabs.com") &&
		!strings.Contains(lower, ".sprint.apps.dynatracelabs.com") {
		suggested := fixDomain(environmentURL, ".sprint.dynatracelabs.com", ".sprint.apps.dynatracelabs.com")
		problems = append(problems, URLProblem{
			Message:      "Environment URL uses 'sprint.dynatracelabs.com' without '.apps.' in the domain. dtctl requires the platform API at 'sprint.apps.dynatracelabs.com'",
			SuggestedURL: suggested,
		})
	}

	// --- Managed/ActiveGate: <host>/e/<envid> pattern (classic managed URL) ---
	if strings.Contains(lower, "/e/") && !strings.Contains(lower, "apps.") {
		problems = append(problems, URLProblem{
			Message: "Environment URL looks like a Dynatrace Managed or ActiveGate URL (/e/<envid> path). dtctl requires the Dynatrace Platform (SaaS) 'apps' URL",
		})
	}

	return problems
}

// URLSuggestions returns troubleshooting suggestions based on URL problems.
// This is intended to be appended to existing error suggestions.
func URLSuggestions(environmentURL string) []string {
	problems := CheckEnvironmentURL(environmentURL)
	if len(problems) == 0 {
		return nil
	}

	var suggestions []string
	for _, p := range problems {
		suggestions = append(suggestions, fmt.Sprintf("Possible wrong environment URL: %s", p.Message))
		if p.SuggestedURL != "" {
			suggestions = append(suggestions, fmt.Sprintf("Did you mean %s?", p.SuggestedURL))
		}
	}
	suggestions = append(suggestions, "Update with: dtctl ctx set <name> --environment <correct-url>")
	return suggestions
}

// matchesBareProductionDomain checks if a lowercased URL ends with
// "<something>.dynatrace.com" without any known subdomain prefix like
// "apps.", "live.", "sso.", etc.
func matchesBareProductionDomain(lower string) bool {
	// Parse the URL to extract the host
	u, err := url.Parse(lower)
	if err != nil {
		return false
	}

	host := u.Hostname()
	if host == "" {
		// Maybe it was provided without scheme
		host = lower
		// Strip any path
		if idx := strings.Index(host, "/"); idx >= 0 {
			host = host[:idx]
		}
	}

	// Must end with .dynatrace.com
	if !strings.HasSuffix(host, ".dynatrace.com") {
		return false
	}

	// Extract the part before .dynatrace.com
	prefix := strings.TrimSuffix(host, ".dynatrace.com")

	// It should be just a tenant ID (no dots — that would mean there's a subdomain)
	// e.g., "abc12345" rather than "abc12345.live" or "abc12345.apps"
	return !strings.Contains(prefix, ".")
}

// fixDomain replaces oldSuffix with newSuffix in the URL, case-insensitively.
func fixDomain(rawURL, oldSuffix, newSuffix string) string {
	lower := strings.ToLower(rawURL)
	idx := strings.Index(lower, strings.ToLower(oldSuffix))
	if idx < 0 {
		return rawURL
	}
	return rawURL[:idx] + newSuffix + rawURL[idx+len(oldSuffix):]
}
