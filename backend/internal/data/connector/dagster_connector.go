package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

const dagsterConnectorType = "dagster"

type DagsterConnector struct {
	config   model.DagsterConnectionConfig
	client   *http.Client
	sourceID uuid.UUID
	tenantID uuid.UUID
	logger   zerolog.Logger
}

type LineageEdge struct {
	SourceAsset  string `json:"source_asset"`
	TargetAsset  string `json:"target_asset"`
	Type         string `json:"type"`
	PipelineName string `json:"pipeline_name"`
}

func NewDagsterConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.DagsterConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, newConnectorError(dagsterConnectorType, "connect", ErrorCodeConfigurationError, "decode dagster config", err)
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = int(options.Limits.StatementTimeout.Seconds())
		if cfg.TimeoutSeconds <= 0 {
			cfg.TimeoutSeconds = 30
		}
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, newConnectorError(dagsterConnectorType, "connect", ErrorCodeConfigurationError, "validate dagster config", err)
	}

	connector := &DagsterConnector{
		config: cfg,
		client: &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second},
		logger: options.Logger.With().Str("connector", dagsterConnectorType).Logger(),
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(dagsterConnectorType).Inc()
	return connector, nil
}

func (c *DagsterConnector) SetSourceContext(sourceID, tenantID uuid.UUID) {
	c.sourceID = sourceID
	c.tenantID = tenantID
}

func (c *DagsterConnector) TestConnection(ctx context.Context) (_ *ConnectionTestResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(dagsterConnectorType, "test", start, err) }()

	var versionResp struct {
		Version string `json:"version"`
	}
	if err = c.queryGraphQL(ctx, `query { version }`, nil, &versionResp); err != nil {
		return nil, newConnectorError(dagsterConnectorType, "test", ErrorCodeConnectionFailed, "query dagster version", err)
	}

	var repoResp struct {
		RepositoriesOrError struct {
			Nodes []struct {
				Name string `json:"name"`
			} `json:"nodes"`
		} `json:"repositoriesOrError"`
	}
	if err = c.queryGraphQL(ctx, `query { repositoriesOrError { ... on RepositoryConnection { nodes { name } } } }`, nil, &repoResp); err != nil {
		return nil, newConnectorError(dagsterConnectorType, "test", ErrorCodeConnectionFailed, "query dagster repositories", err)
	}

	repositories := make([]string, 0, len(repoResp.RepositoriesOrError.Nodes))
	for _, node := range repoResp.RepositoriesOrError.Nodes {
		repositories = append(repositories, node.Name)
	}

	return &ConnectionTestResult{
		Success:     true,
		LatencyMs:   time.Since(start).Milliseconds(),
		Version:     versionResp.Version,
		Message:     fmt.Sprintf("Connected to Dagster. %d repositories accessible.", len(repositories)),
		Permissions: repositories,
	}, nil
}

func (c *DagsterConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (_ *model.DiscoveredSchema, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(dagsterConnectorType, "discover", start, err) }()

	var resp struct {
		AssetsOrError struct {
			Nodes []struct {
				Key struct {
					Path []string `json:"path"`
				} `json:"key"`
				Description   string `json:"description"`
				GraphName     string `json:"graphName"`
				ComputeKind   string `json:"computeKind"`
				IsPartitioned bool   `json:"isPartitioned"`
				PartitionDef  *struct {
					Name string `json:"name"`
				} `json:"partitionDefinition"`
				MetadataEntries []struct {
					Label       string `json:"label"`
					Description string `json:"description"`
					Text        string `json:"text"`
				} `json:"metadataEntries"`
			} `json:"nodes"`
		} `json:"assetsOrError"`
	}
	query := `
		query {
		  assetsOrError {
		    ... on AssetConnection {
		      nodes {
		        key { path }
		        description
		        graphName
		        computeKind
		        isPartitioned
		        partitionDefinition { name }
		        metadataEntries {
		          label
		          description
		          __typename
		          ... on TextMetadataEntry { text }
		        }
		      }
		    }
		  }
		}
	`
	if err = c.queryGraphQL(ctx, query, nil, &resp); err != nil {
		return nil, newConnectorError(dagsterConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "query dagster assets", err)
	}

	limit := len(resp.AssetsOrError.Nodes)
	if opts.MaxTables > 0 && opts.MaxTables < limit {
		limit = opts.MaxTables
	}
	tables := make([]model.DiscoveredTable, 0, limit)
	for _, asset := range resp.AssetsOrError.Nodes[:limit] {
		name := strings.Join(asset.Key.Path, ".")
		description := strings.TrimSpace(asset.Description)
		if asset.ComputeKind != "" {
			if description != "" {
				description += " | "
			}
			description += "compute kind: " + asset.ComputeKind
		}
		if asset.IsPartitioned && asset.PartitionDef != nil && asset.PartitionDef.Name != "" {
			if description != "" {
				description += " | "
			}
			description += "partitioned by " + asset.PartitionDef.Name
		}
		tables = append(tables, model.DiscoveredTable{
			Name:           name,
			Type:           "asset",
			Comment:        description,
			Columns:        []model.DiscoveredColumn{},
			InferredClass:  model.DataClassificationInternal,
			ContainsPII:    false,
			PIIColumnCount: 0,
		})
	}
	observeSchemaMetrics(dagsterConnectorType, tables)
	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  0,
		ContainsPII:  false,
		HighestClass: model.DataClassificationInternal,
	}, nil
}

