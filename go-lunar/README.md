LUNAR (Go)

A Go-idiomatic port of the core LUNAR log template parsing workflow. It provides:

- LLM abstraction with an OpenAI-compatible provider
- TopKToken-like clustering (length-based MVP) and sampling
- Template database with simple merge heuristic
- Parser orchestrator and CLI

Install

```bash
cd /workspace/go-lunar
go build ./...
```

CLI Usage

```bash
go run ./cmd/lunar \
  -in /path/to/YourDataset_full.log_structured.csv \
  -out ./saved_results/LUNAR-go \
  -model gpt-3.5-turbo-0125 \
  -api_key $OPENAI_API_KEY \
  -base_url $OPENAI_BASE_URL
```

Input format: CSV with headers `LineId,Content,EventId,EventTemplate`.
Outputs mirror the Python version: `*_full.log_structured.csv` and `*_full.log_templates.csv` under the output directory.

Extending providers

Implement `internal/llm.Provider` for other LLMs (Anthropic, Azure, local) and wire it in `cmd/lunar/main.go`.

Notes

- This port focuses on the LUNAR core parsing flow rather than full parity (prompt variants, evaluation, etc.).
- The clustering is a simplified MVP that can be enhanced with token-based grouping.

