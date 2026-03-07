package scanner

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// CloudScanner discovers assets from cloud provider APIs (AWS, GCP, Azure).
// Credentials are loaded from cfg.Options or standard environment variables.
//
// Supported providers (cfg.Options["provider"]):
//   - "aws"   — AWS EC2 via DescribeInstances (SigV4 signed)
//   - "gcp"   — GCP Compute Engine via aggregated instances (service account JWT)
//   - "azure" — Azure Virtual Machines via ARM API (service principal OAuth2)
type CloudScanner struct {
	repo   AssetUpsertRepo
	client *http.Client
	logger zerolog.Logger
}

// NewCloudScanner creates a CloudScanner.
func NewCloudScanner(repo AssetUpsertRepo, logger zerolog.Logger) *CloudScanner {
	return &CloudScanner{
		repo:   repo,
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

// Type implements Scanner.
func (s *CloudScanner) Type() model.ScanType { return model.ScanTypeCloud }

// Scan performs cloud asset discovery against all specified targets.
// cfg.Targets contains account / project / subscription IDs.
// cfg.Options must include "provider" and provider-specific credential keys.
func (s *CloudScanner) Scan(ctx context.Context, cfg *model.ScanConfig) (*model.ScanResult, error) {
	if len(cfg.Targets) == 0 {
		return &model.ScanResult{
			Status: model.ScanStatusFailed,
			Errors: []string{"no cloud targets specified"},
		}, fmt.Errorf("cloud scan requires at least one account/project ID in targets")
	}

	provider := "aws"
	if p, ok := cfg.Options["provider"].(string); ok && p != "" {
		provider = strings.ToLower(strings.TrimSpace(p))
	}

	tenantID := tenantIDFromCtx(ctx)

	var allAssets []*model.DiscoveredAsset
	var scanErrors []string

	for _, target := range cfg.Targets {
		var discovered []*model.DiscoveredAsset
		var err error

		switch provider {
		case "aws":
			discovered, err = s.scanAWS(ctx, target, cfg.Options)
		case "gcp":
			discovered, err = s.scanGCP(ctx, target, cfg.Options)
		case "azure":
			discovered, err = s.scanAzure(ctx, target, cfg.Options)
		default:
			err = fmt.Errorf("unsupported cloud provider %q — must be aws, gcp, or azure", provider)
		}

		if err != nil {
			msg := fmt.Sprintf("provider=%s target=%s: %v", provider, target, err)
			scanErrors = append(scanErrors, msg)
			s.logger.Warn().Err(err).Str("provider", provider).Str("target", target).Msg("cloud scan target failed")
			continue
		}

		s.logger.Info().
			Str("provider", provider).
			Str("target", target).
			Int("discovered", len(discovered)).
			Msg("cloud scan target completed")

		allAssets = append(allAssets, discovered...)
	}

	assetsNew, assetsUpdated := 0, 0
	for _, d := range allAssets {
		assetID, isNew, err := s.repo.UpsertFromScan(ctx, tenantID, d)
		if err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("upsert %s: %v", d.IPAddress, err))
			continue
		}
		d.AssetID = assetID
		d.IsNew = isNew
		if isNew {
			assetsNew++
		} else {
			assetsUpdated++
		}
	}

	status := model.ScanStatusCompleted
	if len(scanErrors) > 0 && len(allAssets) == 0 {
		status = model.ScanStatusFailed
	}

	return &model.ScanResult{
		Status:           status,
		AssetsDiscovered: len(allAssets),
		AssetsNew:        assetsNew,
		AssetsUpdated:    assetsUpdated,
		Errors:           scanErrors,
	}, nil
}

// ────────────────────────────────────────────────────────────────────────────
// AWS
// ────────────────────────────────────────────────────────────────────────────

type awsCreds struct {
	accessKeyID     string
	secretAccessKey string
	sessionToken    string // optional — set when using IAM role temporary credentials
	region          string
}

