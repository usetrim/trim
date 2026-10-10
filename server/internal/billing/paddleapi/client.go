package paddleapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to Paddle Billing API v1 for subscription updates.
// Docs: https://developer.paddle.com/api-reference/subscriptions/update-subscription
type Client struct {
	APIKey       string
	BaseURL      string
	HTTP         *http.Client
	ListMaxPages int
	ListPerPage  int
}

func New(apiKey, env string, timeoutSec, listMaxPages, listPerPage int) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	env = strings.TrimSpace(strings.ToLower(env))
	if apiKey == "" {
		return nil, fmt.Errorf("PADDLE_API_KEY is required")
	}
	if timeoutSec < 1 || timeoutSec > 300 {
		return nil, fmt.Errorf("PADDLE_HTTP_TIMEOUT_SEC must be 1-300 (got %d)", timeoutSec)
	}
	if listMaxPages < 1 || listMaxPages > 1000 {
		return nil, fmt.Errorf("PADDLE_LIST_MAX_PAGES must be 1-1000 (got %d)", listMaxPages)
	}
	if listPerPage < 1 || listPerPage > 200 {
		return nil, fmt.Errorf("PADDLE_LIST_PER_PAGE must be 1-200 (got %d)", listPerPage)
	}
	base := "https://api.paddle.com"
	if env == "sandbox" {
		base = "https://sandbox-api.paddle.com"
	} else if env != "production" {
		return nil, fmt.Errorf("PADDLE_ENV must be sandbox or production")
	}
	return &Client{
		APIKey:       apiKey,
		BaseURL:      base,
		HTTP:         &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		ListMaxPages: listMaxPages,
		ListPerPage:  listPerPage,
	}, nil
}

// CatalogWriteReady reports whether the API key looks usable for product/price writes.
// Rejects empty and common placeholder values so admin sync fails closed with a clear error
// instead of opaque 401s from Paddle.
//
// Do not substring-match short tokens like "xxx" inside the secret: real Paddle keys are
// random and can contain those letters, which would false-reject a valid key.
func (c *Client) CatalogWriteReady() error {
	if c == nil {
		return fmt.Errorf("paddle client nil")
	}
	key := strings.TrimSpace(c.APIKey)
	if key == "" {
		return fmt.Errorf("paddle api key empty")
	}
	lower := strings.ToLower(key)
	switch {
	case strings.HasPrefix(lower, "paste"),
		lower == "changeme",
		lower == "your_api_key",
		lower == "your-api-key",
		lower == "pdl_xxx",
		lower == "pdl_live_xxx",
		lower == "pdl_sdbx_xxx",
		strings.HasPrefix(lower, "pdl_your_"),
		strings.HasPrefix(lower, "pdl_changeme"),
		strings.Contains(lower, "your_api_key"),
		strings.Contains(lower, "your-api-key"),
		strings.HasSuffix(lower, "_xxx"),
		strings.HasSuffix(lower, "..."),
		strings.Contains(lower, "apikey_..."):
		return fmt.Errorf("paddle api key looks like a placeholder; set a real sandbox/production key")
	}
	// Paddle Billing API keys are issued as pdl_… (sandbox or live).
	if !strings.HasPrefix(lower, "pdl_") {
		return fmt.Errorf("paddle api key must start with pdl_")
	}
	// Client-side tokens must not be used as the server API key.
	if strings.Contains(lower, "client") && !strings.Contains(lower, "apikey") {
		return fmt.Errorf("paddle api key looks like a client-side token; use a server API key (pdl_…apikey_…)")
	}
	if !strings.Contains(lower, "apikey") {
		return fmt.Errorf("paddle api key must be a server API key containing apikey_ (not a client-side token)")
	}
	return nil
}

type Item struct {
	PriceID  string `json:"price_id"`
	Quantity int    `json:"quantity"`
}

type UpdateSubscriptionRequest struct {
	Items                []Item                 `json:"items"`
	ProrationBillingMode string                 `json:"proration_billing_mode"`
	OnPaymentFailure     string                 `json:"on_payment_failure,omitempty"`
	CustomData           map[string]interface{} `json:"custom_data,omitempty"`
}

type SubscriptionResponse struct {
	Data struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Items  []struct {
			Price struct {
				ID string `json:"id"`
			} `json:"price"`
			Quantity int `json:"quantity"`
		} `json:"items"`
		CurrentBillingPeriod struct {
			StartsAt string `json:"starts_at"`
			EndsAt   string `json:"ends_at"`
		} `json:"current_billing_period"`
	} `json:"data"`
	Error *struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"error"`
}

type PreviewUpdateResponse struct {
	Data struct {
		ID                   string `json:"id"`
		Status               string `json:"status"`
		ImmediateTransaction *struct {
			Details struct {
				Totals struct {
					Subtotal     string `json:"subtotal"`
					Tax          string `json:"tax"`
					Total        string `json:"total"`
					Credit       string `json:"credit"`
					Balance      string `json:"balance"`
					GrandTotal   string `json:"grand_total"`
					CurrencyCode string `json:"currency_code"`
				} `json:"totals"`
			} `json:"details"`
		} `json:"immediate_transaction"`
		CurrentBillingPeriod struct {
			StartsAt string `json:"starts_at"`
			EndsAt   string `json:"ends_at"`
		} `json:"current_billing_period"`
	} `json:"data"`
	Error *struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"error"`
}

