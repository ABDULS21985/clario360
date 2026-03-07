package connector

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

type APIConnector struct {
	config      model.APIConnectionConfig
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	logger      zerolog.Logger
}

func NewAPIConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.APIConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("decode api config: %w", err)
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base_url: %w", err)
	}
	if parsed.Scheme != "https" && !(cfg.AllowHTTP && parsed.Scheme == "http") {
		return nil, fmt.Errorf("base_url must use HTTPS unless allow_http is explicitly enabled")
	}
	if err := validateURLTarget(parsed, cfg); err != nil {
		return nil, err
	}
	if cfg.RateLimit <= 0 {
		cfg.RateLimit = options.Limits.APIRateLimit
	}

	client := &http.Client{
		Timeout: options.Limits.StatementTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) == 0 {
				return nil
			}
			for _, redirectReq := range via {
				redirectReq.Header.Del("Authorization")
			}
			if req.URL.Hostname() != via[0].URL.Hostname() {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	return &APIConnector{
		config:      cfg,
		httpClient:  client,
		rateLimiter: rate.NewLimiter(rate.Limit(cfg.RateLimit), cfg.RateLimit),
		logger:      options.Logger.With().Str("connector", "api").Logger(),
	}, nil
}

func (c *APIConnector) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	target := c.config.BaseURL
	if c.config.HealthURL != "" {
		target = c.config.HealthURL
	}

	start := time.Now()
	req, err := c.buildRequest(ctx, target, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return &ConnectionTestResult{
			Success:   true,
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("API responded with %s", resp.Status),
		}, nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return &ConnectionTestResult{
			Success:   false,
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   "API authentication failed.",
		}, nil
	default:
		return &ConnectionTestResult{
			Success:   false,
			LatencyMs: time.Since(start).Milliseconds(),
			Message:   fmt.Sprintf("API responded with %s", resp.Status),
		}, nil
	}
}

func (c *APIConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (*model.DiscoveredSchema, error) {
	req, err := c.buildRequest(ctx, c.config.BaseURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read api discovery payload: %w", err)
	}
	rows, err := c.extractRows(payload)
	if err != nil {
		return nil, err
	}
	if len(rows) > 10 {
		rows = rows[:10]
	}

	columns := inferJSONColumns(rows)
	columns = discovery.DetectPII(columns)
	table := model.DiscoveredTable{
		Name:            "api_response",
		Type:            "api",
		Columns:         columns,
		InferredClass:   discovery.TableClassification(columns),
		ContainsPII:     hasPII(columns),
		PIIColumnCount:  countPII(columns),
		SampledRowCount: len(rows),
	}

	return &model.DiscoveredSchema{
		Tables:       []model.DiscoveredTable{table},
		TableCount:   1,
		ColumnCount:  len(columns),
		ContainsPII:  table.ContainsPII,
		HighestClass: table.InferredClass,
	}, nil
}

func (c *APIConnector) FetchData(ctx context.Context, table string, params FetchParams) (*DataBatch, error) {
	queryParams := cloneStringMap(c.config.QueryParams)
	switch c.config.PaginationType {
	case model.APIPaginationOffset:
		queryParams["offset"] = strconv.FormatInt(params.Offset, 10)
		queryParams["limit"] = strconv.Itoa(defaultBatchSize(params.BatchSize))
	case model.APIPaginationCursor:
		if params.Cursor != "" {
			queryParams["cursor"] = params.Cursor
		}
	case model.APIPaginationPage:
		page := 1
		if params.BatchSize > 0 {
			page = int(params.Offset/int64(defaultBatchSize(params.BatchSize))) + 1
		}
		queryParams["page"] = strconv.Itoa(page)
		queryParams["per_page"] = strconv.Itoa(defaultBatchSize(params.BatchSize))
	}

	req, err := c.buildRequest(ctx, c.config.BaseURL, queryParams)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, fmt.Errorf("read api payload: %w", err)
	}
	rows, err := c.extractRows(payload)
	if err != nil {
		return nil, err
	}

	columns := make([]string, 0)
	if len(rows) > 0 {
		for key := range rows[0] {
			columns = append(columns, key)
		}
	}

	nextCursor := ""
	hasMore := false
	switch c.config.PaginationType {
	case model.APIPaginationCursor:
		if cursor, ok := extractStringPath(payload, anyStringValue(c.config.PaginationConfig["pagination_cursor_field"])); ok {
			nextCursor = cursor
			hasMore = cursor != ""
		}
	case model.APIPaginationLinkHeader:
		hasMore = strings.Contains(resp.Header.Get("Link"), `rel="next"`)
	case model.APIPaginationPage:
		currentPage, _ := extractIntPath(payload, "meta.page")
		totalPages, _ := extractIntPath(payload, "meta.total_pages")
		hasMore = totalPages > 0 && currentPage < totalPages
	default:
		hasMore = len(rows) == defaultBatchSize(params.BatchSize)
	}

	batchRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		batchRows = append(batchRows, row)
	}

	return &DataBatch{
		Columns:  columns,
		Rows:     batchRows,
		RowCount: len(batchRows),
		HasMore:  hasMore,
		Cursor:   nextCursor,
	}, nil
}

