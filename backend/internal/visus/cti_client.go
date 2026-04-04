package visus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type CTIClient struct {
	baseURL    string
	httpClient *http.Client
	logger     zerolog.Logger
}

type ctiEnvelope[T any] struct {
	Data T `json:"data"`
}

func NewCTIClient(cyberServiceURL string, logger zerolog.Logger) *CTIClient {
	return &CTIClient{
		baseURL: strings.TrimRight(cyberServiceURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger.With().Str("component", "visus_cti_client").Logger(),
	}
}

func (c *CTIClient) GetExecutiveDashboard(ctx context.Context, tenantID string, authToken string) (*CTIExecutiveDashboardResponse, error) {
	result, err := doCTIEnvelopedGet[CTIExecutiveDashboardResponse](c, ctx, tenantID, authToken, "/api/v1/cyber/cti/dashboard/executive", nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CTIClient) GetGlobalThreatMap(ctx context.Context, tenantID string, authToken string, period string) (*CTIGlobalThreatMapResponse, error) {
	query := url.Values{}
	if strings.TrimSpace(period) != "" {
		query.Set("period", strings.TrimSpace(period))
	}
	result, err := doCTIEnvelopedGet[CTIGlobalThreatMapResponse](c, ctx, tenantID, authToken, "/api/v1/cyber/cti/dashboard/threat-map", query)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CTIClient) GetSectorThreatOverview(ctx context.Context, tenantID string, authToken string, period string) (*CTISectorThreatResponse, error) {
	query := url.Values{}
	if strings.TrimSpace(period) != "" {
		query.Set("period", strings.TrimSpace(period))
	}
	result, err := doCTIEnvelopedGet[CTISectorThreatResponse](c, ctx, tenantID, authToken, "/api/v1/cyber/cti/dashboard/sectors", query)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CTIClient) GetActiveCampaigns(ctx context.Context, tenantID string, authToken string, limit int) (*CTICampaignListResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	query := url.Values{}
	query.Set("status", "active")
	query.Set("sort", "event_count")
	query.Set("order", "desc")
	query.Set("per_page", strconv.Itoa(limit))
	query.Set("page", "1")

	result, err := doCTIRawGet[CTICampaignListResponse](c, ctx, tenantID, authToken, "/api/v1/cyber/cti/campaigns", query)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CTIClient) GetCriticalBrandAbuse(ctx context.Context, tenantID string, authToken string, limit int) (*CTIBrandAbuseListResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	query := url.Values{}
	query.Set("risk_level", "critical")
	query.Set("sort", "last_detected_at")
	query.Set("order", "desc")
	query.Set("per_page", strconv.Itoa(limit))
	query.Set("page", "1")

	result, err := doCTIRawGet[CTIBrandAbuseListResponse](c, ctx, tenantID, authToken, "/api/v1/cyber/cti/brand-abuse", query)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CTIClient) GetThreatActors(ctx context.Context, tenantID string, authToken string, limit int) (*CTIActorListResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	query := url.Values{}
	query.Set("sort", "risk_score")
	query.Set("order", "desc")
	query.Set("per_page", strconv.Itoa(limit))
	query.Set("page", "1")
	query.Set("is_active", "true")

	result, err := doCTIRawGet[CTIActorListResponse](c, ctx, tenantID, authToken, "/api/v1/cyber/cti/actors", query)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func doCTIEnvelopedGet[T any](c *CTIClient, ctx context.Context, tenantID string, authToken string, path string, query url.Values) (*T, error) {
	body, err := c.doRequest(ctx, tenantID, authToken, path, query)
	if err != nil {
		return nil, err
	}

	var envelope ctiEnvelope[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("cti_client: decode enveloped response: %w", err)
	}
	return &envelope.Data, nil
}

func doCTIRawGet[T any](c *CTIClient, ctx context.Context, tenantID string, authToken string, path string, query url.Values) (*T, error) {
	body, err := c.doRequest(ctx, tenantID, authToken, path, query)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("cti_client: decode response: %w", err)
	}
	return &result, nil
}

func (c *CTIClient) doRequest(ctx context.Context, tenantID string, authToken string, path string, query url.Values) ([]byte, error) {
	tenantID = strings.TrimSpace(tenantID)
	authToken = strings.TrimSpace(authToken)
	if tenantID == "" {
		return nil, fmt.Errorf("cti_client: tenant id is required")
	}
	if authToken == "" {
		return nil, fmt.Errorf("cti_client: auth token is required")
	}

	requestURL := c.baseURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cti_client: create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Accept", "application/json")

	c.logger.Debug().
		Str("method", http.MethodGet).
		Str("url", requestURL).
		Str("tenant_id", tenantID).
		Msg("calling cyber-service CTI API")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().Err(err).Str("url", requestURL).Str("tenant_id", tenantID).Msg("cti_client: request failed")
		return nil, fmt.Errorf("cti_client: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if readErr != nil {
		c.logger.Error().Err(readErr).Str("url", requestURL).Str("tenant_id", tenantID).Msg("cti_client: read response failed")
		return nil, fmt.Errorf("cti_client: read response: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error().
			Int("status", resp.StatusCode).
			Str("url", requestURL).
			Str("tenant_id", tenantID).
			Str("body", string(body)).
			Msg("cti_client: non-200 response")
		return nil, fmt.Errorf("cti_client: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	c.logger.Debug().
		Str("url", requestURL).
		Str("tenant_id", tenantID).
		Msg("cti_client: success")

	return body, nil
}
