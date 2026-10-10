package receipts

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestBuildReceiptPDFMatchesDetailPatterns(t *testing.T) {
	et := "ET"
	pay := "visa ending 7299"
	price := "Pro (monthly)"
	d := ReceiptDetail{
		ReceiptSummary: ReceiptSummary{
			StatusLabel:         "PAID",
			CurrencyCode:        "USD",
			SubtotalCents:       100,
			TaxCents:            0,
			TotalCents:          100,
			DisplayID:           "43682-10002",
			PaddleTransactionID: "txn_01m4gn6ccarwgk7neq29sn8yjn",
			BillToEmail:         "ihateofficial59@gmail.com",
		},
		DocumentTitle:         "Tax invoice",
		PaidAtLabel:           "9th October 2026",
		HeaderMetaSep:         " - ",
		MoneyLocale:           "en",
		TaxRatePercent:        "0%",
		PeriodLabel:           "9th October 2026 - 9th November 2026",
		BillToCountry:         &et,
		PaymentMethodSummary:  &pay,
		CompanyLegalName:      "Trim",
		MerchantVia:           "via Paddle.com",
		SectionBillTo:         "Invoice to",
		SectionInvoiceFrom:    "Invoice from",
		SectionInvoiceDetails: "Invoice details",
		SectionTransaction:    "Transaction",
		SectionTaxBreakdown:   "Tax breakdown",
		SectionPeriod:         "Billing period",
		SectionPayment:        "Payment method",
		LabelInvoiceReference: "Invoice reference",
		LabelTransactionID:    "Transaction",
		LabelCurrency:         "Currency code",
		LabelSubtotal:         "Subtotal",
		LabelTax:              "VAT",
		LabelTotal:            "Total",
		LabelAmountPaid:       "Amount paid",
		LabelTaxPercent:       "Tax %",
		LabelTaxTotal:         "Tax total",
		ColProduct:            "Product",
		ColQty:                "Qty",
		ColUnit:               "Unit price",
		ColTaxRate:            "Tax rate",
		ColAmount:             "Amount",
		Footer:                "Questions about this invoice? Contact support using the address on this document. Card statements may show Paddle as the merchant of record.",
		LineItems: []LineItem{
			{
				Position:        1,
				Description:     "Pro",
				Quantity:        1,
				UnitAmountCents: 100,
				AmountCents:     100,
				PriceName:       &price,
				PeriodLabel:     "9th October 2026 - 9th November 2026",
			},
		},
	}

	pdf, err := BuildReceiptPDF(d)
	if err != nil {
		t.Fatalf("BuildReceiptPDF: %v", err)
	}
	if len(pdf) < 500 || !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("expected PDF bytes, got %d", len(pdf))
	}
	if out := strings.TrimSpace(os.Getenv("TRIM_RECEIPT_PDF_OUT")); out != "" {
		if err := os.WriteFile(out, pdf, 0o644); err != nil {
			t.Fatalf("write sample pdf: %v", err)
		}
	}
	body := string(pdf)
	if !strings.Contains(body, "/SMask") {
		t.Fatal("expected soft-mask logo (transparent mark), got opaque flatten")
	}
	if !strings.Contains(body, "Ethiopia") {
		t.Fatal("expected full country name Ethiopia, not ISO code")
	}
	if !strings.Contains(body, "Card statements may show Paddle") {
		t.Fatal("footer text was truncated")
	}
	// Header meta is separate Tj runs (muted date + sep + bold amount).
	if !strings.Contains(body, "(9th October 2026)") || !strings.Contains(body, "($1.00)") {
		t.Fatal("expected header paid date and total amount")
	}
	if !strings.Contains(body, "( - )") {
		t.Fatal("header meta separator spaces were stripped")
	}
	if !strings.Contains(body, "FlateDecode") {
		t.Fatal("expected FlateDecode RGB mark stream")
	}
	// Locality line under footer mark includes company legal name (detail page).
	if !strings.Contains(body, "(Trim)") {
		t.Fatal("expected company brand in PDF letterhead/footer locality")
	}
	if !strings.Contains(body, "Payment method: ") {
		t.Fatal("expected spaced Payment method label (bold-width must not collide)")
	}
	// Amount paid must use the same hairline weight as Subtotal/VAT/Total (0.4 w),
	// not a thicker 1.2 w emphasis stroke.
	if strings.Contains(body, "1.2 w\n0.20 0.20 0.20 RG") {
		t.Fatal("expected no thick Amount paid separator; use same 0.4w hairline as other totals")
	}
	// Structural parity with receipt-view.tsx sections.
	for _, want := range []string{
		"Tax invoice",
		"Invoice to",
		"Invoice from",
		"Invoice details",
		"Transaction",
		"Tax breakdown",
		"Amount paid",
		"via Paddle.com",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing detail-page section/label %q", want)
		}
	}
}

