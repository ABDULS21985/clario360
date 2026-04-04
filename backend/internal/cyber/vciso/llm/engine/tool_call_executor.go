package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/auth"
	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
	llmtools "github.com/clario360/platform/internal/cyber/vciso/llm/tools"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrToolCallNil          = errors.New("tool: call is nil")
	ErrToolUnknown          = errors.New("tool: unknown function")
	ErrToolValidation       = errors.New("tool: argument validation failed")
	ErrToolPermission       = errors.New("tool: permission denied")
	ErrToolConfirmation     = errors.New("tool: destructive action requires confirmation")
	ErrToolExecution        = errors.New("tool: execution failed")
	ErrToolTimeout          = errors.New("tool: execution timed out")
	ErrToolCircuitOpen      = errors.New("tool: circuit breaker open")
	ErrToolConcurrencyLimit = errors.New("tool: concurrency limit reached")
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const (
	DefaultMaxToolResultSize = 10_000
	DefaultToolTimeout       = 10 * time.Second
	DefaultMaxConcurrency    = 8
	DefaultSummaryMaxLen     = 240

	// Circuit breaker defaults
	DefaultCBFailThreshold = 5
	DefaultCBResetAfter    = 30 * time.Second
)

// ExecutorOption configures the ToolCallExecutor via functional options.
type ExecutorOption func(*ToolCallExecutor)

func WithMaxResultSize(n int) ExecutorOption {
	return func(e *ToolCallExecutor) {
		if n > 0 {
			e.maxResultSize = n
		}
	}
}

func WithDefaultTimeout(d time.Duration) ExecutorOption {
	return func(e *ToolCallExecutor) {
		if d > 0 {
			e.defaultTimeout = d
		}
	}
}

func WithMaxConcurrency(n int) ExecutorOption {
	return func(e *ToolCallExecutor) {
		if n > 0 {
			e.maxConcurrency = n
		}
	}
}

func WithExecutorLogger(l zerolog.Logger) ExecutorOption {
	return func(e *ToolCallExecutor) { e.logger = l }
}

func WithCircuitBreaker(failThreshold int, resetAfter time.Duration) ExecutorOption {
	return func(e *ToolCallExecutor) {
		if failThreshold > 0 {
			e.cbFailThreshold = failThreshold
		}
		if resetAfter > 0 {
			e.cbResetAfter = resetAfter
		}
	}
}

// WithToolTimeout registers a per-tool timeout override.  Tools not
// registered here fall back to the executor-level default.
func WithToolTimeout(toolName string, d time.Duration) ExecutorOption {
	return func(e *ToolCallExecutor) {
		if d > 0 {
			e.toolTimeouts[toolName] = d
		}
	}
}

// ---------------------------------------------------------------------------
// ToolCallExecutor
// ---------------------------------------------------------------------------

// ToolCallExecutor resolves, validates, authorises, and executes LLM tool
// calls with:
//   - per-tool configurable timeouts
//   - bounded concurrency for parallel execution
//   - per-tool circuit breakers to shed load on repeated failures
//   - structured result truncation with metadata
//   - sentinel error types for programmatic handling
type ToolCallExecutor struct {
	registry       *llmtools.Registry
	maxResultSize  int
	defaultTimeout time.Duration
	maxConcurrency int
	toolTimeouts   map[string]time.Duration
	logger         zerolog.Logger

	// Circuit breaker state
	cbFailThreshold int
	cbResetAfter    time.Duration
	cbMu            sync.RWMutex
	cbState         map[string]*circuitState
}

type circuitState struct {
	failures    int
	lastFailure time.Time
}