func (c *Client) PreviewUpdate(ctx context.Context, subscriptionID string, body UpdateSubscriptionRequest) (*PreviewUpdateResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.BaseURL+"/subscriptions/"+subscriptionID+"/preview", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Paddle-Version", "1")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	var out PreviewUpdateResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("paddle preview decode: %w body=%s", err, string(raw))
	}
	if res.StatusCode >= 400 {
		detail := string(raw)
		if out.Error != nil {
			detail = out.Error.Detail
			if detail == "" {
				detail = out.Error.Code
			}
		}
		return nil, fmt.Errorf("paddle preview api %s: %s", res.Status, detail)
	}
	return &out, nil
}

func (c *Client) UpdateSubscription(ctx context.Context, subscriptionID string, body UpdateSubscriptionRequest) (*SubscriptionResponse, error) {
	return c.patch(ctx, "/subscriptions/"+subscriptionID, body)
}

type InvoicePDFResponse struct {
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
	Error *struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"error"`
}

// GetTransactionInvoicePDF returns a short-lived Paddle invoice PDF URL for a transaction.
// Docs: GET /transactions/{transaction_id}/invoice
func (c *Client) GetTransactionInvoicePDF(ctx context.Context, transactionID string) (string, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return "", fmt.Errorf("transaction id is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/transactions/"+transactionID+"/invoice?disposition=inline", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Paddle-Version", "1")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	var out InvoicePDFResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("paddle invoice decode: %w body=%s", err, string(raw))
	}
	if res.StatusCode >= 400 {
		detail := string(raw)
		if out.Error != nil {
			detail = out.Error.Detail
			if detail == "" {
				detail = out.Error.Code
			}
		}
		return "", fmt.Errorf("paddle invoice api %s: %s", res.Status, detail)
	}
	if out.Data.URL == "" {
		return "", fmt.Errorf("paddle invoice url empty")
	}
	return out.Data.URL, nil
}

type getTransactionResponse struct {
	Data  TransactionJSON `json:"data"`
	Error *struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"error"`
}

// GetTransaction fetches one Paddle Billing transaction by id (include customer + address).
// Docs: GET /transactions/{transaction_id}
func (c *Client) GetTransaction(ctx context.Context, transactionID string) (*TransactionJSON, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, fmt.Errorf("transaction id is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/transactions/"+transactionID+"?include=customer,address", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Paddle-Version", "1")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	var out getTransactionResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("paddle get transaction decode: %w body=%s", err, string(raw))
	}
	if res.StatusCode >= 400 {
		detail := string(raw)
		if out.Error != nil {
			detail = out.Error.Detail
			if detail == "" {
				detail = out.Error.Code
			}
		}
		return nil, fmt.Errorf("paddle get transaction api %s: %s", res.Status, detail)
	}
	if strings.TrimSpace(out.Data.ID) == "" {
		return nil, fmt.Errorf("paddle transaction empty")
	}
	return &out.Data, nil
}

// TransactionJSON is a Paddle Billing transaction entity (subset used by Trim receipts).
type TransactionJSON struct {
	ID             string  `json:"id"`
	CustomerID     string  `json:"customer_id"`
	Status         string  `json:"status"`
	CurrencyCode   string  `json:"currency_code"`
	InvoiceNumber  *string `json:"invoice_number"`
	InvoiceID      *string `json:"invoice_id"`
	SubscriptionID *string `json:"subscription_id"`
	CustomData     struct {
		UserID string `json:"userId"`
		PlanID string `json:"plan_id"`
	} `json:"custom_data"`
	Details struct {
		Totals struct {
			Subtotal string `json:"subtotal"`
			Tax      string `json:"tax"`
			Total    string `json:"total"`
		} `json:"totals"`
		TaxRatesUsed []struct {
			TaxRate string `json:"tax_rate"`
		} `json:"tax_rates_used"`
		LineItems []struct {
			PriceID  string `json:"price_id"`
			Quantity int    `json:"quantity"`
			TaxRate  string `json:"tax_rate"`
			Totals   struct {
				Subtotal string `json:"subtotal"`
				Tax      string `json:"tax"`
				Total    string `json:"total"`
			} `json:"totals"`
			UnitTotals struct {
				Subtotal string `json:"subtotal"`
				Tax      string `json:"tax"`
				Total    string `json:"total"`
			} `json:"unit_totals"`
			Product struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				SKU  string `json:"sku"`
			} `json:"product"`
		} `json:"line_items"`
	} `json:"details"`
	BillingPeriod *struct {
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	} `json:"billing_period"`
	Address *struct {
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		FirstLine   string `json:"first_line"`
		SecondLine  string `json:"second_line"`
		City        string `json:"city"`
		Region      string `json:"region"`
		PostalCode  string `json:"postal_code"`
		CountryCode string `json:"country_code"`
	} `json:"address"`
	Customer *struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"customer"`
	BillingDetails *struct {
		TaxIdentifier string `json:"tax_identifier"`
	} `json:"billing_details"`
	// Items are catalog selections; priced lines are details.line_items.
	Items []struct {
		Price struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Product     struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				SKU  string `json:"sku"`
			} `json:"product"`
		} `json:"price"`
		Quantity int `json:"quantity"`
	} `json:"items"`
}

type listTransactionsResponse struct {
	Data  []TransactionJSON `json:"data"`
	Error *struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"error"`
	Meta struct {
		Pagination struct {
			HasMore        bool   `json:"has_more"`
			Next           string `json:"next"`
			EstimatedTotal int    `json:"estimated_total"`
		} `json:"pagination"`
	} `json:"meta"`
}

