package router

import (
	"time"

	"github.com/alucardeht/may-la-mcp/internal/types"
)

type QuerySource string

const (
	SourceIndex QuerySource = "index"
	SourceLSP   QuerySource = "lsp"
	SourceRegex QuerySource = "regex"
)

type Symbol = types.Symbol

type Reference = types.Reference

type QueryResult[T any] struct {
	Items    []T           `json:"items"`
	Count    int           `json:"count"`
	Source   QuerySource   `json:"source"`
	Latency  time.Duration `json:"latency_ms"`
	Cached   bool          `json:"cached"`
	Fallback bool          `json:"fallback"`
}

type QueryOptions struct {
	MaxResults    int           `json:"max_results"`
	Timeout       time.Duration `json:"timeout"`
	SkipIndex     bool          `json:"skip_index"`
	SkipLSP       bool          `json:"skip_lsp"`
	UpdateIndex   bool          `json:"update_index"`
	AllowFallback bool          `json:"allow_fallback"`
}

func DefaultQueryOptions() QueryOptions {
	return QueryOptions{
		MaxResults:    500,
		Timeout:       5 * time.Second,
		SkipIndex:     false,
		SkipLSP:       false,
		UpdateIndex:   true,
		AllowFallback: true,
	}
}
