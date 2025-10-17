package llm

import (
	"context"
	"fmt"
)

type Provider interface {
	Summarize(ctx context.Context, request SummarizeRequest) (SummarizeResponse, error)
}

type SummarizeRequest struct {
	Type        SummaryType
	Context     string
	Constraints Constraints
	CacheKey    string
}

type SummarizeResponse struct {
	Summary string
	Cached  bool
	Tokens  int
}

type SummaryType string

const (
	SummaryTypeArchitecture SummaryType = "architecture"
	SummaryTypeModule       SummaryType = "module"
	SummaryTypeFile         SummaryType = "file"
	SummaryTypeFunction     SummaryType = "function"
	SummaryTypeQuickstart   SummaryType = "quickstart"
)

type Constraints struct {
	MaxWords   int
	MaxBullets int
	Style      string
}

type AnthropicConfig struct {
	APIKey   string
	CacheDir string
	Force    bool
	MaxQPS   float64
}

type NoOpProvider struct{}

func NewNoOpProvider() Provider {
	return &NoOpProvider{}
}

func (p *NoOpProvider) Summarize(ctx context.Context, request SummarizeRequest) (SummarizeResponse, error) {
	placeholder := fmt.Sprintf("[%s summary placeholder - dry run mode]", request.Type)
	return SummarizeResponse{
		Summary: placeholder,
		Cached:  false,
		Tokens:  0,
	}, nil
}
