package utils

import (
    "regexp"
    "strings"
)

var reVar = regexp.MustCompile(`\<\*\>`) // literal <*> placeholder

// VerifyTemplateForLogWithFirstToken checks if a template can match the log with basic constraints.
// Heuristic: template tokens must appear in order; <*> matches any substring.
func VerifyTemplateForLogWithFirstToken(log, template string) bool {
    if strings.TrimSpace(template) == "" {
        return false
    }
    // Quick path when template equals log
    if log == template {
        return true
    }
    // Split by <*> and ensure parts appear in order
    parts := reVar.Split(template, -1)
    idx := 0
    for _, p := range parts {
        if p == "" {
            continue
        }
        j := strings.Index(log[idx:], p)
        if j < 0 {
            return false
        }
        idx += j + len(p)
    }
    return true
}

// ValidateTemplate minimal check: not empty and contains at least one static token or one placeholder.
func ValidateTemplate(t string) bool {
    t = strings.TrimSpace(t)
    if t == "" {
        return false
    }
    // Avoid templates that are only placeholders
    onlyPlaceholders := strings.ReplaceAll(t, "<*>", "")
    return strings.TrimSpace(onlyPlaceholders) != ""
}

// PreprocessLogForQuery optionally applies regex replacements before querying an LLM.
// For now, supports simple built-in replacements for IP/URL/path.
func PreprocessLogForQuery(s string) string {
    // Replace common patterns that are obviously variables to reduce prompt length.
    s = regexp.MustCompile(`(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`).ReplaceAllString(s, "<*>")
    s = regexp.MustCompile(`https?://[^\s]+`).ReplaceAllString(s, "<*>")
    s = regexp.MustCompile(`(/[^\s]+)+`).ReplaceAllString(s, "<*>")
    return s
}

