package templatedb

import (
    "sort"
    "strings"
)

// Database stores discovered templates and can suggest merges.
type Database struct {
    Templates []string
}

func New() *Database { return &Database{Templates: []string{}} }

func (d *Database) All() []string { return append([]string(nil), d.Templates...) }

// AddTemplate inserts a new template and tries a simple merge with the most similar one by token overlap.
// Returns possibly-updated template (merged) and whether the merge happened.
func (d *Database) AddTemplate(t string) (string, bool) {
    if len(d.Templates) == 0 {
        d.Templates = append(d.Templates, t)
        return t, false
    }
    bestIdx := -1
    bestSim := -1
    toks := strings.Fields(t)
    for i, ot := range d.Templates {
        sim := jaccard(toks, strings.Fields(ot))
        if sim > bestSim {
            bestSim = sim
            bestIdx = i
        }
    }
    merged, ok := tryMerge(d.Templates[bestIdx], t)
    if ok {
        d.Templates[bestIdx] = merged
        return merged, true
    }
    d.Templates = append(d.Templates, t)
    return t, false
}

func jaccard(a, b []string) int {
    sa := map[string]struct{}{}
    for _, x := range a { sa[x] = struct{}{} }
    sb := map[string]struct{}{}
    for _, x := range b { sb[x] = struct{}{} }
    inter := 0
    union := map[string]struct{}{}
    for x := range sa { union[x] = struct{}{} }
    for x := range sb { union[x] = struct{}{} }
    for x := range sa { if _, ok := sb[x]; ok { inter++ } }
    return inter*1000 + (len(union)-inter) // weighted tie-breaker
}

// tryMerge merges two templates of identical token length by replacing differing positions with <*>.
func tryMerge(a, b string) (string, bool) {
    ta := strings.Fields(a)
    tb := strings.Fields(b)
    if len(ta) != len(tb) || len(ta) == 0 {
        return a, false
    }
    diff := 0
    out := make([]string, len(ta))
    for i := range ta {
        if ta[i] == tb[i] {
            out[i] = ta[i]
        } else {
            out[i] = "<*>"
            diff++
        }
    }
    if diff == 0 { return a, false }
    // avoid degenerating into all placeholders
    if allPlaceholders(out) { return a, false }
    return strings.Join(out, " "), true
}

func allPlaceholders(ts []string) bool {
    for _, t := range ts { if t != "<*>" { return false } }
    return true
}

// Sort ensures deterministic order for output.
func (d *Database) Sort() { sort.Strings(d.Templates) }

