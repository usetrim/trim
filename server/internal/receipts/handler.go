package receipts

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/billing/paddleapi"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/pagination"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

type Handler struct {
	DB     *pgxpool.Pool
	ReadDB *pgxpool.Pool
	Paddle *paddleapi.Client
}

func NewHandler(db, readDB *pgxpool.Pool, paddle *paddleapi.Client) *Handler {
	return &Handler{DB: db, ReadDB: readDB, Paddle: paddle}
}

func (h *Handler) readPool() *pgxpool.Pool {
	if h.ReadDB != nil {
		return h.ReadDB
	}
	return h.DB
}

func writeJSONErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": subscriptions.MessageForCode(code),
	})
}

type ReceiptSummary struct {
	ID                  string  `json:"id"`
	PaddleTransactionID string  `json:"paddle_transaction_id"`
	PaddleInvoiceNumber *string `json:"paddle_invoice_number"`
	PaddleInvoicePDFURL *string `json:"paddle_invoice_pdf_url"`
	// DisplayID is invoice number or full transaction id (no client truncation invent).
	DisplayID     string  `json:"display_id"`
	Status        string  `json:"status"`
	StatusLabel   string  `json:"status_label"`
	CurrencyCode  string  `json:"currency_code"`
	SubtotalCents int64   `json:"subtotal_cents"`
	TaxCents      int64   `json:"tax_cents"`
	TotalCents    int64   `json:"total_cents"`
	BillToName    *string `json:"bill_to_name"`
	BillToEmail   string  `json:"bill_to_email"`
	BillToCompany *string `json:"bill_to_company"`
	PeriodStart   *string `json:"period_start"`
	PeriodEnd     *string `json:"period_end"`
	PaidAt        *string `json:"paid_at"`
	CreatedAt     string  `json:"created_at"`
	// DateLabel is UTC calendar day for list rows (no client toLocaleDateString).
	DateLabel string `json:"date_label"`
}

type LineItem struct {
	Position        int     `json:"position"`
	Description     string  `json:"description"`
	Quantity        int     `json:"quantity"`
	UnitAmountCents int64   `json:"unit_amount_cents"`
	AmountCents     int64   `json:"amount_cents"`
	ProductSKU      *string `json:"product_sku"`
	PriceID         *string `json:"price_id"`
	// PriceName is the Paddle price name (e.g. "Pro (monthly)"), when present.
	PriceName *string `json:"price_name,omitempty"`
	// PeriodLabel is the billing period for this line (Paddle invoice product subtitle).
	PeriodLabel string `json:"period_label,omitempty"`
}