func (c *DagsterConnector) FetchData(ctx context.Context, table string, params FetchParams) (*DataBatch, error) {
	_ = ctx
	_ = table
	_ = params
	return nil, newConnectorError(dagsterConnectorType, "fetch", ErrorCodeUnsupportedOperation, "Dagster orchestrates pipelines and does not support direct data fetch", ErrCapabilityUnsupported)
}

func (c *DagsterConnector) ReadQuery(ctx context.Context, query string, args []any) (*DataBatch, error) {
	_ = ctx
	_ = query
	_ = args
	return nil, newConnectorError(dagsterConnectorType, "read_query", ErrorCodeUnsupportedOperation, "Dagster does not expose SQL querying", ErrCapabilityUnsupported)
}

func (c *DagsterConnector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (*WriteResult, error) {
	_ = ctx
	_ = table
	_ = rows
	_ = params
	return nil, newConnectorError(dagsterConnectorType, "write", ErrorCodeUnsupportedOperation, "Dagster does not support writes", ErrCapabilityUnsupported)
}

func (c *DagsterConnector) EstimateSize(ctx context.Context) (_ *SizeEstimate, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(dagsterConnectorType, "estimate", start, err) }()

	schema, err := c.DiscoverSchema(ctx, DiscoveryOptions{MaxTables: 500})
	if err != nil {
		return nil, err
	}
	return &SizeEstimate{TableCount: schema.TableCount, TotalRows: int64(schema.TableCount)}, nil
}