func loadAWSCreds(opts map[string]any) (awsCreds, error) {
	get := func(optKey, envKey string) string {
		if v, ok := opts[optKey].(string); ok && v != "" {
			return v
		}
		return os.Getenv(envKey)
	}

	creds := awsCreds{
		accessKeyID:     get("aws_access_key_id", "AWS_ACCESS_KEY_ID"),
		secretAccessKey: get("aws_secret_access_key", "AWS_SECRET_ACCESS_KEY"),
		sessionToken:    get("aws_session_token", "AWS_SESSION_TOKEN"),
		region:          get("aws_region", "AWS_REGION"),
	}
	if creds.region == "" {
		creds.region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if creds.region == "" {
		creds.region = "us-east-1"
	}
	if creds.accessKeyID == "" || creds.secretAccessKey == "" {
		return awsCreds{}, fmt.Errorf("AWS credentials not configured: set aws_access_key_id and aws_secret_access_key in scan options or AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY environment variables")
	}
	return creds, nil
}

// ec2Instance holds the fields we parse from an EC2 DescribeInstances XML response.
type ec2Instance struct {
	InstanceID    string
	PrivateIP     string
	PublicIP      string
	PrivateDNS    string
	PublicDNS     string
	InstanceType  string
	Platform      string // "windows" or empty (Linux)
	Architecture  string
	Region        string
	Zone          string
	MACAddress    string
	Tags          map[string]string
	InstanceState string
}

// AWS EC2 DescribeInstances XML structures.
type ec2DescribeInstancesResponse struct {
	XMLName        xml.Name         `xml:"DescribeInstancesResponse"`
	ReservationSet []ec2Reservation `xml:"reservationSet>item"`
	NextToken      string           `xml:"nextToken"`
}

type ec2Reservation struct {
	Instances []ec2InstanceXML `xml:"instancesSet>item"`
}

type ec2InstanceXML struct {
	InstanceID    string       `xml:"instanceId"`
	InstanceType  string       `xml:"instanceType"`
	PrivateDNS    string       `xml:"privateDnsName"`
	PublicDNS     string       `xml:"dnsName"`
	PrivateIP     string       `xml:"privateIpAddress"`
	PublicIP      string       `xml:"ipAddress"`
	Platform      string       `xml:"platform"` // "windows" only for Windows
	Architecture  string       `xml:"architecture"`
	State         ec2StateXML  `xml:"instanceState"`
	Placement     ec2Placement `xml:"placement"`
	Tags          []ec2TagXML  `xml:"tagSet>item"`
	NetworkIfaces []ec2NICXML  `xml:"networkInterfaceSet>item"`
}

type ec2StateXML struct {
	Name string `xml:"name"`
}

type ec2Placement struct {
	Zone string `xml:"availabilityZone"`
}

type ec2TagXML struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type ec2NICXML struct {
	MAC string `xml:"macAddress"`
}

func (s *CloudScanner) scanAWS(ctx context.Context, accountID string, opts map[string]any) ([]*model.DiscoveredAsset, error) {
	creds, err := loadAWSCreds(opts)
	if err != nil {
		return nil, err
	}

	s.logger.Info().Str("region", creds.region).Str("account", accountID).Msg("starting AWS EC2 discovery")

	var allInstances []ec2Instance
	var nextToken string

	for {
		instances, token, err := s.describeEC2Instances(ctx, creds, nextToken)
		if err != nil {
			return nil, fmt.Errorf("describe instances: %w", err)
		}
		allInstances = append(allInstances, instances...)
		if token == "" {
			break
		}
		nextToken = token
	}

	assets := make([]*model.DiscoveredAsset, 0, len(allInstances))
	for _, inst := range allInstances {
		assets = append(assets, ec2InstanceToAsset(inst, creds.region, accountID))
	}
	return assets, nil
}

func (s *CloudScanner) describeEC2Instances(ctx context.Context, creds awsCreds, nextToken string) ([]ec2Instance, string, error) {
	endpoint := fmt.Sprintf("https://ec2.%s.amazonaws.com/", creds.region)

	params := url.Values{}
	params.Set("Action", "DescribeInstances")
	params.Set("Version", "2016-11-15")
	// Only return running instances.
	params.Set("Filter.1.Name", "instance-state-name")
	params.Set("Filter.1.Value.1", "running")
	if nextToken != "" {
		params.Set("NextToken", nextToken)
	}
	params.Set("MaxResults", "200")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("build EC2 request: %w", err)
	}

	awsSignV4(req, creds, "ec2", nil)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("EC2 API call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB cap
	if err != nil {
		return nil, "", fmt.Errorf("read EC2 response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("EC2 API error %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var parsed ec2DescribeInstancesResponse
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("parse EC2 XML: %w", err)
	}

	var instances []ec2Instance
	for _, res := range parsed.ReservationSet {
		for _, xi := range res.Instances {
			if xi.State.Name != "running" {
				continue
			}
			tags := make(map[string]string, len(xi.Tags))
			for _, t := range xi.Tags {
				tags[t.Key] = t.Value
			}
			var mac string
			if len(xi.NetworkIfaces) > 0 {
				mac = xi.NetworkIfaces[0].MAC
			}
			os := "Linux"
			if strings.EqualFold(xi.Platform, "windows") {
				os = "Windows"
			}
			instances = append(instances, ec2Instance{
				InstanceID:    xi.InstanceID,
				PrivateIP:     xi.PrivateIP,
				PublicIP:      xi.PublicIP,
				PrivateDNS:    xi.PrivateDNS,
				PublicDNS:     xi.PublicDNS,
				InstanceType:  xi.InstanceType,
				Platform:      os,
				Architecture:  xi.Architecture,
				Zone:          xi.Placement.Zone,
				MACAddress:    mac,
				Tags:          tags,
				InstanceState: xi.State.Name,
			})
		}
	}
	return instances, parsed.NextToken, nil
}

func ec2InstanceToAsset(inst ec2Instance, region, accountID string) *model.DiscoveredAsset {
	name, hasDNS := inst.Tags["Name"], inst.PrivateDNS != "" || inst.PublicDNS != ""
	_ = hasDNS

	ip := inst.PrivateIP
	if ip == "" {
		ip = inst.PublicIP
	}

	var hostname *string
	if inst.PrivateDNS != "" {
		hostname = &inst.PrivateDNS
	} else if inst.PublicDNS != "" {
		hostname = &inst.PublicDNS
	}

	var mac *string
	if inst.MACAddress != "" {
		mac = &inst.MACAddress
	}

	osName := inst.Platform
	var osVal *string
	if osName != "" {
		osVal = &osName
	}

	tagSlice := make([]string, 0, len(inst.Tags))
	for k, v := range inst.Tags {
		tagSlice = append(tagSlice, k+":"+v)
	}
	if name != "" {
		tagSlice = append(tagSlice, "name:"+name)
	}

	extra := map[string]any{
		"cloud_provider":    "aws",
		"aws_account_id":    accountID,
		"aws_region":        region,
		"aws_zone":          inst.Zone,
		"aws_instance_id":   inst.InstanceID,
		"aws_instance_type": inst.InstanceType,
		"aws_architecture":  inst.Architecture,
	}
	for k, v := range inst.Tags {
		extra["aws_tag_"+strings.ToLower(k)] = v
	}

	return &model.DiscoveredAsset{
		IPAddress:       ip,
		Hostname:        hostname,
		OS:              osVal,
		MACAddress:      mac,
		AssetType:       model.AssetTypeCloudResource,
		OpenPorts:       []int{},
		Banners:         map[int]string{},
		ExtraMetadata:   extra,
		DiscoverySource: "cloud_scan",
	}
}

// awsSignV4 signs an HTTP request using AWS Signature Version 4.
// It mutates req by adding the Authorization, X-Amz-Date, and optionally
// X-Amz-Security-Token headers in-place.
func awsSignV4(req *http.Request, creds awsCreds, service string, body []byte) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("X-Amz-Date", amzDate)
	if creds.sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", creds.sessionToken)
	}

	// ── 1. Canonical Request ──────────────────────────────────────────────────
	host := req.URL.Host
	req.Header.Set("Host", host)

	// Sorted canonical headers.
	signedHeaderKeys := []string{"host", "x-amz-date"}
	if creds.sessionToken != "" {
		signedHeaderKeys = append(signedHeaderKeys, "x-amz-security-token")
	}
	sort.Strings(signedHeaderKeys)

	var canonHeaders strings.Builder
	for _, k := range signedHeaderKeys {
		canonHeaders.WriteString(k)
		canonHeaders.WriteString(":")
		canonHeaders.WriteString(strings.TrimSpace(req.Header.Get(http.CanonicalHeaderKey(k))))
		canonHeaders.WriteString("\n")
	}

	payloadHash := hashSHA256(body)

	// Canonical query string — parameters must be sorted.
	queryParams := req.URL.Query()
	queryKeys := make([]string, 0, len(queryParams))
	for k := range queryParams {
		queryKeys = append(queryKeys, k)
	}
	sort.Strings(queryKeys)
	var canonQuery strings.Builder
	for i, k := range queryKeys {
		if i > 0 {
			canonQuery.WriteString("&")
		}
		canonQuery.WriteString(url.QueryEscape(k))
		canonQuery.WriteString("=")
		canonQuery.WriteString(url.QueryEscape(queryParams.Get(k)))
	}

	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		canonQuery.String(),
		canonHeaders.String(),
		strings.Join(signedHeaderKeys, ";"),
		payloadHash,
	}, "\n")

	// ── 2. String To Sign ────────────────────────────────────────────────────
	credentialScope := strings.Join([]string{dateStamp, creds.region, service, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hashSHA256([]byte(canonicalRequest)),
	}, "\n")

	// ── 3. Derived Signing Key ───────────────────────────────────────────────
	kDate := hmacSHA256([]byte("AWS4"+creds.secretAccessKey), dateStamp)
	kRegion := hmacSHA256(kDate, creds.region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "aws4_request")

	// ── 4. Authorization Header ──────────────────────────────────────────────
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))
	authorization := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		creds.accessKeyID,
		credentialScope,
		strings.Join(signedHeaderKeys, ";"),
		signature,
	)
	req.Header.Set("Authorization", authorization)
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func hashSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// ────────────────────────────────────────────────────────────────────────────
// GCP
// ────────────────────────────────────────────────────────────────────────────

