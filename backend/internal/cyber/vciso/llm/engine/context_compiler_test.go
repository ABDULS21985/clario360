package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// fakeConversationRepo stubs the repository for testing.
type fakeConversationRepo struct {
	messages []chatmodel.Message
	err      error
}

func (r *fakeConversationRepo) ListMessages(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]chatmodel.Message, error) {
	return r.messages, r.err
}

// fakeSummariser captures calls and returns canned output.
type fakeSummariser struct {
	called bool
	input  []chatmodel.Message
	output string
	err    error
}

func (s *fakeSummariser) Summarise(_ context.Context, msgs []chatmodel.Message) (string, error) {
	s.called = true
	s.input = msgs
	return s.output, s.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeAlternating(n int) []chatmodel.Message {
	msgs := make([]chatmodel.Message, n)
	for i := range msgs {
		if i%2 == 0 {
			msgs[i] = chatmodel.Message{Role: chatmodel.MessageRoleUser, Content: fmt.Sprintf("user says %d", i)}
		} else {
			msgs[i] = chatmodel.Message{Role: chatmodel.MessageRoleAssistant, Content: fmt.Sprintf("assistant says %d", i)}
		}
	}
	return msgs
}

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }

func countMessages(compiled *CompiledContext, role string) int {
	n := 0
	for _, m := range compiled.Messages {
		if m.Role == role {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Tests: Guard clauses
// ---------------------------------------------------------------------------

func TestCompile_NilCompiler(t *testing.T) {
	var c *ContextCompiler
	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Error("expected empty messages for nil compiler")
	}
}

func TestCompile_NilConversationID(t *testing.T) {
	c := NewContextCompiler(&fakeConversationRepo{}, TokenBudget{}, nil)
	result, err := c.Compile(context.Background(), nil, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Error("expected empty messages for nil conversation ID")
	}
}

func TestCompile_ZeroUUID(t *testing.T) {
	c := NewContextCompiler(&fakeConversationRepo{}, TokenBudget{}, nil)
	nilID := uuid.Nil
	result, err := c.Compile(context.Background(), &nilID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Error("expected empty messages for zero UUID")
	}
}

// ---------------------------------------------------------------------------
// Tests: Empty history
// ---------------------------------------------------------------------------

func TestCompile_EmptyHistory(t *testing.T) {
	repo := &fakeConversationRepo{messages: []chatmodel.Message{}}
	c := NewContextCompiler(repo, TokenBudget{}, nil)

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Error("expected empty messages for empty history")
	}
	if result.WasTruncated {
		t.Error("should not be truncated with empty history")
	}
}

// ---------------------------------------------------------------------------
// Tests: Repository error
// ---------------------------------------------------------------------------

func TestCompile_RepoError(t *testing.T) {
	repo := &fakeConversationRepo{err: errors.New("db down")}
	c := NewContextCompiler(repo, TokenBudget{}, nil)

	_, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err == nil {
		t.Fatal("expected error from repository failure")
	}
}

// ---------------------------------------------------------------------------
// Tests: Messages within recency window (no summary needed)
// ---------------------------------------------------------------------------

func TestCompile_WithinRecencyWindow(t *testing.T) {
	msgs := makeAlternating(6)
	repo := &fakeConversationRepo{messages: msgs}
	c := NewContextCompiler(repo, TokenBudget{}, nil, WithRecencyWindow(10))

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) != 6 {
		t.Errorf("expected 6 messages, got %d", len(result.Messages))
	}
	if result.WasSummarised {
		t.Error("should not be summarised when within recency window")
	}
	if result.DroppedTurns != 0 {
		t.Errorf("expected 0 dropped turns, got %d", result.DroppedTurns)
	}
}

// ---------------------------------------------------------------------------
// Tests: Messages exceed recency window → summary injected
// ---------------------------------------------------------------------------

func TestCompile_ExceedsRecencyWindow_SummaryInjected(t *testing.T) {
	msgs := makeAlternating(15)
	repo := &fakeConversationRepo{messages: msgs}
	c := NewContextCompiler(repo, TokenBudget{}, nil, WithRecencyWindow(10))

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.WasSummarised {
		t.Error("expected summary when messages exceed recency window")
	}
	if result.DroppedTurns < 5 {
		t.Errorf("expected at least 5 dropped turns, got %d", result.DroppedTurns)
	}
	// First message should be the summary.
	if !strings.HasPrefix(result.Messages[0].Content, "Conversation summary:") {
		t.Errorf("first message should be summary, got: %s", result.Messages[0].Content)
	}
}

// ---------------------------------------------------------------------------
// Tests: Custom summariser
// ---------------------------------------------------------------------------