type ReceiptDetail struct {
	ReceiptSummary
	BillToAddressLine1 *string `json:"bill_to_address_line1"`
	BillToAddressLine2 *string `json:"bill_to_address_line2"`
	BillToCity         *string `json:"bill_to_city"`
	BillToRegion       *string `json:"bill_to_region"`
	BillToPostalCode   *string `json:"bill_to_postal_code"`
	BillToCountry      *string `json:"bill_to_country"`
	TaxID              *string `json:"tax_id"`
	TaxRateBps         int     `json:"tax_rate_bps"`
	// TaxRateLabel is e.g. " (8.25%)" from RECEIPT_TAX_RATE_FMT when tax_rate_bps > 0.
	TaxRateLabel string `json:"tax_rate_label,omitempty"`
	// TaxRatePercent is bare percent for tables / tax breakdown (e.g. "0%" / "8.25%").
	TaxRatePercent       string     `json:"tax_rate_percent,omitempty"`
	PaymentMethodSummary *string    `json:"payment_method_summary"`
	LineItems            []LineItem `json:"line_items"`
	CompanyLegalName     string     `json:"company_legal_name"`
	CompanySupportEmail  string     `json:"company_support_email"`
	CompanyLogoURL       string     `json:"company_logo_url,omitempty"`
	// CompanyLogoJPEG is set only for first-party PDF generation (never JSON).
	CompanyLogoJPEG     []byte `json:"-"`
	CompanyWordmark     string `json:"company_wordmark,omitempty"`
	CompanyAddressLine1 string `json:"company_address_line1,omitempty"`
	CompanyAddressLine2 string `json:"company_address_line2,omitempty"`
	CompanyCity         string `json:"company_city,omitempty"`
	CompanyRegion       string `json:"company_region,omitempty"`
	CompanyPostalCode   string `json:"company_postal_code,omitempty"`
	CompanyCountry      string `json:"company_country,omitempty"`
	CompanyVATID        string `json:"company_vat_id,omitempty"`
	// CompanyVATPrefix is RECEIPT_VAT_ID_PREFIX (e.g. "VAT"); never invent English in PDF.
	CompanyVATPrefix    string `json:"company_vat_prefix,omitempty"`
	CompanyRegistration string `json:"company_registration,omitempty"`
	// CompanyLocalityLine is city/region/postal joined with RECEIPT_LOCALITY_JOIN_SEP (no client invent).
	CompanyLocalityLine string `json:"company_locality_line,omitempty"`
	// BillToLocalityLine is bill-to city/region/postal joined the same way.
	BillToLocalityLine     string `json:"bill_to_locality_line,omitempty"`
	PrintActionLabel       string `json:"print_action_label"`
	PrintPendingLabel      string `json:"print_pending_label"`
	PrintPendingMs         string `json:"print_pending_ms"`
	DocumentTitle          string `json:"document_title"`
	PeriodLabel            string `json:"period_label"`
	PaidAtLabel            string `json:"paid_at_label,omitempty"`
	DownloadPdfLabel       string `json:"download_pdf_action_label"`
	DownloadPdfPending     string `json:"download_pdf_pending_label"`
	DownloadPdfFailed      string `json:"download_pdf_failed_message"`
	DownloadPdfDone        string `json:"download_pdf_done_message"`
	DownloadPdfFilenameFmt string `json:"download_pdf_filename_fmt"`
	// FirstPartyPDFHref is the authenticated API path for Trim-generated PDF
	// when Paddle has no invoice PDF URL (no client invent of download targets).
	FirstPartyPDFHref     string `json:"first_party_pdf_href,omitempty"`
	BackActionLabel       string `json:"back_action_label"`
	BackHref              string `json:"back_href"`
	SellerMissingMessage  string `json:"seller_missing_message"`
	SectionBillTo         string `json:"section_bill_to"`
	SectionInvoiceFrom    string `json:"section_invoice_from"`
	SectionInvoiceDetails string `json:"section_invoice_details"`
	SectionTransaction    string `json:"section_transaction"`
	SectionTaxBreakdown   string `json:"section_tax_breakdown"`
	SectionPeriod         string `json:"section_period"`
	SectionAmount         string `json:"section_amount"`
	SectionPayment        string `json:"section_payment"`
	SectionStatus         string `json:"section_status"`
	ColDescription        string `json:"col_description"`
	ColProduct            string `json:"col_product"`
	ColSKU                string `json:"col_sku"`
	ColQty                string `json:"col_qty"`
	ColUnit               string `json:"col_unit"`
	ColTaxRate            string `json:"col_tax_rate"`
	ColAmount             string `json:"col_amount"`
	LabelSubtotal         string `json:"label_subtotal"`
	LabelTax              string `json:"label_tax"`
	LabelTaxID            string `json:"label_tax_id"`
	LabelTotal            string `json:"label_total"`
	LabelAmountPaid       string `json:"label_amount_paid"`
	LabelInvoiceReference string `json:"label_invoice_reference"`
	LabelTransactionID    string `json:"label_transaction_id"`
	LabelCurrency         string `json:"label_currency"`
	LabelTaxPercent       string `json:"label_tax_percent"`
	LabelTaxTotal         string `json:"label_tax_total"`
	MerchantVia           string `json:"merchant_via"`
	HeaderMetaSep         string `json:"header_meta_sep"`
	EmptySKUPlaceholder   string `json:"empty_sku_placeholder"`
	NotFoundMessage       string `json:"not_found_message"`
	Footer                string `json:"footer"`
	IssuedPrefix          string `json:"issued_prefix"`
	// MoneyLocale is SITE_HTML_LANG for Intl.NumberFormat (no client navigator invent).
	MoneyLocale string `json:"money_locale"`
}