type gcpServiceAccountKey struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

func loadGCPCreds(projectID string, opts map[string]any) (gcpServiceAccountKey, error) {
	var raw string

	// Prefer inline JSON from scan options.
	if v, ok := opts["gcp_service_account_json"].(string); ok && v != "" {
		raw = v
	} else if path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return gcpServiceAccountKey{}, fmt.Errorf("read GOOGLE_APPLICATION_CREDENTIALS %q: %w", path, err)
		}
		raw = string(data)
	} else {
		return gcpServiceAccountKey{}, fmt.Errorf("GCP credentials not configured: set gcp_service_account_json in scan options or GOOGLE_APPLICATION_CREDENTIALS environment variable")
	}

	var key gcpServiceAccountKey
	if err := json.Unmarshal([]byte(raw), &key); err != nil {
		return gcpServiceAccountKey{}, fmt.Errorf("parse GCP service account JSON: %w", err)
	}
	if key.PrivateKey == "" || key.ClientEmail == "" {
		return gcpServiceAccountKey{}, fmt.Errorf("GCP service account JSON missing private_key or client_email")
	}
	if projectID != "" {
		key.ProjectID = projectID
	}
	return key, nil
}

func (s *CloudScanner) scanGCP(ctx context.Context, projectID string, opts map[string]any) ([]*model.DiscoveredAsset, error) {
	creds, err := loadGCPCreds(projectID, opts)
	if err != nil {
		return nil, err
	}

	s.logger.Info().Str("project", creds.ProjectID).Msg("starting GCP Compute Engine discovery")

	token, err := gcpGetAccessToken(ctx, creds, s.client)
	if err != nil {
		return nil, fmt.Errorf("obtain GCP access token: %w", err)
	}

	var allAssets []*model.DiscoveredAsset
	pageToken := ""
	for {
		assets, next, err := s.gcpListInstances(ctx, creds.ProjectID, token, pageToken)
		if err != nil {
			return nil, err
		}
		allAssets = append(allAssets, assets...)
		if next == "" {
			break
		}
		pageToken = next
	}
	return allAssets, nil
}