func (c *APIConnector) EstimateSize(ctx context.Context) (*SizeEstimate, error) {
	batch, err := c.FetchData(ctx, "api_response", FetchParams{BatchSize: 100})
	if err != nil {
		return nil, err
	}
	return &SizeEstimate{
		TableCount: 1,
		TotalRows:  int64(batch.RowCount),
		TotalBytes: int64(len(mustJSON(batch.Rows))),
	}, nil
}

func (c *APIConnector) ReadQuery(ctx context.Context, query string, args []any) (*DataBatch, error) {
	return nil, fmt.Errorf("%w: API connector does not support SQL query execution", ErrCapabilityUnsupported)
}

func (c *APIConnector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (*WriteResult, error) {
	return nil, fmt.Errorf("%w: API connector does not support write operations", ErrCapabilityUnsupported)
}

func (c *APIConnector) Close() error { return nil }

func (c *APIConnector) do(req *http.Request) (*http.Response, error) {
	if err := c.rateLimiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("wait for api rate limiter: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call source API: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized && c.config.AuthType == model.APIAuthOAuth2 {
		resp.Body.Close()
		if err := c.refreshOAuthToken(req.Context()); err != nil {
			return nil, err
		}
		retryReq, err := c.buildRequest(req.Context(), req.URL.String(), nil)
		if err != nil {
			return nil, err
		}
		return c.httpClient.Do(retryReq)
	}
	return resp, nil
}

func (c *APIConnector) buildRequest(ctx context.Context, rawURL string, queryParams map[string]string) (*http.Request, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse api url: %w", err)
	}
	query := parsed.Query()
	for key, value := range queryParams {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build api request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	switch c.config.AuthType {
	case model.APIAuthBasic:
		username, _ := c.config.AuthConfig["username"].(string)
		password, _ := c.config.AuthConfig["password"].(string)
		token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req.Header.Set("Authorization", "Basic "+token)
	case model.APIAuthBearer:
		token, _ := c.config.AuthConfig["token"].(string)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case model.APIAuthAPIKey:
		name, _ := c.config.AuthConfig["key_name"].(string)
		value, _ := c.config.AuthConfig["key_value"].(string)
		location, _ := c.config.AuthConfig["in"].(string)
		if strings.EqualFold(location, "query") {
			q := req.URL.Query()
			q.Set(name, value)
			req.URL.RawQuery = q.Encode()
		} else if name != "" && value != "" {
			req.Header.Set(name, value)
		}
	case model.APIAuthOAuth2:
		if token, _ := c.config.AuthConfig["access_token"].(string); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	return req, nil
}

func (c *APIConnector) extractRows(payload []byte) ([]map[string]any, error) {
	var parsed any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, fmt.Errorf("parse api JSON payload: %w", err)
	}
	value, ok := extractPath(parsed, c.config.DataPath)
	if !ok {
		value = parsed
	}
	array, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("configured data_path did not resolve to an array")
	}

	rows := make([]map[string]any, 0, len(array))
	for _, entry := range array {
		object, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		rows = append(rows, object)
	}
	return rows, nil
}

