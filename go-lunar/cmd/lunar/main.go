package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/example/go-lunar/internal/llm"
    "github.com/example/go-lunar/internal/parser"
)

func main() {
    in := flag.String("in", "", "Path to input CSV (LineId,Content,EventId,EventTemplate)")
    outDir := flag.String("out", "./saved_results/LUNAR-go", "Output directory")
    model := flag.String("model", "gpt-3.5-turbo-0125", "Model name")
    apiKey := flag.String("api_key", os.Getenv("OPENAI_API_KEY"), "API key")
    baseURL := flag.String("base_url", os.Getenv("OPENAI_BASE_URL"), "OpenAI compatible base URL")
    addRegex := flag.Bool("add_regex", false, "Enable simple regex preprocessing before LLM query")
    flag.Parse()

    if *in == "" {
        fmt.Println("-in is required")
        os.Exit(1)
    }
    if *apiKey == "" {
        fmt.Println("-api_key is empty; set OPENAI_API_KEY or pass -api_key")
        os.Exit(1)
    }

    prov := llm.NewOpenAIProvider(*apiKey, *baseURL)
    p := parser.New(prov, *model)

    rows, err := parser.LoadCSV(*in)
    if err != nil { panic(err) }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    outRows, err := p.Parse(ctx, rows, *addRegex)
    if err != nil { panic(err) }

    // Name outputs based on input file stem
    base := strings.TrimSuffix(filepath.Base(*in), filepath.Ext(*in))
    if err := parser.SaveResults(outRows, *outDir, base); err != nil { panic(err) }
    fmt.Println("Done. Saved to", *outDir)
}