// applyReceiptChrome fills operator-owned labels shared by JSON detail + PDF.
func applyReceiptChrome(d *ReceiptDetail) {
	if d == nil {
		return
	}
	d.DocumentTitle = subscriptions.MessageForCode("RECEIPT_DOCUMENT_TITLE")
	d.SectionBillTo = subscriptions.MessageForCode("RECEIPT_SECTION_BILL_TO")
	d.SectionInvoiceFrom = subscriptions.MessageForCode("RECEIPT_SECTION_INVOICE_FROM")
	d.SectionInvoiceDetails = subscriptions.MessageForCode("RECEIPT_SECTION_INVOICE_DETAILS")
	d.SectionTransaction = subscriptions.MessageForCode("RECEIPT_SECTION_TRANSACTION")
	d.SectionTaxBreakdown = subscriptions.MessageForCode("RECEIPT_SECTION_TAX_BREAKDOWN")
	d.SectionPeriod = subscriptions.MessageForCode("RECEIPT_SECTION_PERIOD")
	d.SectionAmount = subscriptions.MessageForCode("RECEIPT_SECTION_AMOUNT")
	d.SectionPayment = subscriptions.MessageForCode("RECEIPT_SECTION_PAYMENT")
	d.SectionStatus = subscriptions.MessageForCode("RECEIPT_SECTION_STATUS")
	d.ColDescription = subscriptions.MessageForCode("RECEIPT_COL_DESCRIPTION")
	d.ColProduct = subscriptions.MessageForCode("RECEIPT_COL_PRODUCT")
	if strings.TrimSpace(d.ColProduct) == "" {
		d.ColProduct = d.ColDescription
	}
	d.ColSKU = subscriptions.MessageForCode("RECEIPT_COL_SKU")
	d.ColQty = subscriptions.MessageForCode("RECEIPT_COL_QTY")
	d.ColUnit = subscriptions.MessageForCode("RECEIPT_COL_UNIT")
	d.ColTaxRate = subscriptions.MessageForCode("RECEIPT_COL_TAX_RATE")
	d.ColAmount = subscriptions.MessageForCode("RECEIPT_COL_AMOUNT")
	d.LabelSubtotal = subscriptions.MessageForCode("RECEIPT_LABEL_SUBTOTAL")
	d.LabelTax = subscriptions.MessageForCode("RECEIPT_LABEL_TAX")
	d.LabelTaxID = subscriptions.MessageForCode("RECEIPT_LABEL_TAX_ID")
	d.LabelTotal = subscriptions.MessageForCode("RECEIPT_LABEL_TOTAL")
	d.LabelAmountPaid = subscriptions.MessageForCode("RECEIPT_LABEL_AMOUNT_PAID")
	d.LabelInvoiceReference = subscriptions.MessageForCode("RECEIPT_LABEL_INVOICE_REFERENCE")
	d.LabelTransactionID = subscriptions.MessageForCode("RECEIPT_LABEL_TRANSACTION_ID")
	d.LabelCurrency = subscriptions.MessageForCode("RECEIPT_LABEL_CURRENCY")
	d.LabelTaxPercent = subscriptions.MessageForCode("RECEIPT_LABEL_TAX_PERCENT")
	d.LabelTaxTotal = subscriptions.MessageForCode("RECEIPT_LABEL_TAX_TOTAL")
	d.MerchantVia = subscriptions.MessageForCode("RECEIPT_MERCHANT_VIA")
	d.HeaderMetaSep = subscriptions.MessageForCode("RECEIPT_HEADER_META_SEP")
	d.EmptySKUPlaceholder = subscriptions.MessageForCode("RECEIPT_EMPTY_SKU")
	d.Footer = subscriptions.MessageForCode("RECEIPT_FOOTER")
	d.IssuedPrefix = subscriptions.MessageForCode("RECEIPT_ISSUED_PREFIX")
	d.CompanyLocalityLine = joinLocalityLine(d.CompanyCity, d.CompanyRegion, d.CompanyPostalCode)
	d.BillToLocalityLine = joinLocalityPtrs(d.BillToCity, d.BillToRegion, d.BillToPostalCode)

	start := formatReceiptDay(d.PeriodStart)
	end := formatReceiptDay(d.PeriodEnd)
	if start != "" && end != "" {
		joinFmt := subscriptions.MessageForCode("RECEIPT_PERIOD_JOIN_FMT")
		if joinFmt == "" {
			d.PeriodLabel = start + " " + end
		} else {
			d.PeriodLabel = fmt.Sprintf(joinFmt, start, end)
		}
	} else {
		d.PeriodLabel = subscriptions.MessageForCode("RECEIPT_PERIOD_ONE_TIME")
	}
	d.PaidAtLabel = formatReceiptDay(d.PaidAt)
	if d.PaidAtLabel == "" {
		d.PaidAtLabel = formatReceiptDay(&d.CreatedAt)
	}
	if d.TaxRateBps > 0 {
		taxFmt := subscriptions.MessageForCode("RECEIPT_TAX_RATE_FMT")
		if taxFmt != "" {
			d.TaxRateLabel = fmt.Sprintf(taxFmt, float64(d.TaxRateBps)/100.0)
		}
		pctFmt := subscriptions.MessageForCode("RECEIPT_TAX_RATE_PERCENT_FMT")
		if pctFmt != "" {
			d.TaxRatePercent = fmt.Sprintf(pctFmt, float64(d.TaxRateBps)/100.0)
		}
	} else {
		d.TaxRatePercent = subscriptions.MessageForCode("RECEIPT_TAX_RATE_ZERO")
	}
	// Paddle tax-invoice product cell: name, then period, then price name.
	for i := range d.LineItems {
		d.LineItems[i].PeriodLabel = d.PeriodLabel
	}
}