func (c *APIConnector) refreshOAuthToken(ctx context.Context) error {
	tokenURL, _ := c.config.AuthConfig["token_url"].(string)
	if tokenURL == "" {
		return fmt.Errorf("oauth2 token_url is required")
	}
	values := url.Values{}
	if refreshToken, _ := c.config.AuthConfig["refresh_token"].(string); refreshToken != "" {
		values.Set("grant_type", "refresh_token")
		values.Set("refresh_token", refreshToken)
	} else {
		values.Set("grant_type", "client_credentials")
	}
	if clientID, _ := c.config.AuthConfig["client_id"].(string); clientID != "" {
		values.Set("client_id", clientID)
	}
	if clientSecret, _ := c.config.AuthConfig["client_secret"].(string); clientSecret != "" {
		values.Set("client_secret", clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("build oauth2 token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("refresh oauth2 token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("refresh oauth2 token: provider returned %s", resp.Status)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decode oauth2 token response: %w", err)
	}
	accessToken, _ := body["access_token"].(string)
	if accessToken == "" {
		return fmt.Errorf("oauth2 token response did not contain access_token")
	}
	c.config.AuthConfig["access_token"] = accessToken
	if refreshToken, ok := body["refresh_token"].(string); ok && refreshToken != "" {
		c.config.AuthConfig["refresh_token"] = refreshToken
	}
	return nil
}

func inferJSONColumns(rows []map[string]any) []model.DiscoveredColumn {
	columns := make(map[string]*model.DiscoveredColumn)
	for _, row := range rows {
		for key, value := range row {
			column, ok := columns[key]
			if !ok {
				column = &model.DiscoveredColumn{
					Name:       key,
					DataType:   "json",
					NativeType: "json",
					MappedType: inferJSONValueType(value),
				}
				columns[key] = column
			}
			if value == nil {
				column.Nullable = true
				continue
			}
			column.SampleValues = append(column.SampleValues, fmt.Sprint(value))
		}
	}

	result := make([]model.DiscoveredColumn, 0, len(columns))
	for _, column := range columns {
		column.SampleStats = discovery.AnalyzeSamples(column.SampleValues)
		if inferred := discovery.InferSampleType(column.SampleValues); column.MappedType == "string" && inferred != "string" {
			column.MappedType = inferred
		}
		result = append(result, *column)
	}
	return result
}

func inferJSONValueType(value any) string {
	switch typed := value.(type) {
	case nil:
		return "string"
	case bool:
		return "boolean"
	case json.Number:
		if strings.Contains(typed.String(), ".") {
			return "float"
		}
		return "integer"
	case float32, float64:
		return "float"
	case int, int8, int16, int32, int64, uint, uint32, uint64:
		return "integer"
	case []any:
		return "array"
	case map[string]any:
		return "json"
	case string:
		if discovery.InferSampleType([]string{typed}) == "datetime" {
			return "datetime"
		}
		return "string"
	default:
		return "string"
	}
}

func validateURLTarget(parsed *url.URL, cfg model.APIConnectionConfig) error {
	host := parsed.Hostname()
	for _, allowlisted := range cfg.AllowlistedHosts {
		if strings.EqualFold(host, allowlisted) {
			return nil
		}
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			if !cfg.AllowPrivateAddresses {
				return fmt.Errorf("base_url resolves to a private address, which is not allowed")
			}
		}
		return nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve api host: %w", err)
	}
	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			if !cfg.AllowPrivateAddresses {
				return fmt.Errorf("base_url resolves to a private address, which is not allowed")
			}
		}
	}
	return nil
}

func extractPath(value any, path string) (any, bool) {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	if path == "" {
		return value, true
	}

	current := value
	for _, segment := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func extractStringPath(payload []byte, path string) (string, bool) {
	var parsed any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", false
	}
	value, ok := extractPath(parsed, path)
	if !ok {
		return "", false
	}
	result, ok := value.(string)
	return result, ok
}

func extractIntPath(payload []byte, path string) (int, bool) {
	var parsed any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return 0, false
	}
	value, ok := extractPath(parsed, path)
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case json.Number:
		parsed, err := typed.Int64()
		return int(parsed), err == nil
	default:
		return 0, false
	}
}

func countPII(columns []model.DiscoveredColumn) int {
	count := 0
	for _, column := range columns {
		if column.InferredPII {
			count++
		}
	}
	return count
}

func hasPII(columns []model.DiscoveredColumn) bool {
	return countPII(columns) > 0
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func defaultBatchSize(value int) int {
	if value <= 0 {
		return 100
	}
	return value
}

func anyStringValue(value any) string {
	text, _ := value.(string)
	return text
}

func mustJSON(value any) []byte {
	bytes, _ := json.Marshal(value)
	return bytes
}
