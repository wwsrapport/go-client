package wwsrapport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL = "https://wwsrapport.nl/v1"
	clientHeader   = "wwsrapport-go-client/0.3.0"
)

type OAuthClientCredentials struct {
	ClientID, ClientSecret, TokenURL string
	Scopes                           []string
}

type RequestContext struct {
	MunicipalityCode, PurposeCode, CaseReference, ClientReference string
}

type Client struct {
	APIKey         string
	OAuth          *OAuthClientCredentials
	RequestContext *RequestContext
	BaseURL        string
	HTTPClient     *http.Client
	tokenMu        sync.Mutex
	accessToken    string
	tokenExpires   time.Time
}

func NewOAuthClient(oauth OAuthClientCredentials, requestContext *RequestContext, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{OAuth: &oauth, RequestContext: requestContext, BaseURL: defaultBaseURL, HTTPClient: httpClient}
}

type Error struct {
	StatusCode int
	RequestID  string
	Body       string
}

func (e *Error) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("wwsrapport: HTTP %d: %s", e.StatusCode, e.Body)
	}
	return fmt.Sprintf("wwsrapport: HTTP %d", e.StatusCode)
}

func NewClient(apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		APIKey:     apiKey,
		BaseURL:    defaultBaseURL,
		HTTPClient: httpClient,
	}
}

func (c *Client) PrefillProperty(ctx context.Context, address any) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/properties/prefill", map[string]any{"address": address}, "", nil)
}

func (c *Client) ValidateReport(ctx context.Context, input any) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/reports/validate", input, "", nil)
}

func (c *Client) CreateReport(ctx context.Context, input any, idempotencyKey string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/reports", input, idempotencyKey, nil)
}

func (c *Client) RecalculateReport(ctx context.Context, reportID string, input any, idempotencyKey string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/reports/"+url.PathEscape(reportID)+"/recalculate", input, idempotencyKey, nil)
}

func (c *Client) ListReports(ctx context.Context, query url.Values) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/reports", nil, "", query)
}

func (c *Client) GetReport(ctx context.Context, reportID string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/reports/"+url.PathEscape(reportID), nil, "", nil)
}

func (c *Client) GetCalculation(ctx context.Context, reportID string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/reports/"+url.PathEscape(reportID)+"/calculation", nil, "", nil)
}

func (c *Client) GetImprovementAdvice(ctx context.Context, reportID string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/reports/"+url.PathEscape(reportID)+"/improvement-advice", nil, "", nil)
}

func (c *Client) GetReportVerification(ctx context.Context, reportID string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/reports/"+url.PathEscape(reportID)+"/verification", nil, "", nil)
}

func (c *Client) ReviewReport(ctx context.Context, reportID string, review any, idempotencyKey string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/reports/"+url.PathEscape(reportID)+"/human-review", review, idempotencyKey, nil)
}

func (c *Client) CreateBatch(ctx context.Context, input any, idempotencyKey string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/batches", input, idempotencyKey, nil)
}

func (c *Client) GetBatch(ctx context.Context, id string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/batches/"+url.PathEscape(id), nil, "", nil)
}

func (c *Client) RetryBatch(ctx context.Context, id, idempotencyKey string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/batches/"+url.PathEscape(id)+"/retry", nil, idempotencyKey, nil)
}

func (c *Client) RequestTenantExport(ctx context.Context, idempotencyKey string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/exports", nil, idempotencyKey, nil)
}

func (c *Client) GetTenantExport(ctx context.Context, id string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/exports/"+url.PathEscape(id), nil, "", nil)
}

func (c *Client) CreateTenantExportDownloadURL(ctx context.Context, id string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/exports/"+url.PathEscape(id)+"/download-url", nil, "", nil)
}

func (c *Client) RequestOffboarding(ctx context.Context, requestedByReference, reason string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/offboarding", map[string]string{"confirmation": "REQUEST_OFFBOARDING", "requested_by_reference": requestedByReference, "reason": reason}, "", nil)
}

func (c *Client) DeriveBagReference(ctx context.Context, bagVboID string) (json.RawMessage, error) {
	if !validBagVboID(bagVboID) {
		return nil, fmt.Errorf("wwsrapport: BAG verblijfsobject ID must contain exactly sixteen digits")
	}
	return c.doJSON(ctx, http.MethodPost, "/registry/bag-reference", map[string]string{"bagVboId": bagVboID}, "", nil)
}

func (c *Client) SearchRegistryByBag(ctx context.Context, bagVboID string) (json.RawMessage, error) {
	if !validBagVboID(bagVboID) {
		return nil, fmt.Errorf("wwsrapport: BAG verblijfsobject ID must contain exactly sixteen digits")
	}
	return c.doJSON(ctx, http.MethodPost, "/registry/search-by-bag", map[string]string{"bagVboId": bagVboID}, "", nil)
}