func TestCompile_CustomSummariser(t *testing.T) {
	msgs := makeAlternating(15)
	repo := &fakeConversationRepo{messages: msgs}
	summariser := &fakeSummariser{output: "TL;DR: they talked about security"}

	c := NewContextCompiler(repo, TokenBudget{}, nil,
		WithRecencyWindow(10),
		WithSummariser(summariser),
	)

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !summariser.called {
		t.Error("custom summariser was not called")
	}
	if len(summariser.input) != 5 {
		t.Errorf("summariser received %d messages, expected 5", len(summariser.input))
	}
	if !strings.Contains(result.Messages[0].Content, "TL;DR") {
		t.Error("expected custom summary content in first message")
	}
}

func TestCompile_SummariserError_Nonfatal(t *testing.T) {
	msgs := makeAlternating(15)
	repo := &fakeConversationRepo{messages: msgs}
	summariser := &fakeSummariser{err: errors.New("LLM unavailable")}

	c := NewContextCompiler(repo, TokenBudget{}, nil,
		WithRecencyWindow(10),
		WithSummariser(summariser),
	)

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("summary failure should be non-fatal, got: %v", err)
	}
	if result.WasSummarised {
		t.Error("should not be marked summarised when summariser failed")
	}
	// Recent messages should still be present.
	if len(result.Messages) == 0 {
		t.Error("recent messages should still be compiled despite summary failure")
	}
}

// ---------------------------------------------------------------------------
// Tests: Token budget enforcement
// ---------------------------------------------------------------------------

func TestCompile_TokenBudgetTruncatesRecent(t *testing.T) {
	// Create 8 messages, each roughly 10 tokens (~40 chars).
	msgs := make([]chatmodel.Message, 8)
	for i := range msgs {
		msgs[i] = chatmodel.Message{
			Role:    chatmodel.MessageRoleUser,
			Content: strings.Repeat("word ", 10), // ~50 chars ≈ 13 tokens
		}
	}
	repo := &fakeConversationRepo{messages: msgs}

	// Budget of 30 tokens — should fit ~2 messages.
	c := NewContextCompiler(repo, TokenBudget{ConversationHistoryBudget: 30}, nil,
		WithRecencyWindow(10),
	)

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) >= 8 {
		t.Errorf("expected budget to truncate messages, got %d", len(result.Messages))
	}
	if !result.WasTruncated {
		t.Error("expected WasTruncated to be true")
	}
	if result.TotalTokens > 30+15 { // some slack for the first message always included
		t.Errorf("total tokens %d significantly exceeds budget 30", result.TotalTokens)
	}
}

func TestCompile_UnlimitedBudget(t *testing.T) {
	msgs := makeAlternating(8)
	repo := &fakeConversationRepo{messages: msgs}

	// Budget 0 = unlimited.
	c := NewContextCompiler(repo, TokenBudget{ConversationHistoryBudget: 0}, nil,
		WithRecencyWindow(10),
	)

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Messages) != 8 {
		t.Errorf("expected all 8 messages with unlimited budget, got %d", len(result.Messages))
	}
}

// ---------------------------------------------------------------------------
// Tests: System messages filtered
// ---------------------------------------------------------------------------

func TestCompile_SystemMessagesSkipped(t *testing.T) {
	msgs := []chatmodel.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: chatmodel.MessageRoleUser, Content: "Hello"},
		{Role: chatmodel.MessageRoleAssistant, Content: "Hi there"},
		{Role: "system", Content: "Another system message"},
		{Role: chatmodel.MessageRoleUser, Content: "Bye"},
	}
	repo := &fakeConversationRepo{messages: msgs}
	c := NewContextCompiler(repo, TokenBudget{}, nil)

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countMessages(result, "system") != 0 {
		t.Error("system messages should be filtered out")
	}
	if len(result.Messages) != 3 {
		t.Errorf("expected 3 non-system messages, got %d", len(result.Messages))
	}
}

// ---------------------------------------------------------------------------
// Tests: Summary budget enforcement
// ---------------------------------------------------------------------------

func TestCompile_SummaryBudgetTruncatesSummary(t *testing.T) {
	msgs := makeAlternating(20)
	repo := &fakeConversationRepo{messages: msgs}

	// Very tight summary budget.
	c := NewContextCompiler(repo, TokenBudget{ConversationHistoryBudget: 5000}, nil,
		WithRecencyWindow(10),
		WithSummaryBudget(10), // ~40 characters
	)

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WasSummarised && result.SummaryTokens > 15 {
		// Allow some slack since truncation is approximate.
		t.Errorf("summary tokens %d should be near budget 10", result.SummaryTokens)
	}
}

