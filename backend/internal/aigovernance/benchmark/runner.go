package benchmark

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

// BenchmarkPrompt is a single test input in the prompt dataset.
type BenchmarkPrompt struct {
	SystemPrompt string `json:"system_prompt"`
	UserMessage  string `json:"user_message"`
	ExpectedRef  string `json:"expected_reference,omitempty"`
}

// IterationResult captures metrics from one inference call.
type IterationResult struct {
	PromptIndex      int     `json:"prompt_index"`
	LatencyMS        float64 `json:"latency_ms"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	ResponseContent  string  `json:"response_content"`
	TTFT_MS          float64 `json:"ttft_ms"`
	Error            string  `json:"error,omitempty"`
	Retries          int     `json:"retries,omitempty"`
}

// RetryConfig controls retry behaviour for transient failures.
type RetryConfig struct {
	MaxRetries    int           // Maximum number of retry attempts (0 = no retries).
	InitialDelay  time.Duration // Delay before the first retry.
	MaxDelay      time.Duration // Upper-bound on back-off delay.
	BackoffFactor float64       // Multiplier applied after each attempt (e.g. 2.0 for exponential).
	RetryableHTTP []int         // HTTP status codes that should trigger a retry (e.g. 429, 502, 503).
}

// DefaultRetryConfig returns a sensible starting point for retry behaviour.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		RetryableHTTP: []int{429, 500, 502, 503, 504},
	}
}

// RunConfig configures a benchmark execution.
type RunConfig struct {
	BaseURL        string
	ModelName      string
	APIKey         string // Optional Bearer token / API key.
	Prompts        []BenchmarkPrompt
	WarmupCount    int
	IterationCount int
	Concurrency    int
	Timeout        time.Duration
	Stream         bool        // When true, use SSE streaming to measure real TTFT.
	Retry          RetryConfig // Retry policy for transient failures.
}

// AggregatedResults holds statistical summaries computed from raw iterations.
type AggregatedResults struct {
	P50LatencyMS      float64
	P95LatencyMS      float64
	P99LatencyMS      float64
	AvgLatencyMS      float64
	MinLatencyMS      float64
	MaxLatencyMS      float64
	P50TTFT_MS        float64
	P95TTFT_MS        float64
	AvgTTFT_MS        float64
	TokensPerSecond   float64
	RequestsPerSecond float64
	TotalTokens       int64
	TotalRequests     int
	FailedRequests    int
	RetriedRequests   int
	DurationSeconds   int
	Raw               []IterationResult
}

// Runner executes benchmark iterations against an OpenAI-compatible inference server.
type Runner struct {
	client *http.Client
	logger zerolog.Logger
}

func NewRunner(logger zerolog.Logger) *Runner {
	return &Runner{
		client: &http.Client{Timeout: 120 * time.Second},
		logger: logger.With().Str("component", "benchmark_runner").Logger(),
	}
}

// Execute runs warmup + measured iterations and returns aggregated results.
func (r *Runner) Execute(ctx context.Context, cfg RunConfig) (*AggregatedResults, error) {
	if len(cfg.Prompts) == 0 {
		return nil, fmt.Errorf("benchmark requires at least one prompt")
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}

	// Warmup phase (results discarded).
	r.logger.Info().Int("warmup_count", cfg.WarmupCount).Msg("starting warmup phase")
	for i := 0; i < cfg.WarmupCount; i++ {
		prompt := cfg.Prompts[i%len(cfg.Prompts)]
		_, _ = r.callWithRetry(ctx, cfg, prompt)
	}

	// Measured phase.
	r.logger.Info().
		Int("iteration_count", cfg.IterationCount).
		Int("concurrency", cfg.Concurrency).
		Bool("stream", cfg.Stream).
		Msg("starting measured phase")

	start := time.Now()
	results := make([]IterationResult, cfg.IterationCount)
	var failedCount int64
	var retriedCount int64

	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup

	for i := 0; i < cfg.IterationCount; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			prompt := cfg.Prompts[idx%len(cfg.Prompts)]
			result, err := r.callWithRetry(ctx, cfg, prompt)
			if err != nil {
				result = &IterationResult{PromptIndex: idx % len(cfg.Prompts), Error: err.Error()}
				atomic.AddInt64(&failedCount, 1)
			}
			result.PromptIndex = idx % len(cfg.Prompts)
			if result.Retries > 0 {
				atomic.AddInt64(&retriedCount, 1)
			}
			results[idx] = *result
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	return r.aggregate(results, int(failedCount), int(retriedCount), elapsed), nil
}

// ---------------------------------------------------------------------------
// Retry wrapper
// ---------------------------------------------------------------------------

func (r *Runner) callWithRetry(ctx context.Context, cfg RunConfig, prompt BenchmarkPrompt) (*IterationResult, error) {
	var lastErr error
	delay := cfg.Retry.InitialDelay

	for attempt := 0; attempt <= cfg.Retry.MaxRetries; attempt++ {
		if attempt > 0 {
			r.logger.Debug().
				Int("attempt", attempt+1).
				Dur("backoff", delay).
				Msg("retrying inference call")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			// Exponential back-off with cap.
			delay = time.Duration(float64(delay) * cfg.Retry.BackoffFactor)
			if delay > cfg.Retry.MaxDelay {
				delay = cfg.Retry.MaxDelay
			}
		}

		var result *IterationResult
		var err error
		if cfg.Stream {
			result, err = r.callInferenceStream(ctx, cfg, prompt)
		} else {
			result, err = r.callInference(ctx, cfg, prompt)
		}

		if err == nil {
			result.Retries = attempt
			return result, nil
		}

		// Decide whether this error is retryable.
		if !r.isRetryable(err, cfg.Retry.RetryableHTTP) {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("exhausted %d retries: %w", cfg.Retry.MaxRetries, lastErr)
}

// httpStatusError is returned when the server sends a retryable HTTP status.
type httpStatusError struct {
	Code int
	Body string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("inference server returned %d: %s", e.Code, e.Body)
}

func (r *Runner) isRetryable(err error, retryableCodes []int) bool {
	var httpErr *httpStatusError
	if ok := asHTTPStatusError(err, &httpErr); ok {
		for _, code := range retryableCodes {
			if httpErr.Code == code {
				return true
			}
		}
		return false
	}
	// Treat context-deadline-exceeded (per-request timeout) as retryable,
	// but NOT the parent context cancellation.
	if ctx_err, ok := err.(interface{ Unwrap() error }); ok {
		_ = ctx_err
	}
	// Network-level errors (connection refused, DNS, etc.) are retryable.
	return true
}

// asHTTPStatusError mimics errors.As for the local httpStatusError type.
func asHTTPStatusError(err error, target **httpStatusError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*httpStatusError); ok {
		*target = e
		return true
	}
	if w, ok := err.(interface{ Unwrap() error }); ok {
		return asHTTPStatusError(w.Unwrap(), target)
	}
	return false
}

// ---------------------------------------------------------------------------
// Non-streaming inference (original behaviour, now with auth header)
// ---------------------------------------------------------------------------

func (r *Runner) callInference(ctx context.Context, cfg RunConfig, prompt BenchmarkPrompt) (*IterationResult, error) {
	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	payload, err := r.buildPayload(cfg.ModelName, prompt, false)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, cfg.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	r.setHeaders(req, cfg.APIKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inference call: %w", err)
	}
	defer resp.Body.Close()
	latency := time.Since(start)

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, &httpStatusError{Code: resp.StatusCode, Body: string(respBody)}
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	content := ""
	if len(parsed.Choices) > 0 {
		content = parsed.Choices[0].Message.Content
	}

	return &IterationResult{
		LatencyMS:        float64(latency.Milliseconds()),
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
		ResponseContent:  content,
	}, nil
}

// ---------------------------------------------------------------------------
// Streaming SSE inference – measures real TTFT
// ---------------------------------------------------------------------------

// streamChunk represents one parsed SSE data frame from the server.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage,omitempty"`
}

