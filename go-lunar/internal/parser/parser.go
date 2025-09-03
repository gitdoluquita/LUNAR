package parser

import (
    "context"
    "encoding/csv"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strconv"

    "github.com/samber/lo"

    "github.com/example/go-lunar/internal/clustering"
    "github.com/example/go-lunar/internal/llm"
    "github.com/example/go-lunar/internal/templatedb"
    "github.com/example/go-lunar/internal/utils"
)

type Config struct {
    InputCSV   string
    OutputDir  string
    AddRegex   bool
    Model      string
}

type Parser struct {
    LLM       *llm.Grouping
    Clus      *clustering.TopKTokenClustering
    TempDB    *templatedb.Database
}

func New(p llm.Provider, model string) *Parser {
    return &Parser{
        LLM: &llm.Grouping{Provider: p, Model: model, PromptMode: "VarExam"},
        Clus: clustering.NewTopKTokenClustering(),
        TempDB: templatedb.New(),
    }
}

// LoadCSV expects a CSV with columns LineId,Content,EventId,EventTemplate.
func LoadCSV(path string) ([]clustering.LogRow, error) {
    f, err := os.Open(path)
    if err != nil { return nil, err }
    defer f.Close()
    r := csv.NewReader(f)
    r.FieldsPerRecord = -1
    rows, err := r.ReadAll()
    if err != nil { return nil, err }
    if len(rows) == 0 { return nil, fmt.Errorf("empty csv") }
    // Find column indices
    hdr := rows[0]
    col := func(name string) int {
        for i, h := range hdr { if h == name { return i } }
        return -1
    }
    iLine := col("LineId")
    iContent := col("Content")
    iEventID := col("EventId")
    iTemplate := col("EventTemplate")
    if iLine < 0 || iContent < 0 {
        return nil, fmt.Errorf("missing required headers")
    }
    var out []clustering.LogRow
    for _, row := range rows[1:] {
        lineID := 0
        if iLine < len(row) && row[iLine] != "" { _id, _ := strconv.Atoi(row[iLine]); lineID = _id }
        content := ""
        if iContent < len(row) { content = row[iContent] }
        eventID := ""
        if iEventID >= 0 && iEventID < len(row) { eventID = row[iEventID] }
        template := ""
        if iTemplate >= 0 && iTemplate < len(row) { template = row[iTemplate] }
        out = append(out, clustering.LogRow{LineID: lineID, Content: content, EventID: eventID, Template: template})
    }
    return out, nil
}

func SaveResults(rows []clustering.LogRow, outDir, logName string) error {
    if err := os.MkdirAll(outDir, 0o755); err != nil { return err }
    // Write structured csv
    structured := filepath.Join(outDir, fmt.Sprintf("%s_full.log_structured.csv", logName))
    if err := writeCSV(structured, rows); err != nil { return err }

    // Write templates csv (EventId, EventTemplate)
    // Assign EventId based on template order.
    tmplSet := []string{}
    type pair struct{ EventID, Template string }
    var pairs []pair
    for i := range rows {
        t := rows[i].Template
        if t == "" { continue }
        idx := lo.IndexOf(tmplSet, t)
        if idx < 0 { tmplSet = append(tmplSet, t); idx = len(tmplSet)-1 }
        pairs = append(pairs, pair{EventID: fmt.Sprintf("E%d", idx+1), Template: t})
    }
    // unique by (EventID, Template)
    uniq := map[string]pair{}
    for _, p := range pairs { uniq[p.EventID+"|"+p.Template] = p }
    var list []pair
    for _, p := range uniq { list = append(list, p) }
    sort.Slice(list, func(i,j int) bool { return list[i].EventID < list[j].EventID })

    templates := filepath.Join(outDir, fmt.Sprintf("%s_full.log_templates.csv", logName))
    f, err := os.Create(templates)
    if err != nil { return err }
    w := csv.NewWriter(f)
    _ = w.Write([]string{"EventId", "EventTemplate"})
    for _, p := range list { _ = w.Write([]string{p.EventID, p.Template}) }
    w.Flush(); _ = f.Close()
    return nil
}

func writeCSV(path string, rows []clustering.LogRow) error {
    f, err := os.Create(path)
    if err != nil { return err }
    w := csv.NewWriter(f)
    _ = w.Write([]string{"LineId","Content","EventId","EventTemplate"})
    for _, r := range rows {
        _ = w.Write([]string{strconv.Itoa(r.LineID), r.Content, r.EventID, r.Template})
    }
    w.Flush(); _ = f.Close()
    return nil
}

func (p *Parser) Parse(ctx context.Context, rows []clustering.LogRow, addRegex bool) ([]clustering.LogRow, error) {
    // optional preprocess
    if addRegex {
        for i := range rows { rows[i].Content = utils.PreprocessLogForQuery(rows[i].Content) }
    }
    p.Clus.Load(rows)
    _ = p.Clus.Clustering()

    // Iterate until all templates assigned or no progress
    for p.Clus.NumProcessed < p.Clus.NumTotal {
        bucketID, logs := p.Clus.SampleForLLM()
        if bucketID < 0 || len(logs) == 0 { break }
        tmpl, _, err := p.LLM.ParsingLogTemplates(ctx, logs, nil, false)
        if err != nil { return nil, err }
        if !utils.ValidateTemplate(tmpl) { tmpl = llm.NaivePostProcess(logs[0]) }
        matched, _ := p.Clus.UpdateLogsWithTemplate(bucketID, tmpl, utils.VerifyTemplateForLogWithFirstToken)
        if matched == 0 {
            // Retry: use compromise response (naive)
            tmpl = llm.NaivePostProcess(logs[0])
            _, _ = p.Clus.UpdateLogsWithTemplate(bucketID, tmpl, utils.VerifyTemplateForLogWithFirstToken)
        }
        // Update template DB (best-effort)
        if tmpl != "" { _, _ = p.TempDB.AddTemplate(tmpl) }
        // Exit condition: avoid infinite loops if nothing progresses
        if matched == 0 { break }
    }

    // Flatten current buckets back into rows
    var out []clustering.LogRow
    for _, b := range p.Clus.Buckets {
        out = append(out, b.Rows...)
    }
    return out, nil
}

