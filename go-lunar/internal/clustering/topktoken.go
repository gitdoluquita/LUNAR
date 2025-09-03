package clustering

import (
    "fmt"
    "sort"
    "strings"

    "github.com/agnivade/levenshtein"
    "github.com/samber/lo"
)

type LogRow struct {
    LineID   int    `csv:"LineId"`
    Content  string `csv:"Content"`
    EventID  string `csv:"EventId"`
    Template string `csv:"EventTemplate"`
}

type Bucket struct {
    ID      int
    Rows    []LogRow
    Length  int
    Parent  int
}

type TopKTokenClustering struct {
    MinClusterSize int
    ClusterTopK    int
    SampleSize     int
    SampleAuto     bool
    LcuLambda      float64
    LcuSampleSize  int

    Buckets        map[int]*Bucket
    ParentToChild  map[int][]int
    ChildToParent  map[int]int
    NumProcessed   int
    NumTotal       int
}

func NewTopKTokenClustering() *TopKTokenClustering {
    return &TopKTokenClustering{
        MinClusterSize: 100,
        ClusterTopK:    3,
        SampleSize:     3,
        SampleAuto:     true,
        LcuLambda:      0.6,
        LcuSampleSize:  3,
        Buckets:        map[int]*Bucket{},
        ParentToChild:  map[int][]int{},
        ChildToParent:  map[int]int{},
    }
}

func tokenLength(s string) int { return len(strings.Fields(s)) }

func (c *TopKTokenClustering) Load(rows []LogRow) {
    c.NumProcessed = 0
    c.NumTotal = len(rows)
    // cluster by length first
    byLen := map[int][]LogRow{}
    order := []int{}
    for _, r := range rows {
        l := tokenLength(r.Content)
        if _, ok := byLen[l]; !ok {
            order = append(order, l)
        }
        byLen[l] = append(byLen[l], r)
    }
    sort.Ints(order)
    buckets := map[int]*Bucket{}
    id := 0
    for _, l := range order {
        b := &Bucket{ID: id, Rows: byLen[l], Length: l}
        buckets[id] = b
        id++
    }
    c.Buckets = buckets
    fmt.Printf("Clustering by log length: %d\n", len(c.Buckets))
}

func (c *TopKTokenClustering) Clustering() map[int]*Bucket {
    // For the Go MVP, we keep only the length-based buckets to keep logic concise and idiomatic.
    // Further top-k token refinement can be added if needed.
    return c.Buckets
}

// SampleForLLM returns a bucket id and sampled logs with simple diversity using edit distance.
func (c *TopKTokenClustering) SampleForLLM() (int, []string) {
    // Pick the largest unprocessed bucket
    var best *Bucket
    for _, b := range c.Buckets {
        remaining := 0
        for _, r := range b.Rows {
            if r.Template == "" {
                remaining++
            }
        }
        if remaining == 0 {
            continue
        }
        if best == nil || remaining > bestRemaining(best) {
            best = b
        }
    }
    if best == nil {
        return -1, nil
    }
    // anchor + select others by min pairwise similarity (Levenshtein distance)
    candidates := lo.Filter(best.Rows, func(r LogRow, _ int) bool { return r.Template == "" })
    if len(candidates) == 0 {
        return best.ID, nil
    }
    anchor := candidates[0].Content
    sampled := []string{anchor}
    rest := lo.Map(candidates[1:], func(r LogRow, _ int) string { return r.Content })
    // select up to SampleSize-1 by increasing distance to anchor
    sort.Slice(rest, func(i, j int) bool { return levenshtein.ComputeDistance(anchor, rest[i]) < levenshtein.ComputeDistance(anchor, rest[j]) })
    for i := 0; i < len(rest) && len(sampled) < c.SampleSize; i++ {
        sampled = append(sampled, rest[i])
    }
    return best.ID, sampled
}

func bestRemaining(b *Bucket) int {
    n := 0
    for _, r := range b.Rows {
        if r.Template == "" {
            n++
        }
    }
    return n
}

// UpdateLogsWithTemplate applies a template to all rows that match in a bucket and returns the number matched.
func (c *TopKTokenClustering) UpdateLogsWithTemplate(bucketID int, template string, match func(string, string) bool) (int, []int) {
    b := c.Buckets[bucketID]
    if b == nil {
        return 0, nil
    }
    matched := 0
    var idxs []int
    for i := range b.Rows {
        if b.Rows[i].Template != "" {
            continue
        }
        if match(b.Rows[i].Content, template) {
            b.Rows[i].Template = template
            matched++
            idxs = append(idxs, i)
        }
    }
    c.NumProcessed += matched
    return matched, idxs
}