func (r *Runner) callInferenceStream(ctx context.Context, cfg RunConfig, prompt BenchmarkPrompt) (*IterationResult, error) {
	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	payload, err := r.buildPayload(cfg.ModelName, prompt, true)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, cfg.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	r.setHeaders(req, cfg.APIKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inference call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, &httpStatusError{Code: resp.StatusCode, Body: string(body)}
	}

	// Parse SSE stream, recording TTFT on the first content-bearing chunk.
	var (
		ttft         time.Duration
		ttftRecorded bool
		content      strings.Builder
		promptTok    int
		completeTok  int
	)

	scanner := bufio.NewScanner(resp.Body)
	// Raise the scanner buffer for large chunks (256 KB).
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// SSE lines are prefixed with "data: ".
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// The stream terminator.
		if strings.TrimSpace(data) == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			r.logger.Warn().Err(err).Str("raw", data).Msg("failed to parse SSE chunk")
			continue
		}

		// Extract delta content.
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				if !ttftRecorded {
					ttft = time.Since(start)
					ttftRecorded = true
				}
				content.WriteString(delta)
			}
		}

		// Some providers send usage in the final chunk (OpenAI with stream_options).
		if chunk.Usage != nil {
			promptTok = chunk.Usage.PromptTokens
			completeTok = chunk.Usage.CompletionTokens
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read SSE stream: %w", err)
	}

	latency := time.Since(start)

	var ttftMS float64
	if ttftRecorded {
		ttftMS = float64(ttft.Microseconds()) / 1000.0 // sub-millisecond precision
	}

	return &IterationResult{
		LatencyMS:        float64(latency.Milliseconds()),
		TTFT_MS:          ttftMS,
		PromptTokens:     promptTok,
		CompletionTokens: completeTok,
		ResponseContent:  content.String(),
	}, nil
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