func NewToolCallExecutor(registry *llmtools.Registry, opts ...ExecutorOption) *ToolCallExecutor {
	e := &ToolCallExecutor{
		registry:        registry,
		maxResultSize:   DefaultMaxToolResultSize,
		defaultTimeout:  DefaultToolTimeout,
		maxConcurrency:  DefaultMaxConcurrency,
		toolTimeouts:    make(map[string]time.Duration),
		cbFailThreshold: DefaultCBFailThreshold,
		cbResetAfter:    DefaultCBResetAfter,
		cbState:         make(map[string]*circuitState),
		logger:          zerolog.Nop(),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ---------------------------------------------------------------------------
// Single execution
// ---------------------------------------------------------------------------

// Execute runs a single tool call through the full pipeline:
//
//	resolve → circuit-breaker check → validate → authorise → confirm → execute → truncate
func (e *ToolCallExecutor) Execute(
	ctx context.Context, toolCall *llmmodel.LLMToolCall,
	tenantID, userID uuid.UUID,
) (*llmmodel.ToolCallResult, error) {
	// --- Guard: nil call ------------------------------------------------
	if toolCall == nil {
		return nil, ErrToolCallNil
	}

	name := toolCall.FunctionName
	start := time.Now()
	log := e.logger.With().Str("tool", name).Logger()

	// --- Resolve --------------------------------------------------------
	tool := e.registry.Get(name)
	if tool == nil {
		return nil, fmt.Errorf("%w: %s", ErrToolUnknown, name)
	}

	// --- Circuit breaker ------------------------------------------------
	if e.circuitOpen(name) {
		log.Warn().Msg("circuit breaker open, rejecting call")
		return e.failureResult(name, ErrToolCircuitOpen, start), ErrToolCircuitOpen
	}

	// --- Validate -------------------------------------------------------
	if err := validateArguments(tool.Schema(), toolCall.Arguments); err != nil {
		return e.failureResult(name, err, start), fmt.Errorf("%w: %v", ErrToolValidation, err)
	}

	// --- Authorise ------------------------------------------------------
	if missing := missingPermissions(permissionsFromContext(ctx), tool.RequiredPermissions()); len(missing) > 0 {
		err := fmt.Errorf("%w: requires %s", ErrToolPermission, strings.Join(missing, ", "))
		return e.failureResult(name, err, start), err
	}

	// --- Destructive confirmation ---------------------------------------
	if tool.IsDestructive() && !truthy(toolCall.Arguments["confirm"]) {
		return e.failureResult(name, ErrToolConfirmation, start), ErrToolConfirmation
	}

	// --- Execute with timeout -------------------------------------------
	timeout := e.timeoutFor(name)
	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, execErr := tool.Execute(toolCtx, tenantID, userID, toolCall.Arguments)
	latency := time.Since(start)
	latencyMs := int(latency.Milliseconds())

	if execErr != nil {
		e.recordFailure(name)

		// Distinguish timeout from general execution error.
		if errors.Is(toolCtx.Err(), context.DeadlineExceeded) {
			log.Warn().Dur("timeout", timeout).Msg("tool execution timed out")
			return e.failureResult(name, ErrToolTimeout, start),
				fmt.Errorf("%w: %s after %v", ErrToolTimeout, name, timeout)
		}

		log.Warn().Err(execErr).Int("latency_ms", latencyMs).Msg("tool execution failed")
		return &llmmodel.ToolCallResult{
			ToolName:  name,
			Success:   false,
			Error:     execErr.Error(),
			LatencyMs: latencyMs,
			Summary:   execErr.Error(),
		}, fmt.Errorf("%w: %v", ErrToolExecution, execErr)
	}

	// Success — reset circuit breaker for this tool.
	e.recordSuccess(name)

	out := e.buildSuccessResult(name, result, latencyMs)

	log.Debug().
		Int("latency_ms", latencyMs).
		Str("data_type", out.DataType).
		Bool("truncated", out.Truncated).
		Msg("tool execution succeeded")

	return out, nil
}

// ---------------------------------------------------------------------------
// Parallel execution
// ---------------------------------------------------------------------------

// ExecuteAll runs multiple tool calls concurrently with bounded parallelism.
// Results are returned in the same order as the input calls.  Individual
// failures are captured in the result slice — the returned error is non-nil
// only for systemic issues (e.g. context cancellation).
func (e *ToolCallExecutor) ExecuteAll(
	ctx context.Context, calls []llmmodel.LLMToolCall,
	tenantID, userID uuid.UUID,
) ([]*llmmodel.ToolCallResult, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	results := make([]*llmmodel.ToolCallResult, len(calls))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(e.maxConcurrency)

	for idx := range calls {
		idx := idx
		group.Go(func() error {
			result, _ := e.Execute(groupCtx, &calls[idx], tenantID, userID)
			results[idx] = result

			// Individual tool failures are captured in the result; they
			// should NOT cancel sibling goroutines.  We return nil here
			// so errgroup keeps running the rest.
			return nil
		})
	}

	// The only error from Wait() would be context cancellation.
	if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return results, err
	}

	return results, nil
}

// ---------------------------------------------------------------------------
// Result building
// ---------------------------------------------------------------------------

func (e *ToolCallExecutor) buildSuccessResult(
	name string, result *chattools.ToolResult, latencyMs int,
) *llmmodel.ToolCallResult {
	summary := summarizeToolResult(result)

	out := &llmmodel.ToolCallResult{
		ToolName:  name,
		Success:   true,
		Data:      result.Data,
		Summary:   summary,
		LatencyMs: latencyMs,
		DataType:  result.DataType,
		Actions:   result.Actions,
		Entities:  result.Entities,
		Text:      result.Text,
	}

	// Measure payload size for truncation.
	payload, _ := json.Marshal(result.Data)
	payloadSize := len(payload)
	out.TotalItems = payloadSize

	if payloadSize <= e.maxResultSize {
		out.ReturnedItems = payloadSize
		return out
	}

	// Truncate: keep the summary informative and flag the result.
	out.Truncated = true
	out.ReturnedItems = e.maxResultSize
	out.Summary = fmt.Sprintf("%s [truncated: %d → %d bytes]",
		summary, payloadSize, e.maxResultSize)

	return out
}

func (e *ToolCallExecutor) failureResult(name string, err error, start time.Time) *llmmodel.ToolCallResult {
	return &llmmodel.ToolCallResult{
		ToolName:  name,
		Success:   false,
		Error:     err.Error(),
		LatencyMs: int(time.Since(start).Milliseconds()),
		Summary:   err.Error(),
	}
}

// ---------------------------------------------------------------------------
// Circuit breaker (per-tool, lightweight)
// ---------------------------------------------------------------------------
//
// The circuit trips after cbFailThreshold consecutive failures and stays open
// for cbResetAfter.  A single success resets the counter (half-open → closed).
// This is intentionally simple — no external dependencies, no background
// goroutines.  Upgrade to sony/gobreaker if you need more sophistication.

func (e *ToolCallExecutor) circuitOpen(tool string) bool {
	e.cbMu.RLock()
	defer e.cbMu.RUnlock()

	cs, ok := e.cbState[tool]
	if !ok {
		return false
	}
	if cs.failures < e.cbFailThreshold {
		return false
	}
	// Allow a probe if enough time has passed (half-open).
	return time.Since(cs.lastFailure) < e.cbResetAfter
}

func (e *ToolCallExecutor) recordFailure(tool string) {
	e.cbMu.Lock()
	defer e.cbMu.Unlock()

	cs, ok := e.cbState[tool]
	if !ok {
		cs = &circuitState{}
		e.cbState[tool] = cs
	}
	cs.failures++
	cs.lastFailure = time.Now()
}

func (e *ToolCallExecutor) recordSuccess(tool string) {
	e.cbMu.Lock()
	defer e.cbMu.Unlock()

	if cs, ok := e.cbState[tool]; ok {
		cs.failures = 0
	}
}

// ResetCircuit manually resets the circuit breaker for a tool.
// Useful in tests or after a deployment/config change.
func (e *ToolCallExecutor) ResetCircuit(tool string) {
	e.cbMu.Lock()
	defer e.cbMu.Unlock()
	delete(e.cbState, tool)
}

// ---------------------------------------------------------------------------
// Timeout resolution
// ---------------------------------------------------------------------------

func (e *ToolCallExecutor) timeoutFor(tool string) time.Duration {
	if d, ok := e.toolTimeouts[tool]; ok {
		return d
	}
	return e.defaultTimeout
}

// ---------------------------------------------------------------------------
// Argument validation
// ---------------------------------------------------------------------------
//
// This validates against the JSON Schema subset that tool schemas use:
//   - required fields present and non-empty
//   - enum constraints
//   - basic type checks (integer, boolean, array)

func validateArguments(schema map[string]any, args map[string]any) error {
	if err := validateRequired(schema, args); err != nil {
		return err
	}
	return validateProperties(schema, args)
}

func validateRequired(schema map[string]any, args map[string]any) error {
	required := extractRequiredFields(schema)
	for _, key := range required {
		value, ok := args[key]
		if !ok || value == nil || fmt.Sprint(value) == "" {
			return fmt.Errorf("missing required argument: %s", key)
		}
	}
	return nil
}

// extractRequiredFields handles both []string and []any (the latter comes
// from JSON unmarshalling of the schema).
func extractRequiredFields(schema map[string]any) []string {
	switch raw := schema["required"].(type) {
	case []string:
		return raw
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func validateProperties(schema map[string]any, args map[string]any) error {
	properties, _ := schema["properties"].(map[string]any)
	for key, raw := range properties {
		value, ok := args[key]
		if !ok || value == nil {
			continue
		}

		property, _ := raw.(map[string]any)

		if err := validateEnum(key, property, value); err != nil {
			return err
		}
		if err := validateType(key, property, value); err != nil {
			return err
		}
	}
	return nil
}

func validateEnum(key string, property map[string]any, value any) error {
	// Normalise the enum slice (handles both []string and []any from JSON).
	var allowed []string
	switch raw := property["enum"].(type) {
	case []string:
		allowed = raw
	case []any:
		allowed = make([]string, 0, len(raw))
		for _, item := range raw {
			allowed = append(allowed, fmt.Sprint(item))
		}
	}
	if len(allowed) > 0 && !containsFold(allowed, fmt.Sprint(value)) {
		return fmt.Errorf("invalid value for %s: got %q, allowed %v", key, value, allowed)
	}
	return nil
}

func validateType(key string, property map[string]any, value any) error {
	switch property["type"] {
	case "integer":
		if !isNumeric(value) {
			return fmt.Errorf("%s must be an integer, got %T", key, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok && !isBooleanString(value) {
			return fmt.Errorf("%s must be a boolean, got %T", key, value)
		}
	case "array":
		switch value.(type) {
		case []any, []string:
			// ok
		default:
			return fmt.Errorf("%s must be an array, got %T", key, value)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Permission resolution
// ---------------------------------------------------------------------------

func permissionsFromContext(ctx context.Context) []string {
	var values []string

	if claims := auth.ClaimsFromContext(ctx); claims != nil {
		values = append(values, claims.Permissions...)
		values = appendRolePermissions(values, claims.Roles)
	}
	if user := auth.UserFromContext(ctx); user != nil {
		values = appendRolePermissions(values, user.Roles)
	}

	return dedup(values)
}

func appendRolePermissions(perms, roles []string) []string {
	for _, role := range roles {
		normalised := strings.ReplaceAll(role, "-", "_")
		perms = append(perms, auth.RolePermissions[normalised]...)
	}
	return perms
}

func missingPermissions(have, required []string) []string {
	out := make([]string, 0, len(required))
	for _, need := range required {
		if need == "" {
			continue
		}
		if !granted(have, need) {
			out = append(out, need)
		}
	}
	return out
}

func granted(have []string, required string) bool {
	for _, item := range have {
		if item == auth.PermAdminAll || item == required {
			return true
		}
		// Wildcard: "module:*" matches "module:read", "module:write", etc.
		if strings.HasSuffix(item, ":*") &&
			strings.HasPrefix(required, strings.TrimSuffix(item, "*")) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Pure helpers
// ---------------------------------------------------------------------------

func summarizeToolResult(result *chattools.ToolResult) string {
	if result == nil {
		return "No result returned."
	}
	if text := strings.TrimSpace(result.Text); text != "" {
		return truncateSummaryText(strings.ReplaceAll(text, "\n", " "), DefaultSummaryMaxLen)
	}
	payload, _ := json.Marshal(result.Data)
	return truncateSummaryText(string(payload), DefaultSummaryMaxLen)
}

func dedup(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func containsFold(values []string, target string) bool {
	for _, v := range values {
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	default:
		return false
	}
}

func isNumeric(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

func isBooleanString(value any) bool {
	text, ok := value.(string)
	return ok && (strings.EqualFold(text, "true") || strings.EqualFold(text, "false"))
}

func truncateSummaryText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