// ---------------------------------------------------------------------------
// Tests: Accounting metadata
// ---------------------------------------------------------------------------

func TestCompile_AccountingMetadata(t *testing.T) {
	msgs := makeAlternating(15)
	repo := &fakeConversationRepo{messages: msgs}
	c := NewContextCompiler(repo, TokenBudget{}, nil, WithRecencyWindow(10))

	result, err := c.Compile(context.Background(), ptrUUID(uuid.New()), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ContextTurns != 10 {
		t.Errorf("expected ContextTurns=10, got %d", result.ContextTurns)
	}
	if result.TotalTokens <= 0 {
		t.Error("expected positive TotalTokens")
	}
	if result.DroppedTurns < 5 {
		t.Errorf("expected at least 5 dropped turns, got %d", result.DroppedTurns)
	}
}

// ---------------------------------------------------------------------------
// Tests: DefaultSummariser
// ---------------------------------------------------------------------------

func TestDefaultSummariser_Empty(t *testing.T) {
	s := &DefaultSummariser{}
	out, err := s.Summarise(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty string, got %q", out)
	}
}

func TestDefaultSummariser_TailSelection(t *testing.T) {
	msgs := makeAlternating(10)
	s := &DefaultSummariser{MaxMessages: 3, MaxCharsPer: 50}
	out, err := s.Summarise(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain content from the last 3 messages (indices 7, 8, 9).
	parts := strings.Split(out, " | ")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts in summary, got %d: %s", len(parts), out)
	}
}

func TestDefaultSummariser_Truncation(t *testing.T) {
	longMsg := chatmodel.Message{
		Role:    chatmodel.MessageRoleUser,
		Content: strings.Repeat("a", 300),
	}
	s := &DefaultSummariser{MaxCharsPer: 20}
	out, err := s.Summarise(context.Background(), []chatmodel.Message{longMsg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "User: " prefix + truncated content should be under ~30 chars.
	if len(out) > 30 {
		t.Errorf("expected truncated summary, got length %d", len(out))
	}
}

// ---------------------------------------------------------------------------
// Tests: Helper functions
// ---------------------------------------------------------------------------

func TestCapitalise(t *testing.T) {
	tests := []struct{ in, want string }{
		{"user", "User"},
		{"assistant", "Assistant"},
		{"", ""},
		{"A", "A"},
	}
	for _, tc := range tests {
		if got := capitalise(tc.in); got != tc.want {
			t.Errorf("capitalise(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		suffix string // expected suffix
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "…"},
		{"ab", 2, "ab"},
	}
	for _, tc := range tests {
		got := truncateStr(tc.input, tc.maxLen)
		if !strings.HasSuffix(got, tc.suffix) {
			t.Errorf("truncateStr(%q, %d) = %q, expected suffix %q", tc.input, tc.maxLen, got, tc.suffix)
		}
		if len(got) > tc.maxLen {
			t.Errorf("truncateStr(%q, %d) length %d exceeds max", tc.input, tc.maxLen, len(got))
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		min   int
		max   int
	}{
		{"", 0, 0},
		{"hi", 1, 1},
		{strings.Repeat("word ", 100), 100, 150},
	}
	for _, tc := range tests {
		got := estimateTokens(tc.input)
		if got < tc.min || got > tc.max {
			t.Errorf("estimateTokens(%d chars) = %d, expected [%d, %d]",
				len(tc.input), got, tc.min, tc.max)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Option validation
// ---------------------------------------------------------------------------

func TestWithRecencyWindow_IgnoresZero(t *testing.T) {
	c := NewContextCompiler(&fakeConversationRepo{}, TokenBudget{}, nil, WithRecencyWindow(0))
	if c.recencyWindow != DefaultRecencyWindow {
		t.Errorf("expected default %d, got %d", DefaultRecencyWindow, c.recencyWindow)
	}
}

func TestWithSummariser_IgnoresNil(t *testing.T) {
	c := NewContextCompiler(&fakeConversationRepo{}, TokenBudget{}, nil, WithSummariser(nil))
	if c.summariser == nil {
		t.Error("summariser should not be nil after passing nil option")
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkCompile_LargeHistory(b *testing.B) {
	msgs := makeAlternating(200)
	repo := &fakeConversationRepo{messages: msgs}
	c := NewContextCompiler(repo, TokenBudget{ConversationHistoryBudget: 4000}, nil,
		WithRecencyWindow(20),
	)
	convID := uuid.New()
	tenantID := uuid.New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Compile(ctx, &convID, tenantID)
	}
}
