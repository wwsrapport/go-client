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
	"time"
)

const (
	defaultBaseURL = "https://wwsrapport.nl/v1"
	clientHeader  = "wwsrapport-go-client/0.1.0"
)

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
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
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("wwsrapport: API key is required")
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
	request.Header.Set("Authorization", "Bearer "+c.APIKey)
	request.Header.Set("X-WWSrapport-Client", clientHeader)
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

func (c *Client) buildURL(path string, query url.Values) string {
	base := strings.TrimRight(c.BaseURL, "/")
	full := base + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		full += "?" + query.Encode()
	}
	return full
}
