package receipts

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

// LoadDetail loads a billing receipt + line items and applies receipt chrome.
// When ownerUserID is non-empty, the receipt must belong to that user (customer API).
// When ownerUserID is empty, any receipt may be loaded (admin API).
func LoadDetail(ctx context.Context, db *pgxpool.Pool, receiptID string, ownerUserID string, seller SellerLetterhead) (ReceiptDetail, error) {
	var d ReceiptDetail
	if db == nil {
		return d, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	receiptID = strings.TrimSpace(receiptID)
	if receiptID == "" {
		return d, fmt.Errorf("RECEIPT_ID_REQUIRED")
	}

	d.CompanyLegalName = seller.LegalName
	d.CompanySupportEmail = seller.SupportEmail
	d.CompanyLogoURL = strings.TrimSpace(seller.LogoURL)
	d.CompanyWordmark = companyWordmark(seller.LegalName)
	d.CompanyAddressLine1 = strings.TrimSpace(seller.AddressLine1)
	d.CompanyAddressLine2 = strings.TrimSpace(seller.AddressLine2)
	d.CompanyCity = strings.TrimSpace(seller.City)
	d.CompanyRegion = strings.TrimSpace(seller.Region)
	d.CompanyPostalCode = strings.TrimSpace(seller.PostalCode)
	d.CompanyCountry = strings.TrimSpace(seller.Country)
	d.CompanyVATID = strings.TrimSpace(seller.VATID)
	d.CompanyVATPrefix = subscriptions.MessageForCode("RECEIPT_VAT_ID_PREFIX")
	d.CompanyRegistration = strings.TrimSpace(seller.Registration)

	var err error
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID != "" {
		err = db.QueryRow(ctx, `
			select id::text, paddle_transaction_id, paddle_invoice_number, paddle_invoice_pdf_url,
			       status, currency_code, subtotal_cents, tax_cents, total_cents, tax_rate_bps,
			       bill_to_name, bill_to_email, bill_to_company,
			       bill_to_address_line1, bill_to_address_line2, bill_to_city, bill_to_region,
			       bill_to_postal_code, bill_to_country, tax_id, payment_method_summary,
			       period_start::text, period_end::text, paid_at::text, created_at::text
			from public.billing_receipts
			where id = $1 and user_id = $2
		`, receiptID, ownerUserID).Scan(
			&d.ID, &d.PaddleTransactionID, &d.PaddleInvoiceNumber, &d.PaddleInvoicePDFURL,
			&d.Status, &d.CurrencyCode, &d.SubtotalCents, &d.TaxCents, &d.TotalCents, &d.TaxRateBps,
			&d.BillToName, &d.BillToEmail, &d.BillToCompany,
			&d.BillToAddressLine1, &d.BillToAddressLine2, &d.BillToCity, &d.BillToRegion,
			&d.BillToPostalCode, &d.BillToCountry, &d.TaxID, &d.PaymentMethodSummary,
			&d.PeriodStart, &d.PeriodEnd, &d.PaidAt, &d.CreatedAt,
		)
	} else {
		err = db.QueryRow(ctx, `
			select id::text, paddle_transaction_id, paddle_invoice_number, paddle_invoice_pdf_url,
			       status, currency_code, subtotal_cents, tax_cents, total_cents, tax_rate_bps,
			       bill_to_name, bill_to_email, bill_to_company,
			       bill_to_address_line1, bill_to_address_line2, bill_to_city, bill_to_region,
			       bill_to_postal_code, bill_to_country, tax_id, payment_method_summary,
			       period_start::text, period_end::text, paid_at::text, created_at::text
			from public.billing_receipts
			where id = $1
		`, receiptID).Scan(
			&d.ID, &d.PaddleTransactionID, &d.PaddleInvoiceNumber, &d.PaddleInvoicePDFURL,
			&d.Status, &d.CurrencyCode, &d.SubtotalCents, &d.TaxCents, &d.TotalCents, &d.TaxRateBps,
			&d.BillToName, &d.BillToEmail, &d.BillToCompany,
			&d.BillToAddressLine1, &d.BillToAddressLine2, &d.BillToCity, &d.BillToRegion,
			&d.BillToPostalCode, &d.BillToCountry, &d.TaxID, &d.PaymentMethodSummary,
			&d.PeriodStart, &d.PeriodEnd, &d.PaidAt, &d.CreatedAt,
		)
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			return d, fmt.Errorf("RECEIPT_NOT_FOUND")
		}
		return d, fmt.Errorf("RECEIPT_OPERATION_FAILED")
	}

	rows, err := db.Query(ctx, `
		select position, description, quantity, unit_amount_cents, amount_cents, product_sku, price_id, price_name
		from public.billing_receipt_line_items
		where receipt_id = $1
		order by position asc
	`, receiptID)
	if err != nil {
		return d, fmt.Errorf("RECEIPT_OPERATION_FAILED")
	}
	defer rows.Close()

	d.LineItems = make([]LineItem, 0)
	for rows.Next() {
		var li LineItem
		if err := rows.Scan(&li.Position, &li.Description, &li.Quantity, &li.UnitAmountCents, &li.AmountCents, &li.ProductSKU, &li.PriceID, &li.PriceName); err != nil {
			return d, fmt.Errorf("RECEIPT_OPERATION_FAILED")
		}
		d.LineItems = append(d.LineItems, li)
	}

	d.StatusLabel = subscriptions.ReceiptStatusLabel(d.Status)
	if d.PaddleInvoiceNumber != nil && strings.TrimSpace(*d.PaddleInvoiceNumber) != "" {
		d.DisplayID = strings.TrimSpace(*d.PaddleInvoiceNumber)
	} else {
		d.DisplayID = d.PaddleTransactionID
	}
	d.MoneyLocale = strings.TrimSpace(subscriptions.MessageForCode("SITE_HTML_LANG"))
	if d.MoneyLocale == "" {
		return d, fmt.Errorf("RECEIPT_MONEY_LOCALE_REQUIRED")
	}
	d.PrintActionLabel = subscriptions.ActionLabelForCode("RECEIPT_PRINT")
	d.PrintPendingLabel = subscriptions.PendingLabelForCode("RECEIPT_PRINT")
	d.PrintPendingMs = subscriptions.MessageForCode("RECEIPT_PRINT_PENDING_MS")
	d.DownloadPdfLabel = subscriptions.MessageForCode("RECEIPT_DOWNLOAD_PDF")
	d.DownloadPdfPending = subscriptions.PendingLabelForCode("RECEIPT_DOWNLOAD_PDF")
	d.DownloadPdfFailed = subscriptions.MessageForCode("RECEIPT_DOWNLOAD_PDF_FAILED")
	d.DownloadPdfDone = subscriptions.MessageForCode("RECEIPT_PDF_DOWNLOAD_DONE")
	d.DownloadPdfFilenameFmt = subscriptions.MessageForCode("RECEIPT_PDF_FILENAME_FMT")
	d.SellerMissingMessage = subscriptions.MessageForCode("RECEIPT_SELLER_MISSING")
	d.NotFoundMessage = subscriptions.MessageForCode("RECEIPT_NOT_FOUND")
	applyReceiptChrome(&d)
	return d, nil
}