type ListResponse struct {
	Items                   []ReceiptSummary `json:"items"`
	Meta                    pagination.Meta  `json:"meta"`
	EmptyMessage            string           `json:"empty_message"`
	ColDate                 string           `json:"col_date"`
	ColInvoice              string           `json:"col_invoice"`
	ColStatus               string           `json:"col_status"`
	ColTotal                string           `json:"col_total"`
	ColView                 string           `json:"col_view"`
	OpenLabel               string           `json:"open_action_label"`
	DownloadPdfLabel        string           `json:"download_pdf_action_label"`
	DownloadPdfFailed       string           `json:"download_pdf_failed_message"`
	DownloadPdfDone         string           `json:"download_pdf_done_message"`
	DownloadPdfFilenameFmt  string           `json:"download_pdf_filename_fmt"`
	PreviewFieldDescription string           `json:"preview_field_description"`
	MoneyLocale             string           `json:"money_locale"`
	TableRowActions         string           `json:"table_row_actions"`
	TableSelectAll          string           `json:"table_select_all"`
	TableSelectRow          string           `json:"table_select_row"`
	TableSelectedFmt        string           `json:"table_selected_fmt"`
	TableClearSelection     string           `json:"table_clear_selection"`
	SearchPlaceholder       string           `json:"search_placeholder"`
	SearchDescription       string           `json:"search_description"`
	Q                       string           `json:"q"`
}

func escapeILikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func (h *Handler) List(companyLegal, companySupport string) http.HandlerFunc {
	_ = companyLegal
	_ = companySupport
	return func(w http.ResponseWriter, r *http.Request) {
		if h.DB == nil {
			writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
			return
		}
		userID := middleware.UserIDFromContext(r.Context())
		if userID == "" {
			writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
			return
		}

		db := h.readPool()
		maxLimit, err := billingsettings.MaxPageSize(r.Context(), db)
		if err != nil {
			writeJSONErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		skipCap, err := billingsettings.SkipToMaxPages(r.Context(), db)
		if err != nil {
			writeJSONErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		params, err := pagination.Parse(r, maxLimit)
		if err != nil {
			writeJSONErr(w, http.StatusBadRequest, err.Error())
			return
		}

		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(q) > 200 {
			q = q[:200]
		}

		where := `user_id = $1`
		args := []interface{}{userID}
		argN := 2
		if q != "" {
			like := "%" + escapeILikePattern(q) + "%"
			where += fmt.Sprintf(` and (
				coalesce(paddle_transaction_id, '') ilike $%d escape '\'
				or coalesce(paddle_invoice_number, '') ilike $%d escape '\'
				or coalesce(status, '') ilike $%d escape '\'
				or coalesce(bill_to_email, '') ilike $%d escape '\'
				or coalesce(bill_to_name, '') ilike $%d escape '\'
				or coalesce(bill_to_company, '') ilike $%d escape '\'
			)`, argN, argN, argN, argN, argN, argN)
			args = append(args, like)
			argN++
		}

		var total int
		countSQL := `select count(*) from public.billing_receipts where ` + where
		if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "RECEIPT_OPERATION_FAILED")
			return
		}

		listSQL := fmt.Sprintf(`
			select id::text, paddle_transaction_id, paddle_invoice_number, paddle_invoice_pdf_url,
			       status, currency_code, subtotal_cents, tax_cents, total_cents,
			       bill_to_name, bill_to_email, bill_to_company,
			       period_start::text, period_end::text, paid_at::text, created_at::text
			from public.billing_receipts
			where %s
			order by coalesce(paid_at, created_at) desc
			offset $%d limit $%d
		`, where, argN, argN+1)
		listArgs := append(append([]interface{}{}, args...), params.Skip, params.Limit)
		rows, err := db.Query(r.Context(), listSQL, listArgs...)
		if err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "RECEIPT_OPERATION_FAILED")
			return
		}
		defer rows.Close()

		items := make([]ReceiptSummary, 0)
		for rows.Next() {
			var item ReceiptSummary
			if err := rows.Scan(
				&item.ID, &item.PaddleTransactionID, &item.PaddleInvoiceNumber, &item.PaddleInvoicePDFURL,
				&item.Status, &item.CurrencyCode, &item.SubtotalCents, &item.TaxCents, &item.TotalCents,
				&item.BillToName, &item.BillToEmail, &item.BillToCompany,
				&item.PeriodStart, &item.PeriodEnd, &item.PaidAt, &item.CreatedAt,
			); err != nil {
				writeJSONErr(w, http.StatusInternalServerError, "RECEIPT_OPERATION_FAILED")
				return
			}
			item.StatusLabel = subscriptions.ReceiptStatusLabel(item.Status)
			if item.PaddleInvoiceNumber != nil && strings.TrimSpace(*item.PaddleInvoiceNumber) != "" {
				item.DisplayID = strings.TrimSpace(*item.PaddleInvoiceNumber)
			} else {
				item.DisplayID = item.PaddleTransactionID
			}
			if item.PaidAt != nil && strings.TrimSpace(*item.PaidAt) != "" {
				item.DateLabel = subscriptions.FormatUTCDateFromRFC3339(*item.PaidAt)
			} else {
				item.DateLabel = subscriptions.FormatUTCDateFromRFC3339(item.CreatedAt)
			}
			items = append(items, item)
		}

		moneyLocale := strings.TrimSpace(subscriptions.MessageForCode("SITE_HTML_LANG"))
		if moneyLocale == "" {
			writeJSONErr(w, http.StatusServiceUnavailable, "RECEIPT_MONEY_LOCALE_REQUIRED")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListResponse{
			Items:                   items,
			Meta:                    pagination.BuildMeta(params, total, skipCap),
			EmptyMessage:            subscriptions.MessageForCode("RECEIPTS_EMPTY"),
			ColDate:                 subscriptions.MessageForCode("RECEIPTS_COL_DATE"),
			ColInvoice:              subscriptions.MessageForCode("RECEIPTS_COL_INVOICE"),
			ColStatus:               subscriptions.MessageForCode("RECEIPTS_COL_STATUS"),
			ColTotal:                subscriptions.MessageForCode("RECEIPTS_COL_TOTAL"),
			ColView:                 subscriptions.MessageForCode("RECEIPTS_COL_VIEW"),
			OpenLabel:               subscriptions.MessageForCode("RECEIPTS_OPEN_LABEL"),
			DownloadPdfLabel:        subscriptions.MessageForCode("RECEIPT_DOWNLOAD_PDF"),
			DownloadPdfFailed:       subscriptions.MessageForCode("RECEIPT_DOWNLOAD_PDF_FAILED"),
			DownloadPdfDone:         subscriptions.MessageForCode("RECEIPT_PDF_DOWNLOAD_DONE"),
			DownloadPdfFilenameFmt:  subscriptions.MessageForCode("RECEIPT_PDF_FILENAME_FMT"),
			PreviewFieldDescription: subscriptions.MessageForCode("RECEIPTS_PREVIEW_FIELD_DESC"),
			MoneyLocale:             moneyLocale,
			TableRowActions:         subscriptions.MessageForCode("TABLE_ROW_ACTIONS"),
			TableSelectAll:          subscriptions.MessageForCode("TABLE_SELECT_ALL"),
			TableSelectRow:          subscriptions.MessageForCode("TABLE_SELECT_ROW"),
			TableSelectedFmt:        subscriptions.MessageForCode("TABLE_SELECTED_FMT"),
			TableClearSelection:     subscriptions.MessageForCode("TABLE_CLEAR_SELECTION"),
			SearchPlaceholder:       subscriptions.MessageForCode("RECEIPTS_SEARCH"),
			SearchDescription:       subscriptions.MessageForCode("RECEIPTS_SEARCH_DESC"),
			Q:                       q,
		})
	}
}

