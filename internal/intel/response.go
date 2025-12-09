package intel

import (
	"fmt"
	"time"
)

type ResponseConfig struct {
	Mode              ResponseMode
	MaxLength         int
	Summarize         bool
	Truncate          bool
	TruncateMode      TruncateMode
	Format            bool
	Rank              bool
	RankCriteria      RankCriteria
	IncludeContext    bool
	ContextRadius     int
	ApplyFormatting   bool
	LineLimitHead     int
	LineLimitTail     int
	TimestampResponse bool
}

var DefaultResponseConfig = ResponseConfig{
	Mode:              ResponseModeCompact,
	MaxLength:         2000,
	Summarize:         true,
	Truncate:          true,
	TruncateMode:      TruncateModeSmart,
	Format:            true,
	Rank:              false,
	RankCriteria:      DefaultRankCriteria,
	IncludeContext:    false,
	ContextRadius:     5,
	ApplyFormatting:   true,
	LineLimitHead:     5,
	LineLimitTail:     3,
	TimestampResponse: false,
}

type IntelligentResponse struct {
	Data       string                 `json:"data"`
	Summary    string                 `json:"summary,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	Context    *Context               `json:"context,omitempty"`
	Indicators []string               `json:"indicators"`
	Timestamp  time.Time              `json:"timestamp,omitempty"`
	ProcessedAt time.Time             `json:"processed_at,omitempty"`
}

type ResponseProcessor struct {
	config ResponseConfig
}

func NewResponseProcessor(config ResponseConfig) *ResponseProcessor {
	return &ResponseProcessor{
		config: config,
	}
}

func ProcessResponse(data string, config ResponseConfig) IntelligentResponse {
	processor := NewResponseProcessor(config)
	return processor.Process(data)
}

func (rp *ResponseProcessor) Process(data string) IntelligentResponse {
	response := IntelligentResponse{
		Data:        data,
		Metadata:    make(map[string]interface{}),
		Indicators:  []string{},
		ProcessedAt: time.Now(),
	}

	response.Metadata["original_length"] = len(data)
	response.Metadata["mode"] = rp.config.Mode

	if rp.config.Summarize {
		response.Summary = Summarize(data, rp.config.MaxLength/2)
		response.Metadata["summarized"] = true
	}

	if rp.config.Truncate {
		response.Data = Truncate(data, rp.config.MaxLength, rp.config.TruncateMode)
		response.Metadata["truncated"] = true
		response.Metadata["final_length"] = len(response.Data)
	}

	if rp.config.ApplyFormatting {
		formatter := NewFormatterBuilder().
			WithMode(rp.config.Mode).
			WithMaxLength(rp.config.MaxLength).
			WithLineLimits(rp.config.LineLimitHead, rp.config.LineLimitTail).
			Build()

		formatted := formatter.Format(response.Data, response.Metadata)
		response.Data = formatted.Content
		response.Indicators = formatted.Indicators

		for k, v := range formatted.Metadata {
			response.Metadata[k] = v
		}
	}

	if rp.config.IncludeContext {
		response.Context = &Context{
			Content: data,
		}
	}

	if rp.config.TimestampResponse {
		response.Timestamp = time.Now()
	}

	return response
}

type CompactResponseBuilder struct {
	data    string
	config  ResponseConfig
	items   int
	omitted int
}

func NewCompactResponseBuilder(data string) *CompactResponseBuilder {
	return &CompactResponseBuilder{
		data: data,
		config: ResponseConfig{
			Mode:            ResponseModeCompact,
			MaxLength:       1500,
			Summarize:       true,
			Truncate:        true,
			TruncateMode:    TruncateModeSmart,
			ApplyFormatting: true,
			LineLimitHead:   5,
			LineLimitTail:   2,
		},
	}
}

func (crb *CompactResponseBuilder) WithItems(count int) *CompactResponseBuilder {
	crb.items = count
	return crb
}

func (crb *CompactResponseBuilder) WithOmitted(count int) *CompactResponseBuilder {
	crb.omitted = count
	return crb
}

func (crb *CompactResponseBuilder) Build() IntelligentResponse {
	response := ProcessResponse(crb.data, crb.config)

	if crb.items > 0 {
		response.Metadata["total_items"] = crb.items
	}

	if crb.omitted > 0 {
		response.Indicators = append(response.Indicators, fmt.Sprintf("+%d more items", crb.omitted))
	}

	return response
}

type DetailedResponseBuilder struct {
	data   string
	config ResponseConfig
}

func NewDetailedResponseBuilder(data string) *DetailedResponseBuilder {
	return &DetailedResponseBuilder{
		data: data,
		config: ResponseConfig{
			Mode:            ResponseModeDetailed,
			MaxLength:       5000,
			Summarize:       false,
			Truncate:        false,
			ApplyFormatting: true,
			LineLimitHead:   10,
			LineLimitTail:   5,
		},
	}
}

func (drb *DetailedResponseBuilder) WithContext(include bool) *DetailedResponseBuilder {
	drb.config.IncludeContext = include
	return drb
}

func (drb *DetailedResponseBuilder) Build() IntelligentResponse {
	return ProcessResponse(drb.data, drb.config)
}

func (ir IntelligentResponse) String() string {
	output := ir.Data

	if ir.Summary != "" {
		output = ir.Summary + "\n\n" + output
	}

	if len(ir.Indicators) > 0 {
		output = output + "\n\n"
		for i, indicator := range ir.Indicators {
			if i > 0 {
				output += " | "
			}
			output += indicator
		}
	}

	return output
}

func MergeResponses(responses ...IntelligentResponse) IntelligentResponse {
	if len(responses) == 0 {
		return IntelligentResponse{
			Metadata: make(map[string]interface{}),
		}
	}

	merged := IntelligentResponse{
		Data:       "",
		Summary:    "",
		Metadata:   make(map[string]interface{}),
		Indicators: []string{},
	}

	totalLength := 0
	totalItems := 0

	for _, resp := range responses {
		merged.Data += resp.Data + "\n"
		totalLength += len(resp.Data)

		if items, ok := resp.Metadata["total_items"].(int); ok {
			totalItems += items
		}

		for _, indicator := range resp.Indicators {
			merged.Indicators = append(merged.Indicators, indicator)
		}
	}

	merged.Metadata["merged_responses"] = len(responses)
	merged.Metadata["total_length"] = totalLength
	merged.Metadata["total_items"] = totalItems

	return merged
}

func BatchProcessResponses(dataList []string, config ResponseConfig) []IntelligentResponse {
	responses := make([]IntelligentResponse, len(dataList))

	for i, data := range dataList {
		responses[i] = ProcessResponse(data, config)
	}

	return responses
}

type SmartResponseConfig struct {
	ContentType   string
	ItemCount     int
	TotalSize     int
	AutoOptimize  bool
	PreferCompact bool
}

func DetermineOptimalConfig(smartConfig SmartResponseConfig) ResponseConfig {
	config := DefaultResponseConfig

	if smartConfig.PreferCompact {
		config.Mode = ResponseModeCompact
		config.MaxLength = 1500
		config.Truncate = true
		config.TruncateMode = TruncateModeSmart
	}

	if smartConfig.AutoOptimize {
		if smartConfig.ItemCount > 100 {
			config.Summarize = true
			config.Truncate = true
			config.Mode = ResponseModeCompact
		}

		if smartConfig.TotalSize > 10000 {
			config.MaxLength = 3000
			config.TruncateMode = TruncateModeSmart
		}

		if smartConfig.ContentType == "code" {
			config.LineLimitHead = 10
			config.LineLimitTail = 5
		}
	}

	return config
}