func gcpGetAccessToken(ctx context.Context, creds gcpServiceAccountKey, client *http.Client) (string, error) {
	now := time.Now().Unix()
	claims := map[string]any{
		"iss":   creds.ClientEmail,
		"scope": "https://www.googleapis.com/auth/compute.readonly",
		"aud":   "https://oauth2.googleapis.com/token",
		"exp":   now + 3600,
		"iat":   now,
	}

	jwt, err := gcpSignJWT(creds.PrivateKey, claims)
	if err != nil {
		return "", fmt.Errorf("sign GCP JWT: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", jwt)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build GCP token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GCP token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GCP token endpoint error %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse GCP token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("GCP returned empty access_token")
	}
	return tokenResp.AccessToken, nil
}

// gcpSignJWT creates a JWT signed with RS256 using the service account's PEM private key.
func gcpSignJWT(privateKeyPEM string, claims map[string]any) (string, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("invalid PEM block in GCP private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fallback: try PKCS1.
		rsaKey, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return "", fmt.Errorf("parse GCP private key (PKCS8: %v, PKCS1: %v)", err, err2)
		}
		key = rsaKey
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("GCP private key is not RSA")
	}

	header := base64.RawURLEncoding.EncodeToString(mustJSON(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}))
	payload := base64.RawURLEncoding.EncodeToString(mustJSON(claims))
	sigInput := header + "." + payload

	h := crypto.SHA256.New()
	h.Write([]byte(sigInput))
	digest := h.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return sigInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (s *CloudScanner) gcpListInstances(ctx context.Context, projectID, token, pageToken string) ([]*model.DiscoveredAsset, string, error) {
	u := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/aggregated/instances", url.PathEscape(projectID))
	if pageToken != "" {
		u += "?pageToken=" + url.QueryEscape(pageToken)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build GCP instances request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("GCP instances API: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("GCP instances error %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var result struct {
		Items map[string]struct {
			Instances []gcpInstance `json:"instances"`
		} `json:"items"`
		NextPageToken string `json:"nextPageToken"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", fmt.Errorf("parse GCP instances: %w", err)
	}

	var assets []*model.DiscoveredAsset
	for zone, zoneData := range result.Items {
		for _, inst := range zoneData.Instances {
			if strings.ToUpper(inst.Status) != "RUNNING" {
				continue
			}
			assets = append(assets, gcpInstanceToAsset(inst, projectID, zone))
		}
	}
	return assets, result.NextPageToken, nil
}

type gcpInstance struct {
	Name              string            `json:"name"`
	Status            string            `json:"status"`
	MachineType       string            `json:"machineType"`
	Zone              string            `json:"zone"`
	NetworkInterfaces []gcpNIC          `json:"networkInterfaces"`
	Disks             []gcpDisk         `json:"disks"`
	Labels            map[string]string `json:"labels"`
}

type gcpNIC struct {
	Name          string         `json:"name"`
	NetworkIP     string         `json:"networkIP"`
	AccessConfigs []gcpAccessCfg `json:"accessConfigs"`
}

type gcpAccessCfg struct {
	NatIP string `json:"natIP"` // public IP (if assigned)
}

type gcpDisk struct {
	Boot bool   `json:"boot"`
	Type string `json:"type"`
}

func gcpInstanceToAsset(inst gcpInstance, projectID, zone string) *model.DiscoveredAsset {
	ip := ""
	publicIP := ""
	if len(inst.NetworkInterfaces) > 0 {
		ip = inst.NetworkInterfaces[0].NetworkIP
		for _, ac := range inst.NetworkInterfaces[0].AccessConfigs {
			if ac.NatIP != "" {
				publicIP = ac.NatIP
				break
			}
		}
	}
	if ip == "" {
		ip = publicIP
	}

	hostname := inst.Name
	var hostnamePtr *string
	if hostname != "" {
		hostnamePtr = &hostname
	}

	// Derive machine type name from URL (last path segment).
	machineType := inst.MachineType
	if idx := strings.LastIndex(machineType, "/"); idx >= 0 {
		machineType = machineType[idx+1:]
	}

	// Derive zone from URL if needed.
	zoneName := zone
	if idx := strings.LastIndex(zoneName, "/"); idx >= 0 {
		zoneName = zoneName[idx+1:]
	}

	extra := map[string]any{
		"cloud_provider":    "gcp",
		"gcp_project_id":    projectID,
		"gcp_zone":          zoneName,
		"gcp_instance_name": inst.Name,
		"gcp_machine_type":  machineType,
	}
	if publicIP != "" {
		extra["gcp_public_ip"] = publicIP
	}
	for k, v := range inst.Labels {
		extra["gcp_label_"+k] = v
	}

	return &model.DiscoveredAsset{
		IPAddress:       ip,
		Hostname:        hostnamePtr,
		AssetType:       model.AssetTypeCloudResource,
		OpenPorts:       []int{},
		Banners:         map[int]string{},
		ExtraMetadata:   extra,
		DiscoverySource: "cloud_scan",
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Azure
// ────────────────────────────────────────────────────────────────────────────

type azureCreds struct {
	subscriptionID string
	tenantID       string
	clientID       string
	clientSecret   string
}

func loadAzureCreds(subscriptionID string, opts map[string]any) (azureCreds, error) {
	get := func(optKey, envKey string) string {
		if v, ok := opts[optKey].(string); ok && v != "" {
			return v
		}
		return os.Getenv(envKey)
	}

	creds := azureCreds{
		subscriptionID: subscriptionID,
		tenantID:       get("azure_tenant_id", "AZURE_TENANT_ID"),
		clientID:       get("azure_client_id", "AZURE_CLIENT_ID"),
		clientSecret:   get("azure_client_secret", "AZURE_CLIENT_SECRET"),
	}
	if creds.tenantID == "" || creds.clientID == "" || creds.clientSecret == "" {
		return azureCreds{}, fmt.Errorf("Azure credentials not configured: set azure_tenant_id, azure_client_id, azure_client_secret in scan options or AZURE_TENANT_ID/AZURE_CLIENT_ID/AZURE_CLIENT_SECRET environment variables")
	}
	return creds, nil
}

func azureGetAccessToken(ctx context.Context, creds azureCreds, client *http.Client) (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", url.PathEscape(creds.tenantID))

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", creds.clientID)
	form.Set("client_secret", creds.clientSecret)
	form.Set("scope", "https://management.azure.com/.default")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build Azure token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Azure token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Azure token endpoint error %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse Azure token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("Azure returned empty access_token")
	}
	return tokenResp.AccessToken, nil
}

func (s *CloudScanner) scanAzure(ctx context.Context, subscriptionID string, opts map[string]any) ([]*model.DiscoveredAsset, error) {
	creds, err := loadAzureCreds(subscriptionID, opts)
	if err != nil {
		return nil, err
	}

	s.logger.Info().Str("subscription", creds.subscriptionID).Msg("starting Azure VM discovery")

	token, err := azureGetAccessToken(ctx, creds, s.client)
	if err != nil {
		return nil, fmt.Errorf("obtain Azure access token: %w", err)
	}

	// Phase 1: list all VMs.
	vms, err := s.azureListVMs(ctx, creds.subscriptionID, token)
	if err != nil {
		return nil, err
	}

	// Phase 2: resolve NIC IDs to IP addresses (batch via Resource Graph if available,
	// otherwise sequential NIC lookups).
	assets := make([]*model.DiscoveredAsset, 0, len(vms))
	for i := range vms {
		ips, err := s.azureGetVMIPs(ctx, token, vms[i].NICs)
		if err != nil {
			s.logger.Warn().Err(err).Str("vm", vms[i].Name).Msg("failed to resolve VM IP addresses")
		}
		assets = append(assets, azureVMToAsset(vms[i], ips))
	}
	return assets, nil
}

type azureVM struct {
	ID            string
	Name          string
	Location      string
	VMSize        string
	OSType        string
	NICs          []string // NIC resource IDs
	Tags          map[string]string
	ResourceGroup string
}

func (s *CloudScanner) azureListVMs(ctx context.Context, subscriptionID, token string) ([]azureVM, error) {
	const apiVersion = "2023-03-01"
	listURL := fmt.Sprintf("https://management.azure.com/subscriptions/%s/providers/Microsoft.Compute/virtualMachines?api-version=%s",
		url.PathEscape(subscriptionID), apiVersion)

	var vms []azureVM
	nextLink := listURL

	for nextLink != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextLink, nil)
		if err != nil {
			return nil, fmt.Errorf("build Azure VM list request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Azure VM list: %w", err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Azure VM list error %d: %s", resp.StatusCode, truncate(string(body), 300))
		}

		var page struct {
			Value    []azureVMJSON `json:"value"`
			NextLink string        `json:"nextLink"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse Azure VM list: %w", err)
		}

		for _, v := range page.Value {
			vm := azureVM{
				ID:            v.ID,
				Name:          v.Name,
				Location:      v.Location,
				VMSize:        v.Properties.HardwareProfile.VMSize,
				OSType:        v.Properties.StorageProfile.OSDisk.OSType,
				Tags:          v.Tags,
				ResourceGroup: extractResourceGroup(v.ID),
			}
			for _, nic := range v.Properties.NetworkProfile.NetworkInterfaces {
				vm.NICs = append(vm.NICs, nic.ID)
			}
			vms = append(vms, vm)
		}
		nextLink = page.NextLink
	}
	return vms, nil
}

type azureVMJSON struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags"`
	Properties struct {
		HardwareProfile struct {
			VMSize string `json:"vmSize"`
		} `json:"hardwareProfile"`
		StorageProfile struct {
			OSDisk struct {
				OSType string `json:"osType"`
			} `json:"osDisk"`
		} `json:"storageProfile"`
		NetworkProfile struct {
			NetworkInterfaces []struct {
				ID string `json:"id"`
			} `json:"networkInterfaces"`
		} `json:"networkProfile"`
	} `json:"properties"`
}