func validBagVboID(value string) bool {
	if len(value) != 16 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func (c *Client) ListDocuments(ctx context.Context, reportID string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/reports/"+url.PathEscape(reportID)+"/documents", nil, "", nil)
}

func (c *Client) DownloadWwsReport(ctx context.Context, reportID string) ([]byte, error) {
	return c.doBytes(ctx, http.MethodGet, "/reports/"+url.PathEscape(reportID)+"/documents/wws-report", nil, "", nil, "application/pdf, application/octet-stream")
}

func (c *Client) DownloadImprovementAdvice(ctx context.Context, reportID string) ([]byte, error) {
	return c.doBytes(ctx, http.MethodGet, "/reports/"+url.PathEscape(reportID)+"/documents/improvement-advice", nil, "", nil, "application/pdf, application/octet-stream")
}

func (c *Client) CurrentUsage(ctx context.Context) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/usage/current", nil, "", nil)
}

func (c *Client) UsageHistory(ctx context.Context, query url.Values) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/usage/history", nil, "", query)
}

func (c *Client) ListRulesets(ctx context.Context) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/rulesets", nil, "", nil)
}

func (c *Client) ListWebhooks(ctx context.Context) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/webhooks", nil, "", nil)
}

func (c *Client) CreateWebhook(ctx context.Context, input any) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/webhooks", input, "", nil)
}

func (c *Client) GetWebhook(ctx context.Context, webhookID string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/webhooks/"+url.PathEscape(webhookID), nil, "", nil)
}

func (c *Client) UpdateWebhook(ctx context.Context, webhookID string, input any) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPatch, "/webhooks/"+url.PathEscape(webhookID), input, "", nil)
}

func (c *Client) DeleteWebhook(ctx context.Context, webhookID string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodDelete, "/webhooks/"+url.PathEscape(webhookID), nil, "", nil)
}

func (c *Client) SendTestWebhook(ctx context.Context, webhookID string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/webhooks/"+url.PathEscape(webhookID)+"/test", nil, "", nil)
}

func (c *Client) ListWebhookDeliveries(ctx context.Context, webhookID string, query url.Values) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, "/webhooks/"+url.PathEscape(webhookID)+"/deliveries", nil, "", query)
}

func (c *Client) RetryWebhookDelivery(ctx context.Context, webhookID string, deliveryID string) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/webhooks/"+url.PathEscape(webhookID)+"/deliveries/"+url.PathEscape(deliveryID)+"/retry", nil, "", nil)
}

func (c *Client) doJSON(ctx context.Context, method string, path string, body any, idempotencyKey string, query url.Values) (json.RawMessage, error) {
	result, err := c.doBytes(ctx, method, path, body, idempotencyKey, query, "application/json")
	if err != nil {
		return nil, err
	}
	return json.RawMessage(result), nil
}

func (c *Client) doBytes(ctx context.Context, method string, path string, body any, idempotencyKey string, query url.Values, accept string) ([]byte, error) {
	token, err := c.bearerToken(ctx)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.buildURL(path, query), reader)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", accept)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-WWSrapport-Client", clientHeader)
	c.applyRequestContext(request.Header)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &Error{
			StatusCode: response.StatusCode,
			RequestID:  response.Header.Get("X-Request-Id"),
			Body:       string(responseBody),
		}
	}

	return responseBody, nil
}

func (c *Client) bearerToken(ctx context.Context) (string, error) {
	if strings.TrimSpace(c.APIKey) != "" {
		return c.APIKey, nil
	}
	if c.OAuth == nil || strings.TrimSpace(c.OAuth.ClientID) == "" || strings.TrimSpace(c.OAuth.ClientSecret) == "" {
		return "", fmt.Errorf("wwsrapport: API key or OAuth client credentials are required")
	}
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.accessToken != "" && time.Now().Add(30*time.Second).Before(c.tokenExpires) {
		return c.accessToken, nil
	}
	tokenURL := c.OAuth.TokenURL
	if tokenURL == "" {
		base, err := url.Parse(c.BaseURL)
		if err != nil {
			return "", err
		}
		base.Path, base.RawQuery = "/oauth/token", ""
		tokenURL = base.String()
	}
	form := url.Values{"grant_type": {"client_credentials"}}
	if len(c.OAuth.Scopes) > 0 {
		form.Set("scope", strings.Join(c.OAuth.Scopes, " "))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.OAuth.ClientID, c.OAuth.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	response, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", &Error{StatusCode: response.StatusCode, RequestID: response.Header.Get("X-Request-Id"), Body: string(body)}
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("wwsrapport: OAuth response has no access_token")
	}
	if payload.ExpiresIn <= 0 {
		payload.ExpiresIn = 300
	}
	c.accessToken, c.tokenExpires = payload.AccessToken, time.Now().Add(time.Duration(payload.ExpiresIn)*time.Second)
	return c.accessToken, nil
}

func (c *Client) applyRequestContext(headers http.Header) {
	if c.RequestContext == nil {
		return
	}
	values := map[string]string{"X-WWS-Municipality-Code": c.RequestContext.MunicipalityCode, "X-WWS-Purpose-Code": c.RequestContext.PurposeCode, "X-WWS-Case-Reference": c.RequestContext.CaseReference, "X-WWS-Client-Reference": c.RequestContext.ClientReference}
	for key, value := range values {
		if strings.TrimSpace(value) != "" {
			headers.Set(key, value)
		}
	}
}

func (c *Client) buildURL(path string, query url.Values) string {
	base := strings.TrimRight(c.BaseURL, "/")
	full := base + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		full += "?" + query.Encode()
	}
	return full
}
