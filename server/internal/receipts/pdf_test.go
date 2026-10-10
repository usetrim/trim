package receipts

import (
	"bytes"
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
	if !strings.Contains(body, "9th October 2026 - $1.00") {
		t.Fatal("header meta separator spaces were stripped")
	}
	if !strings.Contains(body, "FlateDecode") {
		t.Fatal("expected FlateDecode RGB mark stream")
	}
	// Footer brand under mark (detail page: TrimWordmark + company name).
	if !strings.Contains(body, "(Trim)") {
		t.Fatal("expected company brand under footer mark")
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