type SellerLetterhead struct {
	LegalName    string
	SupportEmail string
	LogoURL      string
	AddressLine1 string
	AddressLine2 string
	City         string
	Region       string
	PostalCode   string
	Country      string
	VATID        string
	Registration string
}

func (h *Handler) Get(seller SellerLetterhead) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.DB == nil {
			writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
			return
		}
		userID := middleware.UserIDFromContext(r.Context())
		if userID == "" {
			writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
			return
		}
		receiptID := chi.URLParam(r, "receiptId")
		if receiptID == "" {
			writeJSONErr(w, http.StatusBadRequest, "RECEIPT_ID_REQUIRED")
			return
		}

		d, err := LoadDetail(r.Context(), h.readPool(), receiptID, userID, seller)
		if err != nil {
			code := err.Error()
			if code == "RECEIPT_NOT_FOUND" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":             subscriptions.MessageForCode("RECEIPT_NOT_FOUND_ERROR"),
					"not_found_message": subscriptions.MessageForCode("RECEIPT_NOT_FOUND"),
					"back_action_label": subscriptions.MessageForCode("RECEIPT_BACK"),
					"back_href":         subscriptions.MessageForCode("APP_PATH_DASHBOARD"),
				})
				return
			}
			if code == "RECEIPT_MONEY_LOCALE_REQUIRED" {
				writeJSONErr(w, http.StatusServiceUnavailable, code)
				return
			}
			writeJSONErr(w, http.StatusInternalServerError, "RECEIPT_OPERATION_FAILED")
			return
		}

		// Paddle invoice PDF URLs expire. Re-fetch a fresh signed URL on every detail view.
		if h.Paddle != nil && d.PaddleTransactionID != "" {
			if fresh, pdfErr := h.Paddle.GetTransactionInvoicePDF(r.Context(), d.PaddleTransactionID); pdfErr == nil && fresh != "" {
				d.PaddleInvoicePDFURL = &fresh
				_, _ = h.DB.Exec(r.Context(), `
					update public.billing_receipts
					set paddle_invoice_pdf_url = $1, updated_at = now()
					where id = $2::uuid
				`, fresh, receiptID)
			}
		}

		d.FirstPartyPDFHref = "/api/v1/billing/receipts/" + d.ID + "/pdf"
		d.BackActionLabel = subscriptions.MessageForCode("RECEIPT_BACK")
		d.BackHref = subscriptions.MessageForCode("APP_PATH_DASHBOARD")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(d)
	}
}

// DownloadPDF serves a first-party invoice PDF for the authenticated owner.
// Always generates from stored receipt rows (fail closed if receipt missing).
func (h *Handler) DownloadPDF(seller SellerLetterhead) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.DB == nil {
			writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
			return
		}
		userID := middleware.UserIDFromContext(r.Context())
		if userID == "" {
			writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
			return
		}
		receiptID := chi.URLParam(r, "receiptId")
		if receiptID == "" {
			writeJSONErr(w, http.StatusBadRequest, "RECEIPT_ID_REQUIRED")
			return
		}

		d, err := LoadDetail(r.Context(), h.readPool(), receiptID, userID, seller)
		if err != nil {
			code := err.Error()
			switch code {
			case "RECEIPT_NOT_FOUND":
				writeJSONErr(w, http.StatusNotFound, "RECEIPT_NOT_FOUND_ERROR")
			case "RECEIPT_MONEY_LOCALE_REQUIRED":
				writeJSONErr(w, http.StatusServiceUnavailable, code)
			default:
				writeJSONErr(w, http.StatusInternalServerError, "RECEIPT_OPERATION_FAILED")
			}
			return
		}

		writeReceiptPDFResponse(w, d)
	}
}