// ListTransactions returns completed (or filtered) transactions for a Paddle customer.
// Paginates with after= until has_more is false (Paddle Billing list API).
// Docs: GET /transactions?customer_id=&status=
func (c *Client) ListTransactions(ctx context.Context, customerID, status string) ([]TransactionJSON, error) {
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return nil, fmt.Errorf("customer id is required")
	}
	var all []TransactionJSON
	after := ""
	maxPages := c.ListMaxPages
	perPage := c.ListPerPage
	for page := 0; page < maxPages; page++ {
		q := fmt.Sprintf("/transactions?customer_id=%s&per_page=%d&include=customer,address", customerID, perPage)
		if status != "" {
			q += "&status=" + strings.TrimSpace(status)
		}
		if after != "" {
			q += "&after=" + after
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+q, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Paddle-Version", "1")

		res, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		raw, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()

		var out listTransactionsResponse
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("paddle list transactions decode: %w body=%s", err, string(raw))
		}
		if res.StatusCode >= 400 {
			detail := string(raw)
			if out.Error != nil {
				detail = out.Error.Detail
				if detail == "" {
					detail = out.Error.Code
				}
			}
			return nil, fmt.Errorf("paddle list transactions api %s: %s", res.Status, detail)
		}
		all = append(all, out.Data...)
		if !out.Meta.Pagination.HasMore || strings.TrimSpace(out.Meta.Pagination.Next) == "" {
			break
		}
		after = strings.TrimSpace(out.Meta.Pagination.Next)
	}
	return all, nil
}

type PortalSessionRequest struct {
	SubscriptionIDs []string `json:"subscription_ids,omitempty"`
}

type PortalSessionResponse struct {
	Data struct {
		ID         string `json:"id"`
		CustomerID string `json:"customer_id"`
		URLs       struct {
			General struct {
				Overview string `json:"overview"`
			} `json:"general"`
			Subscriptions []struct {
				SubscriptionID                  string `json:"subscription_id"`
				CancelSubscription              string `json:"cancel_subscription"`
				UpdateSubscriptionPaymentMethod string `json:"update_subscription_payment_method"`
			} `json:"subscriptions"`
		} `json:"urls"`
	} `json:"data"`
	Error *struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"error"`
}

// CreatePortalSession creates authenticated Paddle customer portal links.
// Docs: POST /customers/{customer_id}/portal-sessions
func (c *Client) CreatePortalSession(ctx context.Context, customerID string, subscriptionIDs []string) (*PortalSessionResponse, error) {
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return nil, fmt.Errorf("paddle customer id is required")
	}
	body := PortalSessionRequest{SubscriptionIDs: subscriptionIDs}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/customers/"+customerID+"/portal-sessions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Paddle-Version", "1")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	var out PortalSessionResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("paddle portal decode: %w body=%s", err, string(raw))
	}
	if res.StatusCode >= 400 {
		detail := string(raw)
		if out.Error != nil {
			detail = out.Error.Detail
			if detail == "" {
				detail = out.Error.Code
			}
		}
		return nil, fmt.Errorf("paddle portal api %s: %s", res.Status, detail)
	}
	return &out, nil
}

func (c *Client) patch(ctx context.Context, path string, body UpdateSubscriptionRequest) (*SubscriptionResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.BaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Paddle-Version", "1")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	var out SubscriptionResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("paddle decode: %w body=%s", err, string(raw))
	}
	if res.StatusCode >= 400 {
		detail := string(raw)
		if out.Error != nil {
			detail = out.Error.Detail
			if detail == "" {
				detail = out.Error.Code
			}
		}
		return nil, fmt.Errorf("paddle api %s: %s", res.Status, detail)
	}
	return &out, nil
}
