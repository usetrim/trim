package paddleapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Money is a Paddle money object (amount in lowest currency denomination as a string).
type Money struct {
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currency_code"`
}

// Duration is a Paddle billing_cycle / trial_period interval.
type Duration struct {
	Interval  string `json:"interval"`
	Frequency int    `json:"frequency"`
}

// CreateProductRequest is POST /products.
// Docs: https://developer.paddle.com/api-reference/products/create-product
type CreateProductRequest struct {
	Name        string                 `json:"name"`
	TaxCategory string                 `json:"tax_category"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
}

// UpdateProductRequest is PATCH /products/{id}.
type UpdateProductRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
	Status      *string                `json:"status,omitempty"`
}

// ProductResponse wraps a product entity.
type ProductResponse struct {
	Data struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
		TaxCategory string `json:"tax_category"`
	} `json:"data"`
}

// CreatePriceRequest is POST /prices.
// Docs: https://developer.paddle.com/api-reference/prices/create-price
type CreatePriceRequest struct {
	Description  string                 `json:"description"`
	Name         string                 `json:"name,omitempty"`
	ProductID    string                 `json:"product_id"`
	UnitPrice    Money                  `json:"unit_price"`
	BillingCycle *Duration              `json:"billing_cycle,omitempty"`
	TaxMode      string                 `json:"tax_mode,omitempty"`
	Type         string                 `json:"type,omitempty"`
	CustomData   map[string]interface{} `json:"custom_data,omitempty"`
}

// UpdatePriceRequest is PATCH /prices/{id} (archive / metadata).
// Unit price changes for catalog sync create a new price instead of mutating charge amount in place.
type UpdatePriceRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Status      *string                `json:"status,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
}

// PriceResponse wraps a price entity.
type PriceResponse struct {
	Data struct {
		ID           string    `json:"id"`
		ProductID    string    `json:"product_id"`
		Status       string    `json:"status"`
		Description  string    `json:"description"`
		Name         string    `json:"name"`
		UnitPrice    Money     `json:"unit_price"`
		BillingCycle *Duration `json:"billing_cycle"`
	} `json:"data"`
}

// CreateDiscountRequest is POST /discounts.
// Docs: https://developer.paddle.com/api-reference/discounts/create-discount
type CreateDiscountRequest struct {
	Description        string  `json:"description"`
	Type               string  `json:"type"`
	Amount             string  `json:"amount"`
	EnabledForCheckout bool    `json:"enabled_for_checkout"`
	Code               *string `json:"code,omitempty"`
	CurrencyCode       *string `json:"currency_code,omitempty"`
	Recur              bool    `json:"recur,omitempty"`
}

// UpdateDiscountRequest is PATCH /discounts/{id}.
type UpdateDiscountRequest struct {
	Description        *string `json:"description,omitempty"`
	Amount             *string `json:"amount,omitempty"`
	EnabledForCheckout *bool   `json:"enabled_for_checkout,omitempty"`
	Status             *string `json:"status,omitempty"`
}

// DiscountResponse wraps a discount entity.
type DiscountResponse struct {
	Data struct {
		ID                 string  `json:"id"`
		Status             string  `json:"status"`
		Description        string  `json:"description"`
		Type               string  `json:"type"`
		Amount             string  `json:"amount"`
		Code               *string `json:"code"`
		EnabledForCheckout bool    `json:"enabled_for_checkout"`
	} `json:"data"`
}

func (c *Client) CreateProduct(ctx context.Context, body CreateProductRequest) (*ProductResponse, error) {
	body.Name = strings.TrimSpace(body.Name)
	body.TaxCategory = strings.TrimSpace(body.TaxCategory)
	if body.Name == "" || body.TaxCategory == "" {
		return nil, fmt.Errorf("paddle create product: name and tax_category required")
	}
	var out ProductResponse
	if err := c.doJSON(ctx, http.MethodPost, "/products", body, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Data.ID) == "" {
		return nil, fmt.Errorf("paddle create product: empty id")
	}
	return &out, nil
}

func (c *Client) UpdateProduct(ctx context.Context, productID string, body UpdateProductRequest) (*ProductResponse, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("paddle update product: id required")
	}
	var out ProductResponse
	if err := c.doJSON(ctx, http.MethodPatch, "/products/"+productID, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreatePrice(ctx context.Context, body CreatePriceRequest) (*PriceResponse, error) {
	body.ProductID = strings.TrimSpace(body.ProductID)
	body.Description = strings.TrimSpace(body.Description)
	body.UnitPrice.Amount = strings.TrimSpace(body.UnitPrice.Amount)
	body.UnitPrice.CurrencyCode = strings.TrimSpace(strings.ToUpper(body.UnitPrice.CurrencyCode))
	if body.ProductID == "" || body.Description == "" || body.UnitPrice.Amount == "" || body.UnitPrice.CurrencyCode == "" {
		return nil, fmt.Errorf("paddle create price: product_id, description, and unit_price required")
	}
	var out PriceResponse
	if err := c.doJSON(ctx, http.MethodPost, "/prices", body, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Data.ID) == "" {
		return nil, fmt.Errorf("paddle create price: empty id")
	}
	return &out, nil
}

func (c *Client) GetPrice(ctx context.Context, priceID string) (*PriceResponse, error) {
	priceID = strings.TrimSpace(priceID)
	if priceID == "" {
		return nil, fmt.Errorf("paddle get price: id required")
	}
	var out PriceResponse
	if err := c.doJSON(ctx, http.MethodGet, "/prices/"+priceID, nil, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Data.ID) == "" {
		return nil, fmt.Errorf("paddle get price: empty id")
	}
	return &out, nil
}

func (c *Client) ArchivePrice(ctx context.Context, priceID string) error {
	priceID = strings.TrimSpace(priceID)
	if priceID == "" {
		return fmt.Errorf("paddle archive price: id required")
	}
	status := "archived"
	var out PriceResponse
	return c.doJSON(ctx, http.MethodPatch, "/prices/"+priceID, UpdatePriceRequest{Status: &status}, &out)
}

func (c *Client) CreateDiscount(ctx context.Context, body CreateDiscountRequest) (*DiscountResponse, error) {
	body.Description = strings.TrimSpace(body.Description)
	body.Type = strings.TrimSpace(body.Type)
	body.Amount = strings.TrimSpace(body.Amount)
	if body.Description == "" || body.Type == "" || body.Amount == "" {
		return nil, fmt.Errorf("paddle create discount: description, type, and amount required")
	}
	var out DiscountResponse
	if err := c.doJSON(ctx, http.MethodPost, "/discounts", body, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Data.ID) == "" {
		return nil, fmt.Errorf("paddle create discount: empty id")
	}
	return &out, nil
}

func (c *Client) GetDiscount(ctx context.Context, discountID string) (*DiscountResponse, error) {
	discountID = strings.TrimSpace(discountID)
	if discountID == "" {
		return nil, fmt.Errorf("paddle get discount: id required")
	}
	var out DiscountResponse
	if err := c.doJSON(ctx, http.MethodGet, "/discounts/"+discountID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateDiscount(ctx context.Context, discountID string, body UpdateDiscountRequest) (*DiscountResponse, error) {
	discountID = strings.TrimSpace(discountID)
	if discountID == "" {
		return nil, fmt.Errorf("paddle update discount: id required")
	}
	var out DiscountResponse
	if err := c.doJSON(ctx, http.MethodPatch, "/discounts/"+discountID, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
