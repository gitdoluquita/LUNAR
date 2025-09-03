package llm

import (
    "context"
    "fmt"
    "regexp"
    "strings"
    "time"
)

// Grouping encapsulates prompt construction and response parsing for log template grouping.
type Grouping struct {
    Provider   Provider
    Model      string
    Dataset    string
    PromptMode string // e.g., "VarExam"
}

type Exemplar struct {
    Query  string
    Answer string
}

func (g *Grouping) systemPrompt() string {
    return "You are a log parsing assistant for the cloud reliability team, skilled in identifying dynamic values of variables in logs."
}

func (g *Grouping) instruction() string {
    var b strings.Builder
    b.WriteString("# Basic Requirements:\n")
    b.WriteString("- I will provide multiple log messages, each delimited by backticks.\n")
    b.WriteString("- Identify and replace dynamic variables with {*} and output static log templates.\n")
    b.WriteString("- Preserve existing `<*>` placeholders.\n")
    b.WriteString("- For each log line, output one line starting with `LogTemplate[idx]:` and the template wrapped by backticks.\n")
    return b.String()
}

func (g *Grouping) buildMessages(logs []string, exemplars []Exemplar) []ChatMessage {
    msgs := []ChatMessage{
        {Role: "system", Content: g.systemPrompt()},
        {Role: "user", Content: g.instruction()},
        {Role: "assistant", Content: "OK, I'm ready to help."},
    }

    if len(exemplars) > 0 {
        var q []string
        var a []string
        for i, ex := range exemplars {
            q = append(q, fmt.Sprintf("Log[%d]: `%s`", i+1, ex.Query))
            a = append(a, fmt.Sprintf("LogTemplate[%d]: `%s`", i+1, ex.Answer))
        }
        msgs = append(msgs, ChatMessage{Role: "user", Content: strings.Join(q, "\n")})
        msgs = append(msgs, ChatMessage{Role: "assistant", Content: strings.Join(a, "\n")})
    }

    var q []string
    for i, l := range logs {
        q = append(q, fmt.Sprintf("Log[%d]: `%s`", i+1, l))
    }
    msgs = append(msgs, ChatMessage{Role: "user", Content: strings.Join(q, "\n")})
    return msgs
}

// ParsingLogTemplates queries the provider and returns an aggregated template and query duration.
func (g *Grouping) ParsingLogTemplates(ctx context.Context, logs []string, exemplars []Exemplar, reparse bool) (string, time.Duration, error) {
    temperature := float32(0.0)
    if reparse {
        temperature = 0.7
    }
    messages := g.buildMessages(logs, exemplars)
    resp, dur, err := g.Provider.Chat(ctx, g.Model, messages, temperature)
    if err != nil {
        return "", dur, err
    }
    templates := extractTemplatesFromResponse(resp)
    if len(templates) == 0 {
        // Fallback: naive post-process of the first log
        return NaivePostProcess(logs[0]), dur, nil
    }
    // Aggregate by majority (most frequent template)
    best := aggregateByMajority(templates)
    return best, dur, nil
}

// naivePostProcess replaces common dynamic artifacts (numbers, hex, ip, paths) with <*>.
func NaivePostProcess(s string) string {
    // IP addresses
    s = regexp.MustCompile(`(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`).ReplaceAllString(s, "<*>")
    // URLs
    s = regexp.MustCompile(`https?://[^\s]+`).ReplaceAllString(s, "<*>")
    // Paths
    s = regexp.MustCompile(`(/[^\s]+)+`).ReplaceAllString(s, "<*>")
    // Hex
    s = regexp.MustCompile(`0x[0-9a-fA-F]+`).ReplaceAllString(s, "<*>")
    // Numbers
    s = regexp.MustCompile(`\b\d+\b`).ReplaceAllString(s, "<*>")
    return s
}

func extractTemplatesFromResponse(resp string) []string {
    var out []string
    // Prefer lines like: LogTemplate[1]: `...`
    re := regexp.MustCompile("(?m)^\\s*LogTemplate\\[\\d+\\]:\\s*`([^`]*)`")
    for _, m := range re.FindAllStringSubmatch(resp, -1) {
        out = append(out, strings.TrimSpace(m[1]))
    }
    if len(out) > 0 {
        return out
    }
    // Otherwise collect any backticked lines
    reTick := regexp.MustCompile("`([^`]*)`")
    for _, m := range reTick.FindAllStringSubmatch(resp, -1) {
        out = append(out, strings.TrimSpace(m[1]))
    }
    return out
}

func aggregateByMajority(templates []string) string {
    counts := map[string]int{}
    best := ""
    bestC := 0
    for _, t := range templates {
        counts[t]++
        if counts[t] > bestC {
            best = t
            bestC = counts[t]
        }
    }
    if best == "" && len(templates) > 0 {
        return templates[0]
    }
    return best
}