func (r *Runner) buildPayload(model string, prompt BenchmarkPrompt, stream bool) ([]byte, error) {
	messages := []map[string]string{}
	if prompt.SystemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": prompt.SystemPrompt})
	}
	messages = append(messages, map[string]string{"role": "user", "content": prompt.UserMessage})

	body := map[string]any{
		"model":       model,
		"messages":    messages,
		"max_tokens":  1024,
		"temperature": 0.1,
	}
	if stream {
		body["stream"] = true
		// Request usage stats in the final streamed chunk (OpenAI-compatible).
		body["stream_options"] = map[string]any{"include_usage": true}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	return payload, nil
}

func (r *Runner) setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

// ---------------------------------------------------------------------------
// Aggregation
// ---------------------------------------------------------------------------

func (r *Runner) aggregate(results []IterationResult, failedCount, retriedCount int, elapsed time.Duration) *AggregatedResults {
	latencies := make([]float64, 0, len(results))
	ttfts := make([]float64, 0, len(results))
	var totalTokens int64
	successful := 0

	for _, res := range results {
		if res.Error != "" {
			continue
		}
		latencies = append(latencies, res.LatencyMS)
		if res.TTFT_MS > 0 {
			ttfts = append(ttfts, res.TTFT_MS)
		}
		totalTokens += int64(res.PromptTokens + res.CompletionTokens)
		successful++
	}

	if len(latencies) == 0 {
		return &AggregatedResults{
			TotalRequests:   len(results),
			FailedRequests:  failedCount,
			RetriedRequests: retriedCount,
			DurationSeconds: int(elapsed.Seconds()),
			Raw:             results,
		}
	}

	sort.Float64s(latencies)
	sort.Float64s(ttfts)

	elapsedSec := elapsed.Seconds()
	if elapsedSec == 0 {
		elapsedSec = 1
	}

	agg := &AggregatedResults{
		P50LatencyMS:      percentile(latencies, 0.50),
		P95LatencyMS:      percentile(latencies, 0.95),
		P99LatencyMS:      percentile(latencies, 0.99),
		AvgLatencyMS:      avg(latencies),
		MinLatencyMS:      latencies[0],
		MaxLatencyMS:      latencies[len(latencies)-1],
		TokensPerSecond:   float64(totalTokens) / elapsedSec,
		RequestsPerSecond: float64(successful) / elapsedSec,
		TotalTokens:       totalTokens,
		TotalRequests:     len(results),
		FailedRequests:    failedCount,
		RetriedRequests:   retriedCount,
		DurationSeconds:   int(elapsed.Seconds()),
		Raw:               results,
	}

	// TTFT aggregates (only populated when streaming was used).
	if len(ttfts) > 0 {
		agg.P50TTFT_MS = percentile(ttfts, 0.50)
		agg.P95TTFT_MS = percentile(ttfts, 0.95)
		agg.AvgTTFT_MS = avg(ttfts)
	}

	return agg
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := p * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper || upper >= len(sorted) {
		return sorted[lower]
	}
	frac := idx - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