func (c *DagsterConnector) QueryAccessLogs(ctx context.Context, since time.Time) (_ []DataAccessEvent, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(dagsterConnectorType, "access_logs", start, err) }()

	query := `
		query($updatedAfter: Float!) {
		  runsOrError(filter: {updatedAfter: $updatedAfter}, limit: 1000) {
		    ... on Runs {
		      results {
		        runId
		        pipelineName
		        status
		        startTime
		        endTime
		        tags { key value }
		      }
		    }
		  }
		}
	`
	var resp struct {
		RunsOrError struct {
			Results []struct {
				RunID        string  `json:"runId"`
				PipelineName string  `json:"pipelineName"`
				Status       string  `json:"status"`
				StartTime    float64 `json:"startTime"`
				EndTime      float64 `json:"endTime"`
				Tags         []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"tags"`
			} `json:"results"`
		} `json:"runsOrError"`
	}
	if err = c.queryGraphQL(ctx, query, map[string]any{"updatedAfter": float64(since.Unix())}, &resp); err != nil {
		return nil, newConnectorError(dagsterConnectorType, "access_logs", ErrorCodeQueryFailed, "query dagster run history", err)
	}

	events := make([]DataAccessEvent, 0, len(resp.RunsOrError.Results))
	for _, run := range resp.RunsOrError.Results {
		user := "unknown"
		for _, tag := range run.Tags {
			switch strings.ToLower(tag.Key) {
			case "dagster/user", "launched_by", "user":
				if strings.TrimSpace(tag.Value) != "" {
					user = tag.Value
				}
			}
		}
		startAt := unixFloat(run.StartTime)
		endAt := unixFloat(run.EndTime)
		preview := truncateString(run.PipelineName+" "+run.Status, 500)
		event := DataAccessEvent{
			Timestamp:    endAt,
			User:         user,
			Action:       "pipeline_run",
			Database:     run.PipelineName,
			QueryHash:    sha256Hex(run.RunID + run.PipelineName + run.Status),
			QueryPreview: preview,
			DurationMs:   endAt.Sub(startAt).Milliseconds(),
			Success:      strings.EqualFold(run.Status, "SUCCESS"),
			SourceType:   dagsterConnectorType,
			SourceID:     c.sourceID,
			TenantID:     c.tenantID,
		}
		if !event.Success {
			event.ErrorMsg = truncateString(run.Status, 200)
		}
		events = append(events, event)
	}
	getConnectorMetrics().AccessEventsTotal.WithLabelValues(dagsterConnectorType).Add(float64(len(events)))
	return events, nil
}

func (c *DagsterConnector) ListDataLocations(ctx context.Context) (_ []DataLocation, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(dagsterConnectorType, "locations", start, err) }()

	var resp struct {
		AssetsOrError struct {
			Nodes []struct {
				Key struct {
					Path []string `json:"path"`
				} `json:"key"`
				MetadataEntries []struct {
					Label string `json:"label"`
					Text  string `json:"text"`
				} `json:"metadataEntries"`
			} `json:"nodes"`
		} `json:"assetsOrError"`
	}
	if err = c.queryGraphQL(ctx, `query { assetsOrError { ... on AssetConnection { nodes { key { path } metadataEntries { label ... on TextMetadataEntry { text } } } } } }`, nil, &resp); err != nil {
		return nil, newConnectorError(dagsterConnectorType, "locations", ErrorCodeQueryFailed, "query dagster assets for locations", err)
	}

	locations := make([]DataLocation, 0, len(resp.AssetsOrError.Nodes))
	now := time.Now().UTC()
	for _, node := range resp.AssetsOrError.Nodes {
		location := "dagster://asset/" + strings.Join(node.Key.Path, "/")
		format := "managed"
		for _, entry := range node.MetadataEntries {
			label := strings.ToLower(entry.Label)
			if strings.Contains(label, "path") || strings.Contains(label, "location") {
				if strings.TrimSpace(entry.Text) != "" {
					location = entry.Text
				}
			}
			if strings.Contains(label, "format") && strings.TrimSpace(entry.Text) != "" {
				format = entry.Text
			}
		}
		locations = append(locations, DataLocation{
			SourceID:     c.sourceID,
			SourceType:   dagsterConnectorType,
			Table:        strings.Join(node.Key.Path, "."),
			Database:     c.config.Workspace,
			Location:     location,
			Format:       format,
			LastModified: now,
		})
	}
	return locations, nil
}

func (c *DagsterConnector) GetAssetLineage(ctx context.Context) (_ []LineageEdge, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(dagsterConnectorType, "lineage", start, err) }()

	query := `
		query {
		  assetsOrError {
		    ... on AssetConnection {
		      nodes {
		        key { path }
		        graphName
		        dependencyKeys { path }
		      }
		    }
		  }
		}
	`
	var resp struct {
		AssetsOrError struct {
			Nodes []struct {
				Key struct {
					Path []string `json:"path"`
				} `json:"key"`
				GraphName      string `json:"graphName"`
				DependencyKeys []struct {
					Path []string `json:"path"`
				} `json:"dependencyKeys"`
			} `json:"nodes"`
		} `json:"assetsOrError"`
	}
	if err = c.queryGraphQL(ctx, query, nil, &resp); err != nil {
		return nil, newConnectorError(dagsterConnectorType, "lineage", ErrorCodeQueryFailed, "query dagster lineage", err)
	}

	edges := make([]LineageEdge, 0)
	for _, node := range resp.AssetsOrError.Nodes {
		target := strings.Join(node.Key.Path, ".")
		for _, dep := range node.DependencyKeys {
			edges = append(edges, LineageEdge{
				SourceAsset:  strings.Join(dep.Path, "."),
				TargetAsset:  target,
				Type:         "dagster_dependency",
				PipelineName: node.GraphName,
			})
		}
	}
	return edges, nil
}

func (c *DagsterConnector) Close() error {
	if c.client != nil {
		c.client.CloseIdleConnections()
		c.client = nil
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(dagsterConnectorType).Dec()
	return nil
}

func (c *DagsterConnector) queryGraphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.GraphQLURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.config.APIToken) != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payloadBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("dagster graphql status %s: %s", resp.Status, truncateString(string(payloadBytes), 500))
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(payloadBytes, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("dagster graphql error: %s", envelope.Errors[0].Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(envelope.Data, out)
}

func unixFloat(value float64) time.Time {
	if value <= 0 {
		return time.Now().UTC()
	}
	seconds := int64(value)
	nanos := int64((value - float64(seconds)) * float64(time.Second))
	return time.Unix(seconds, nanos).UTC()
}