func TestApproxTextWidthFontBoldPaymentMethod(t *testing.T) {
	// AFM still used for right-aligned / centered text; keep bold wider than naive.
	prefix := "Payment method: "
	got := approxTextWidthFont(prefix, 13, true)
	naive := float64(len([]rune(prefix))) * 13 * 0.50
	if got <= naive {
		t.Fatalf("bold AFM width %.2f should exceed naive %.2f", got, naive)
	}
}

func TestBuildReceiptPDFLabeledUsesSequentialTj(t *testing.T) {
	// writeLabeled must emit bold prefix Tj then regular value Tj (cursor advance),
	// not a second absolute Tm that can overlap when widths are wrong.
	pay := "visa - 7299"
	d := ReceiptDetail{
		ReceiptSummary: ReceiptSummary{
			StatusLabel:         "PAID",
			CurrencyCode:        "USD",
			TotalCents:          100,
			DisplayID:           "43682-10003",
			BillToEmail:         "a@b.com",
			PaddleTransactionID: "txn_x",
		},
		DocumentTitle:         "Tax invoice",
		MoneyLocale:           "en",
		PaidAtLabel:           "9th October 2026",
		HeaderMetaSep:         " - ",
		CompanyLegalName:      "Trim",
		SectionBillTo:         "Invoice to",
		SectionInvoiceFrom:    "Invoice from",
		SectionInvoiceDetails: "Invoice details",
		SectionPayment:        "Payment method",
		PaymentMethodSummary:  &pay,
		LabelInvoiceReference: "Invoice reference",
		SectionPeriod:         "Billing period",
		PeriodLabel:           "9th October 2026 - 9th November 2026",
		LabelTransactionID:    "Transaction",
		LabelCurrency:         "Currency code",
		LabelSubtotal:         "Subtotal",
		LabelTax:              "VAT",
		LabelTotal:            "Total",
		LabelAmountPaid:       "Amount paid",
		ColProduct:            "Product",
		ColQty:                "Qty",
		ColUnit:               "Unit",
		ColTaxRate:            "Tax",
		ColAmount:             "Amount",
		LineItems:             []LineItem{{Position: 1, Description: "Team", Quantity: 1, UnitAmountCents: 100, AmountCents: 100}},
	}
	pdf, err := BuildReceiptPDF(d)
	if err != nil {
		t.Fatal(err)
	}
	body := string(pdf)
	// Bold label run then regular value run (Payment method).
	if !strings.Contains(body, "(Payment method: ) Tj") {
		t.Fatal("expected bold Payment method prefix Tj")
	}
	if !strings.Contains(body, "(visa - 7299) Tj") {
		t.Fatal("expected payment value Tj after label (sequential cursor advance)")
	}
	if !strings.Contains(body, "(Invoice reference: ) Tj") || !strings.Contains(body, "(43682-10003) Tj") {
		t.Fatal("expected Invoice reference label/value sequential Tj")
	}
}

func TestCountryDisplayName(t *testing.T) {
	if got := countryDisplayName("ET", "en"); got != "Ethiopia" {
		t.Fatalf("got %q", got)
	}
	if got := countryDisplayName("et", "en"); got != "Ethiopia" {
		t.Fatalf("got %q", got)
	}
}