// WritePDFResponse builds and writes the first-party PDF for a loaded detail.
func WritePDFResponse(w http.ResponseWriter, d ReceiptDetail) {
	writeReceiptPDFResponse(w, d)
}

func writeReceiptPDFResponse(w http.ResponseWriter, d ReceiptDetail) {
	d.CompanyLogoJPEG = FetchCompanyLogoJPEG(d.CompanyLogoURL)

	pdfBytes, err := BuildReceiptPDF(d)
	if err != nil || len(pdfBytes) == 0 {
		switch {
		case errors.Is(err, errPDFTitleRequired):
			writeJSONErr(w, http.StatusServiceUnavailable, "RECEIPT_PDF_TITLE_REQUIRED")
		case errors.Is(err, errPDFCurrencyRequired):
			writeJSONErr(w, http.StatusServiceUnavailable, "RECEIPT_PDF_CURRENCY_REQUIRED")
		case errors.Is(err, errPDFMoneyLocaleRequired):
			writeJSONErr(w, http.StatusServiceUnavailable, "RECEIPT_PDF_MONEY_LOCALE_REQUIRED")
		default:
			writeJSONErr(w, http.StatusInternalServerError, "RECEIPT_OPERATION_FAILED")
		}
		return
	}

	fmtMsg := strings.TrimSpace(d.DownloadPdfFilenameFmt)
	if fmtMsg == "" {
		fmtMsg = subscriptions.MessageForCode("RECEIPT_PDF_FILENAME_FMT")
	}
	safeID := strings.TrimSpace(d.DisplayID)
	if safeID == "" {
		writeJSONErr(w, http.StatusServiceUnavailable, "RECEIPT_PDF_FILENAME_REQUIRED")
		return
	}
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, safeID)
	safe = strings.Trim(safe, "-")
	if fmtMsg == "" || safe == "" || !strings.Contains(fmtMsg, "%s") {
		writeJSONErr(w, http.StatusServiceUnavailable, "RECEIPT_PDF_FILENAME_REQUIRED")
		return
	}
	filename := fmt.Sprintf(fmtMsg, safe)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	// Custom header mirrors Content-Disposition so browsers can read the name even when
	// Content-Disposition is omitted from Access-Control-Expose-Headers on older deploys.
	w.Header().Set("X-Trim-Download-Filename", filename)
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

// companyWordmark is the seller letterhead monogram when no logo URL is set.
// Computed server-side so the web client never invents capitalization.
func companyWordmark(legalName string) string {
	name := strings.TrimSpace(legalName)
	if name == "" {
		return ""
	}
	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return ""
	}
	return string(unicode.ToUpper(r))
}

func formatReceiptDay(iso *string) string {
	if iso == nil {
		return ""
	}
	raw := strings.TrimSpace(*iso)
	if raw == "" {
		return ""
	}
	// Postgres ::text often yields "+00" instead of "+00:00".
	if strings.HasSuffix(raw, "+00") {
		raw = raw + ":00"
	} else if strings.HasSuffix(raw, "-00") {
		raw = raw + ":00"
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999Z07:00",
		"2006-01-02 15:04:05.999999-07",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			// Paddle invoice day style: "9th October 2026"
			return formatPaddleInvoiceDay(t.UTC())
		}
	}
	// Fail closed: never invent a truncated date from unparsed input.
	return ""
}

func formatPaddleInvoiceDay(t time.Time) string {
	d := t.Day()
	return fmt.Sprintf("%d%s %s %d", d, englishDayOrdinal(d), t.Format("January"), t.Year())
}

func englishDayOrdinal(d int) string {
	if d%100 >= 11 && d%100 <= 13 {
		return "th"
	}
	switch d % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

func joinLocalityLine(city, region, postal string) string {
	parts := make([]string, 0, 3)
	for _, p := range []string{strings.TrimSpace(city), strings.TrimSpace(region), strings.TrimSpace(postal)} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	sep := subscriptions.MessageForCode("RECEIPT_LOCALITY_JOIN_SEP")
	if sep == "" {
		// Fail closed: do not invent a space separator when site_messages is cold.
		return ""
	}
	return strings.Join(parts, sep)
}

func joinLocalityPtrs(city, region, postal *string) string {
	deref := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	return joinLocalityLine(deref(city), deref(region), deref(postal))
}