// azureGetVMIPs fetches private + public IP addresses for a list of NIC resource IDs.
func (s *CloudScanner) azureGetVMIPs(ctx context.Context, token string, nicIDs []string) ([]string, error) {
	const apiVersion = "2023-05-01"
	var ips []string

	for _, nicID := range nicIDs {
		nicURL := fmt.Sprintf("https://management.azure.com%s?api-version=%s", nicID, apiVersion)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nicURL, nil)
		if err != nil {
			return ips, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			return ips, err
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue // Non-fatal: skip NIC
		}

		var nic struct {
			Properties struct {
				IPConfigurations []struct {
					Properties struct {
						PrivateIPAddress string `json:"privateIPAddress"`
						PublicIPAddress  *struct {
							Properties struct {
								IPAddress string `json:"ipAddress"`
							} `json:"properties"`
						} `json:"publicIPAddress"`
					} `json:"properties"`
				} `json:"ipConfigurations"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(body, &nic); err != nil {
			continue
		}
		for _, ipc := range nic.Properties.IPConfigurations {
			if ip := ipc.Properties.PrivateIPAddress; ip != "" {
				ips = append(ips, ip)
			}
			if ipc.Properties.PublicIPAddress != nil {
				if ip := ipc.Properties.PublicIPAddress.Properties.IPAddress; ip != "" {
					ips = append(ips, ip)
				}
			}
		}
	}
	return ips, nil
}

func azureVMToAsset(vm azureVM, ips []string) *model.DiscoveredAsset {
	ip := ""
	if len(ips) > 0 {
		ip = ips[0]
	}

	hostname := vm.Name
	var hostnamePtr *string
	if hostname != "" {
		hostnamePtr = &hostname
	}

	var osVal *string
	if vm.OSType != "" {
		osVal = &vm.OSType
	}

	extra := map[string]any{
		"cloud_provider":       "azure",
		"azure_vm_id":          vm.ID,
		"azure_location":       vm.Location,
		"azure_vm_size":        vm.VMSize,
		"azure_resource_group": vm.ResourceGroup,
	}
	if len(ips) > 1 {
		extra["azure_additional_ips"] = ips[1:]
	}
	for k, v := range vm.Tags {
		extra["azure_tag_"+strings.ToLower(k)] = v
	}

	return &model.DiscoveredAsset{
		IPAddress:       ip,
		Hostname:        hostnamePtr,
		OS:              osVal,
		AssetType:       model.AssetTypeCloudResource,
		OpenPorts:       []int{},
		Banners:         map[int]string{},
		ExtraMetadata:   extra,
		DiscoverySource: "cloud_scan",
	}
}

// extractResourceGroup parses the resource group name from an Azure resource ID.
// Example: /subscriptions/xxx/resourceGroups/myRG/providers/...
func extractResourceGroup(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, p := range parts {
		if strings.EqualFold(p, "resourceGroups") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// ────────────────────────────────────────────────────────────────────────────
// Utilities
// ────────────────────────────────────────────────────────────────────────────

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
