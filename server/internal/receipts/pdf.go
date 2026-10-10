package receipts

import (
	"bytes"
	"compress/zlib"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

//go:embed assets/trim-mark-black.png
var trimMarkBlackPNG []byte

var (
	errPDFTitleRequired       = errors.New("receipt_pdf_title_required")
	errPDFCurrencyRequired    = errors.New("receipt_pdf_currency_required")
	errPDFMoneyLocaleRequired = errors.New("receipt_pdf_money_locale_required")
)

// pdfImage is an embedded letterhead/footer mark. Prefer RGB+Mask (true alpha)
// so marks sit on gray bands without a white box. JPEG is for remote company logos.
type pdfImage struct {
	Width, Height int
	JPEG          []byte // DCTDecode path
	RGB           []byte // raw DeviceRGB, len = W*H*3
	Mask          []byte // raw DeviceGray soft-mask, len = W*H
}

type pdfOp struct {
	kind string
	text string
	size int
	bold bool
	x    float64
	// table cells (product, qty, unit, tax rate, amount)
	cells [5]string
	// product_row / two_col / header lines
	lines []string
	left  string
	right string
	logo  *pdfImage
	// band / header height
	h float64
	// rgb 0..1 for fill / text color
	r, g, b float64
	// center text block (footer)
	center bool
	// wrap width in approximate chars (0 = no wrap / truncate)
	wrap int
}

// Layout constants mirror apps/{web,admin} receipt-view.tsx + @media print
// (globals.css). Use 1 CSS px = 1 PDF pt so Tailwind sizes/spacing match the
// live invoice (not a 0.75× downscale that made Download look unlike Print).
//
//	print:px-8 / section pad-x 2rem = 32
//	print header/footer pad 1.75rem 2rem = py-7 (28) px-8 (32)
//	text-[13px]=13  text-[15px]=15  text-[12px]=12  text-[11px]=11
//	sm:text-[1.5rem]=24  badge text-[10px]=10
//	zinc-100 #f4f4f5  border zinc-200  status #27a644  sky-700 price link
const (
	pdfMarginL  = 32.0 // print:px-8
	pdfMarginR  = 32.0
	pdfPageW    = 612.0
	pdfContentR = pdfPageW - pdfMarginR
	pdfContentW = pdfContentR - pdfMarginL

	pdfPtBody     = 13 // text-[13px]
	pdfPtTitle    = 24 // sm:text-[1.5rem]
	pdfPtBrand    = 15 // text-[15px]
	pdfPtAmount   = 15 // Amount paid text-[15px]
	pdfPtMeta     = 13 // header meta text-[13px]
	pdfPtVia      = 11 // text-[11px]
	pdfPtTableH   = 12 // th text-[12px]
	pdfPtFooter   = 12 // footer copy text-[12px]
	pdfPtLocality = 11 // locality text-[11px]
	pdfPtBadge    = 10 // badge text-[10px]

	pdfMarkLG = 28.0 // TrimWordmark size lg
	pdfMarkSM = 18.0 // TrimWordmark size sm
)

// BuildReceiptPDF renders a first-party invoice PDF matching the dashboard
// tax-invoice detail page (header band, parties grid, stacked product cell,
// totals, tax breakdown, centered footer band with transparent mark).
// Source of truth: receipt-view.tsx (web ≡ admin) + print CSS - not a separate design.
func BuildReceiptPDF(d ReceiptDetail) ([]byte, error) {
	title := strings.TrimSpace(d.DocumentTitle)
	if title == "" {
		return nil, errPDFTitleRequired
	}
	currency := strings.ToUpper(strings.TrimSpace(d.CurrencyCode))
	if currency == "" {
		return nil, errPDFCurrencyRequired
	}
	moneyLocale := strings.TrimSpace(d.MoneyLocale)
	if moneyLocale == "" {
		return nil, errPDFMoneyLocaleRequired
	}

	var ops []pdfOp
	add := func(s string, size int, bold bool) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if size < 1 {
			size = 10
		}
		ops = append(ops, pdfOp{kind: "text", text: s, size: size, bold: bold, x: pdfMarginL})
	}
	blank := func(h float64) {
		if h < 1 {
			h = 10
		}
		ops = append(ops, pdfOp{kind: "gap", h: h})
	}
	rule := func() {
		ops = append(ops, pdfOp{kind: "rule"})
	}
	// muted = print:text-zinc-700 (Subtotal/VAT); Total/Amount paid = zinc-900.
	pair := func(label, value string, size int, bold, muted bool) {
		label = strings.TrimSpace(label)
		value = strings.TrimSpace(value)
		if label == "" || value == "" {
			return
		}
		r, g, b := 0.0, 0.0, 0.0
		if muted {
			r, g, b = 0.35, 0.35, 0.35 // zinc-700
		}
		ops = append(ops, pdfOp{
			kind:  "right_pair",
			left:  label,
			right: value,
			size:  size,
			bold:  bold,
			r:     r,
			g:     g,
			b:     b,
		})
	}
	inline := func(label, value string, size int) {
		label = strings.TrimSpace(label)
		value = strings.TrimSpace(value)
		if label == "" || value == "" {
			return
		}
		ops = append(ops, pdfOp{
			kind:  "labeled",
			left:  label,
			right: value,
			size:  size,
			x:     pdfMarginL,
		})
	}

	totalStr, err := formatCents(d.TotalCents, currency, moneyLocale)
	if err != nil {
		return nil, err
	}
	subtotalStr, err := formatCents(d.SubtotalCents, currency, moneyLocale)
	if err != nil {
		return nil, err
	}
	taxStr, err := formatCents(d.TaxCents, currency, moneyLocale)
	if err != nil {
		return nil, err
	}

	// Keep surrounding spaces on the separator (TrimSpace would turn " - " into "-").
	metaSep := d.HeaderMetaSep
	if strings.TrimSpace(metaSep) == "" {
		metaSep = " - "
	}
	headerMeta := strings.TrimSpace(d.PaidAtLabel)
	if headerMeta != "" {
		headerMeta = headerMeta + metaSep + totalStr
	} else {
		headerMeta = totalStr
	}

	// Same as React: company_logo_url first, else TrimWordmark (soft-mask black).
	var headerLogo *pdfImage
	if len(d.CompanyLogoJPEG) > 0 {
		if lg, err := normalizeLogoImage(d.CompanyLogoJPEG); err == nil && lg != nil {
			headerLogo = lg
		}
	}
	if headerLogo == nil {
		if lg, err := blackLetterheadImage(1); err == nil && lg != nil {
			headerLogo = lg
		}
	}
	// header: print py-7 + title/meta or mark+brand+via (fit US Letter with body 13pt).
	headerH := 96.0
	if headerLogo != nil {
		headerH = 104
	}
	// print:bg-zinc-100 / #f4f4f5
	ops = append(ops, pdfOp{
		kind:  "header_block",
		h:     headerH,
		text:  title,
		left:  strings.TrimSpace(d.StatusLabel),
		right: headerMeta,
		lines: []string{strings.TrimSpace(d.CompanyLegalName), strings.TrimSpace(d.MerchantVia), strings.TrimSpace(d.PaidAtLabel), totalStr, metaSep},
		logo:  headerLogo,
		r:     0.957,
		g:     0.957,
		b:     0.961,
	})
	// parties section print:py-8 (slightly tightened to keep one-page letter fit)
	blank(28)

	// --- Parties (md:grid-cols-2 gap-10): bill-to LEFT includes payment pt-2 ---
	leftLines := make([]string, 0, 14)
	rightLines := make([]string, 0, 12)
	if s := strings.TrimSpace(d.SectionBillTo); s != "" {
		leftLines = append(leftLines, "#"+s, "#GAP:8") // h2 then space-y-2 (8px)
	}
	if d.BillToName != nil {
		leftLines = append(leftLines, strings.TrimSpace(*d.BillToName))
	}
	if d.BillToCompany != nil {
		leftLines = append(leftLines, strings.TrimSpace(*d.BillToCompany))
	}
	leftLines = append(leftLines, strings.TrimSpace(d.BillToEmail))
	if d.BillToAddressLine1 != nil {
		leftLines = append(leftLines, strings.TrimSpace(*d.BillToAddressLine1))
	}
	if d.BillToAddressLine2 != nil {
		leftLines = append(leftLines, strings.TrimSpace(*d.BillToAddressLine2))
	}
	if s := strings.TrimSpace(d.BillToLocalityLine); s != "" {
		leftLines = append(leftLines, s)
	}
	if d.BillToCountry != nil {
		leftLines = append(leftLines, countryDisplayName(*d.BillToCountry, moneyLocale))
	}
	if d.TaxID != nil && strings.TrimSpace(*d.TaxID) != "" && strings.TrimSpace(d.LabelTaxID) != "" {
		leftLines = append(leftLines, "#GAP:4", "#L:"+strings.TrimSpace(d.LabelTaxID)+"\x1e"+strings.TrimSpace(*d.TaxID)) // pt-1
	}
	if d.PaymentMethodSummary != nil && strings.TrimSpace(*d.PaymentMethodSummary) != "" && strings.TrimSpace(d.SectionPayment) != "" {
		// Inside bill-to column: <p className="pt-2"> — not after the whole grid.
		leftLines = append(leftLines, "#GAP:8", "#L:"+strings.TrimSpace(d.SectionPayment)+"\x1e"+strings.TrimSpace(*d.PaymentMethodSummary))
	}

	if s := strings.TrimSpace(d.SectionInvoiceFrom); s != "" {
		rightLines = append(rightLines, "#"+s, "#GAP:8")
	}
	rightLines = append(rightLines, strings.TrimSpace(d.CompanyLegalName))
	if s := strings.TrimSpace(d.CompanyAddressLine1); s != "" {
		rightLines = append(rightLines, s)
	}
	if s := strings.TrimSpace(d.CompanyAddressLine2); s != "" {
		rightLines = append(rightLines, s)
	}
	if s := strings.TrimSpace(d.CompanyLocalityLine); s != "" {
		rightLines = append(rightLines, s)
	}
	if s := strings.TrimSpace(d.CompanyCountry); s != "" {
		rightLines = append(rightLines, s)
	}
	if d.CompanyVATID != "" {
		prefix := strings.TrimSpace(d.CompanyVATPrefix)
		if prefix != "" {
			rightLines = append(rightLines, "#GAP:4", "#L:"+prefix+"\x1e"+d.CompanyVATID) // pt-1
		} else {
			rightLines = append(rightLines, d.CompanyVATID)
		}
	}
	if s := strings.TrimSpace(d.CompanyRegistration); s != "" {
		rightLines = append(rightLines, s)
	}

	ops = append(ops, pdfOp{kind: "two_col", lines: encodeTwoCol(leftLines, rightLines)})
	blank(28) // parties py-8 bottom

	// --- Invoice details (pb-6) ---
	add(d.SectionInvoiceDetails, pdfPtBody, true)
	blank(12) // mt-3
	inline(d.LabelInvoiceReference, d.DisplayID, pdfPtBody)
	inline(d.SectionPeriod, d.PeriodLabel, pdfPtBody)
	inline(d.LabelTransactionID, d.PaddleTransactionID, pdfPtBody)
	inline(d.LabelCurrency, currency, pdfPtBody)
	blank(20) // pb-6
	rule()    // border-t border-border
	blank(20) // transaction py-6 top

	// --- Transaction ---
	add(d.SectionTransaction, pdfPtBody, true)
	blank(14) // mb-4
	productCol := strings.TrimSpace(d.ColProduct)
	if productCol == "" {
		productCol = d.ColDescription
	}
	ops = append(ops, pdfOp{
		kind:  "table_header",
		size:  pdfPtTableH,
		bold:  true,
		cells: [5]string{productCol, d.ColQty, d.ColUnit, d.ColTaxRate, d.ColAmount},
	})
	taxPercent := strings.TrimSpace(d.TaxRatePercent)
	for _, li := range d.LineItems {
		unitStr, err := formatCents(li.UnitAmountCents, currency, moneyLocale)
		if err != nil {
			return nil, err
		}
		amtStr, err := formatCents(li.AmountCents, currency, moneyLocale)
		if err != nil {
			return nil, err
		}
		stack := make([]string, 0, 3)
		if s := strings.TrimSpace(li.Description); s != "" {
			stack = append(stack, s)
		}
		linePeriod := strings.TrimSpace(li.PeriodLabel)
		if linePeriod == "" {
			linePeriod = strings.TrimSpace(d.PeriodLabel)
		}
		if linePeriod != "" {
			stack = append(stack, linePeriod)
		}
		if li.PriceName != nil {
			if pn := strings.TrimSpace(*li.PriceName); pn != "" {
				stack = append(stack, pn)
			}
		}
		ops = append(ops, pdfOp{
			kind:  "product_row",
			size:  pdfPtBody, // td text-[13px]
			lines: stack,
			cells: [5]string{"", fmt.Sprintf("%d", li.Quantity), unitStr, taxPercent, amtStr},
		})
	}
	blank(8) // mt-2 before totals

	// Totals: max-w-[240px], border-b each row, Amount paid text-[15px] bold.
	pair(d.LabelSubtotal, subtotalStr, pdfPtBody, false, true)
	pair(d.LabelTax, taxStr, pdfPtBody, false, true)
	pair(d.LabelTotal, totalStr, pdfPtBody, false, false)
	pair(d.LabelAmountPaid, totalStr, pdfPtAmount, true, false)
	blank(24) // mt-8 before tax breakdown

	if strings.TrimSpace(d.SectionTaxBreakdown) != "" {
		add(d.SectionTaxBreakdown, pdfPtBody, true)
		blank(8) // mt-2
		ops = append(ops, pdfOp{
			kind:  "tax_table",
			size:  pdfPtTableH,
			cells: [5]string{d.LabelTaxPercent, d.LabelTax, taxPercent, taxStr, d.LabelTaxTotal},
			right: taxStr,
		})
	}

	// React: footer only when receipt.footer is set.
	if s := strings.TrimSpace(d.Footer); s != "" {
		blank(14) // mt-4
		addrParts := make([]string, 0, 4)
		for _, p := range []string{d.CompanyLegalName, d.CompanyAddressLine1, d.CompanyLocalityLine, d.CompanyCountry} {
			p = strings.TrimSpace(p)
			if p != "" {
				addrParts = append(addrParts, p)
			}
		}
		addrLine := strings.Join(addrParts, ", ")
		var footerLogo *pdfImage
		// print mark = full opacity (screen-only opacity-70).
		if lg, err := blackLetterheadImage(1); err == nil && lg != nil {
			footerLogo = lg
		}
		// py-7*2 + copy + gap-3 + sm mark + gap-3 + locality
		footerH := 120.0
		if footerLogo != nil {
			footerH = 140
		}
		ops = append(ops, pdfOp{
			kind:  "footer_block",
			h:     footerH,
			lines: []string{s},
			left:  addrLine,
			logo:  footerLogo,
			r:     0.957,
			g:     0.957,
			b:     0.961,
		})
	}

	return writeReceiptPDF(ops)
}

func encodeTwoCol(left, right []string) []string {
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out = append(out, l+"\x1f"+r)
	}
	return out
}

// countryDisplayName mirrors the web Intl.DisplayNames region lookup.
func countryDisplayName(code, locale string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	if len(code) != 2 {
		return code
	}
	code = strings.ToUpper(code)
	tag := language.Make("und-" + code)
	namer := display.English.Regions()
	if loc := strings.TrimSpace(locale); loc != "" {
		if base, err := language.Parse(loc); err == nil {
			namer = display.Regions(base)
		}
	}
	name := strings.TrimSpace(namer.Name(tag))
	if name == "" || strings.EqualFold(name, "Unknown Region") {
		return code
	}
	return name
}

func formatCents(cents int64, currency, locale string) (string, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	locale = strings.TrimSpace(locale)
	if currency == "" {
		return "", errPDFCurrencyRequired
	}
	if locale == "" {
		return "", errPDFMoneyLocaleRequired
	}
	neg := cents < 0
	if neg {
		cents = -cents
	}
	dec, grp := separatorsForMoneyLocale(locale)
	whole := cents / 100
	frac := cents % 100
	wholeStr := groupDigits(whole, grp)
	amount := fmt.Sprintf("%s%s%02d", wholeStr, dec, frac)
	if neg {
		amount = "-" + amount
	}
	if sym := currencySymbol(currency); sym != "" {
		return sym + amount, nil
	}
	return amount + "\u00a0" + currency, nil
}

func currencySymbol(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "JPY":
		return "¥"
	default:
		return ""
	}
}

func separatorsForMoneyLocale(locale string) (decimal, group string) {
	tag := strings.ToLower(strings.ReplaceAll(locale, "_", "-"))
	primary := tag
	if i := strings.IndexByte(tag, '-'); i > 0 {
		primary = tag[:i]
	}
	switch primary {
	case "de", "es", "it", "nl", "pt", "pl", "ru", "tr", "cs", "sk", "hu", "ro", "uk", "bg", "hr", "sr", "sl", "id", "vi", "el":
		return ",", "."
	case "fr", "sv", "nb", "nn", "no", "da", "fi":
		return ",", "\u00a0"
	default:
		return ".", ","
	}
}

func groupDigits(n int64, sep string) string {
	if n < 1000 || sep == "" {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		b.WriteString(sep)
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// blackLetterheadImage returns the embedded black Trim mark with a soft mask
// so it composites onto gray header/footer bands without a white rectangle.
// opacity (0..1) scales the alpha channel (footer uses ~0.7 like the detail page).
func blackLetterheadImage(opacity float64) (*pdfImage, error) {
	if len(trimMarkBlackPNG) == 0 {
		return nil, errors.New("trim_mark_missing")
	}
	if opacity <= 0 {
		opacity = 1
	}
	if opacity > 1 {
		opacity = 1
	}
	src, _, err := image.Decode(bytes.NewReader(trimMarkBlackPNG))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return nil, errors.New("invalid_mark_dims")
	}
	rgb := make([]byte, w*h*3)
	mask := make([]byte, w*h)
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r16, g16, b16, a16 := src.At(x, y).RGBA()
			// RGBA() returns 16-bit premul-ish values in 0..65535
			a := uint8((uint32(a16) * uint32(opacity*255+0.5)) / 65535)
			rgb[i*3+0] = uint8(r16 >> 8)
			rgb[i*3+1] = uint8(g16 >> 8)
			rgb[i*3+2] = uint8(b16 >> 8)
			mask[i] = a
			i++
		}
	}
	return &pdfImage{Width: w, Height: h, RGB: rgb, Mask: mask}, nil
}

// FetchCompanyLogoJPEG downloads CompanyLogoURL and returns JPEG bytes suitable for PDF embed.
func FetchCompanyLogoJPEG(logoURL string) []byte {
	logoURL = strings.TrimSpace(logoURL)
	if logoURL == "" {
		return nil
	}
	if !strings.HasPrefix(logoURL, "https://") && !strings.HasPrefix(logoURL, "http://") {
		return nil
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(logoURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	const maxLogoBytes = 2 << 20 // 2 MiB
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxLogoBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > maxLogoBytes {
		return nil
	}
	lg, err := normalizeLogoImage(raw)
	if err != nil || lg == nil {
		return nil
	}
	if len(lg.JPEG) > 0 {
		return lg.JPEG
	}
	// Soft-mask PNG → flatten onto white only for the CompanyLogoJPEG cache
	// (legacy field). Live PDF prefers blackLetterheadImage soft-mask path.
	img := image.NewRGBA(image.Rect(0, 0, lg.Width, lg.Height))
	for y := 0; y < lg.Height; y++ {
		for x := 0; x < lg.Width; x++ {
			i := y*lg.Width + x
			a := lg.Mask[i]
			if a == 0 {
				img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
				continue
			}
			img.SetRGBA(x, y, color.RGBA{lg.RGB[i*3], lg.RGB[i*3+1], lg.RGB[i*3+2], 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil
	}
	return buf.Bytes()
}

func normalizeLogoImage(raw []byte) (*pdfImage, error) {
	if len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xd8 {
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		w, h := cfg.Width, cfg.Height
		if w < 1 || h < 1 {
			return nil, errors.New("invalid_jpeg_dims")
		}
		return &pdfImage{JPEG: raw, Width: w, Height: h}, nil
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return nil, errors.New("invalid_image_dims")
	}
	// Prefer soft-mask when source has alpha.
	if _, ok := img.(*image.RGBA); ok || img.ColorModel() == color.RGBAModel || img.ColorModel() == color.NRGBAModel {
		rgb := make([]byte, w*h*3)
		mask := make([]byte, w*h)
		i := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r16, g16, b16, a16 := img.At(x, y).RGBA()
				rgb[i*3+0] = uint8(r16 >> 8)
				rgb[i*3+1] = uint8(g16 >> 8)
				rgb[i*3+2] = uint8(b16 >> 8)
				mask[i] = uint8(a16 >> 8)
				i++
			}
		}
		return &pdfImage{Width: w, Height: h, RGB: rgb, Mask: mask}, nil
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 88}); err != nil {
		return nil, err
	}
	return &pdfImage{JPEG: buf.Bytes(), Width: w, Height: h}, nil
}

// Column left edges; numeric columns right-align to tableColX[i]+tableColW.
// Amount column right edge == pdfContentR (same as totals values).
const tableColW = 60.0

var tableColX = [5]float64{
	pdfMarginL,              // Product
	320,                     // Qty
	380,                     // Unit price
	460,                     // Tax rate
	pdfContentR - tableColW, // Amount
}

func writeReceiptPDF(ops []pdfOp) ([]byte, error) {
	var content bytes.Buffer
	y := 760.0
	curSize := 10
	curBold := false

	type imgSlot struct {
		name string
		img  *pdfImage
	}
	images := make([]imgSlot, 0, 2)
	registerImage := func(img *pdfImage) string {
		if img == nil {
			return ""
		}
		for _, s := range images {
			if s.img == img {
				return s.name
			}
			// Same soft-mask mark (header vs faded footer are different instances).
			if len(img.RGB) > 0 && len(s.img.RGB) > 0 &&
				s.img.Width == img.Width && s.img.Height == img.Height &&
				bytes.Equal(s.img.RGB, img.RGB) && bytes.Equal(s.img.Mask, img.Mask) {
				return s.name
			}
			if len(img.JPEG) > 0 && bytes.Equal(s.img.JPEG, img.JPEG) {
				return s.name
			}
		}
		name := fmt.Sprintf("Im%d", len(images)+1)
		images = append(images, imgSlot{name: name, img: img})
		return name
	}

	setFont := func(size int, bold bool) {
		if size < 1 {
			size = 10
		}
		if size == curSize && bold == curBold {
			return
		}
		name := "/F1"
		if bold {
			name = "/F2"
		}
		content.WriteString(fmt.Sprintf("%s %d Tf\n", name, size))
		curSize = size
		curBold = bold
	}

	writeTextAt := func(x, ty float64, size int, bold bool, s string, r, g, b float64) {
		s = truncateRunes(s, 120)
		setFont(size, bold)
		// Callers pass 0,0,0 for black; non-zero for gray/blue accents.
		colored := r != 0 || g != 0 || b != 0
		if colored {
			content.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", r, g, b))
		}
		content.WriteString(fmt.Sprintf("1 0 0 1 %.2f %.2f Tm\n", x, ty))
		content.WriteString("(")
		content.WriteString(pdfEscape(s))
		content.WriteString(") Tj\n")
		if colored {
			content.WriteString("0 0 0 rg\n")
		}
	}

	writeTextRight := func(rightX, ty float64, size int, bold bool, s string, r, g, b float64) {
		s = truncateRunes(s, 48)
		w := approxTextWidthFont(s, size, bold)
		writeTextAt(rightX-w, ty, size, bold, s, r, g, b)
	}

	writeTextCentered := func(ty float64, size int, bold bool, s string, r, g, b float64) {
		s = truncateRunes(s, 140)
		w := approxTextWidthFont(s, size, bold)
		x := (pdfPageW - w) / 2
		if x < pdfMarginL {
			x = pdfMarginL
		}
		writeTextAt(x, ty, size, bold, s, r, g, b)
	}

	// Labeled value = InlineDetail / payment: <span className="font-semibold">{l}:</span> {v}
	// AFM bold widths + pad so values never collide ("Payment methodvisa").
	writeLabeled := func(x, ty float64, size int, label, value string, r, g, b float64) {
		label = strings.TrimSpace(label)
		value = strings.TrimSpace(value)
		if label == "" || value == "" {
			return
		}
		prefix := label + ": "
		writeTextAt(x, ty, size, true, prefix, r, g, b)
		gap := approxTextWidthFont(prefix, size, true)*1.04 + 4.0
		writeTextAt(x+gap, ty, size, false, value, r, g, b)
	}

	drawImage := func(name string, img *pdfImage, x, topY, maxW, maxH float64) float64 {
		if name == "" || img == nil {
			return 0
		}
		dw, dh := fitLogo(float64(img.Width), float64(img.Height), maxW, maxH)
		logoY := topY - dh
		content.WriteString("ET\n")
		content.WriteString(fmt.Sprintf("q\n%.2f 0 0 %.2f %.2f %.2f cm\n/%s Do\nQ\n", dw, dh, x, logoY, name))
		content.WriteString("BT\n")
		setFont(curSize, curBold)
		return dh
	}

	content.WriteString("BT\n/F1 10 Tf\n")
	for _, op := range ops {
		switch op.kind {
		case "header_block":
			h := op.h
			if h < 90 {
				h = 100
			}
			bottom := y - h
			pageTop := 792.0
			content.WriteString("ET\n")
			content.WriteString(fmt.Sprintf(
				"%.3f %.3f %.3f rg\n%.2f %.2f %.2f %.2f re\nf\n0 0 0 rg\n",
				op.r, op.g, op.b,
				0.0, bottom, pdfPageW, pageTop-bottom,
			))
			logoH := 0.0
			if op.logo != nil {
				imgName := registerImage(op.logo)
				// TrimWordmark lg=28; company img sm:h-9 max-w-[140px]
				maxW, maxH := pdfMarkLG, pdfMarkLG
				if len(op.logo.JPEG) > 0 {
					maxW, maxH = 140, 36
				}
				dw, dh := fitLogo(float64(op.logo.Width), float64(op.logo.Height), maxW, maxH)
				logoH = dh
				logoY := y - 28 - dh // py-7
				content.WriteString(fmt.Sprintf("q\n%.2f 0 0 %.2f %.2f %.2f cm\n/%s Do\nQ\n", dw, dh, pdfContentR-dw, logoY, imgName))
			}
			content.WriteString("BT\n")
			setFont(pdfPtBody, false)
			ty := y - 28
			writeTextAt(pdfMarginL, ty, pdfPtTitle, true, op.text, 0, 0, 0)
			badge := strings.TrimSpace(op.left)
			if badge != "" {
				bw := approxTextWidthFont(strings.ToUpper(badge), pdfPtBadge, true) + 16 // px-2
				bx := pdfMarginL + approxTextWidthFont(op.text, pdfPtTitle, true) + 10   // gap-2.5
				content.WriteString("ET\n")
				content.WriteString(fmt.Sprintf("0.153 0.651 0.267 rg\n%.2f %.2f %.2f 14.00 re\nf\n", bx, ty-3, bw))
				content.WriteString("BT\n")
				setFont(pdfPtBadge, true)
				content.WriteString("1 1 1 rg\n")
				content.WriteString(fmt.Sprintf("1 0 0 1 %.2f %.2f Tm\n", bx+8, ty))
				content.WriteString("(")
				content.WriteString(pdfEscape(strings.ToUpper(badge)))
				content.WriteString(") Tj\n")
				content.WriteString("0 0 0 rg\n")
			}
			paidAt, totalAmt, sep := "", "", " - "
			brand, via := "", ""
			if len(op.lines) > 0 {
				brand = op.lines[0]
			}
			if len(op.lines) > 1 {
				via = op.lines[1]
			}
			if len(op.lines) > 2 {
				paidAt = op.lines[2]
			}
			if len(op.lines) > 3 {
				totalAmt = op.lines[3]
			}
			if len(op.lines) > 4 && strings.TrimSpace(op.lines[4]) != "" {
				sep = op.lines[4]
			}
			metaY := ty - 18 // space-y-1.5 under 24pt title
			if paidAt != "" || totalAmt != "" {
				mx := pdfMarginL
				if paidAt != "" {
					writeTextAt(mx, metaY, pdfPtMeta, false, paidAt, 0.45, 0.45, 0.45)
					mx += approxTextWidthFont(paidAt, pdfPtMeta, false)
					if totalAmt != "" {
						writeTextAt(mx, metaY, pdfPtMeta, false, sep, 0.45, 0.45, 0.45)
						mx += approxTextWidthFont(sep, pdfPtMeta, false)
					}
				}
				if totalAmt != "" {
					writeTextAt(mx, metaY, pdfPtMeta, true, totalAmt, 0.15, 0.15, 0.15)
				}
			} else if meta := strings.TrimSpace(op.right); meta != "" {
				writeTextAt(pdfMarginL, metaY, pdfPtMeta, false, meta, 0.45, 0.45, 0.45)
			}
			brandY := ty
			if op.logo != nil {
				brandY = y - 28 - logoH - 4 // gap-1
			}
			if brand != "" {
				writeTextRight(pdfContentR, brandY, pdfPtBrand, true, brand, 0, 0, 0)
			}
			if via != "" {
				writeTextRight(pdfContentR, brandY-15, pdfPtVia, false, via, 0.45, 0.45, 0.45)
			}
			y = bottom

		case "footer_block":
			// Measure footer (py-7 + copy + gap-3 + mark + gap-3 + locality + py-7)
			// so the band matches detail height and disclaimer is never truncated.
			footerParts := make([]string, 0, 8)
			for _, line := range op.lines {
				footerParts = append(footerParts, wrapWords(line, 72)...)
			}
			addrParts := wrapWords(strings.TrimSpace(op.left), 68)
			markH := 0.0
			if op.logo != nil {
				_, markH = fitLogo(float64(op.logo.Width), float64(op.logo.Height), pdfMarkSM, pdfMarkSM)
			}
			need := 28.0 + float64(len(footerParts))*16 + 12
			if op.logo != nil {
				need += markH + 12
			}
			need += float64(len(addrParts))*14 + 28
			if need < op.h {
				need = op.h
			}
			if need < 100 {
				need = 100
			}
			pageFloor := 18.0
			contentBottom := y - need
			if contentBottom < pageFloor {
				contentBottom = pageFloor
				need = y - contentBottom
			}
			content.WriteString("ET\n")
			content.WriteString(fmt.Sprintf(
				"%.3f %.3f %.3f rg\n%.2f %.2f %.2f %.2f re\nf\n0 0 0 rg\n",
				op.r, op.g, op.b,
				0.0, contentBottom, pdfPageW, need,
			))
			content.WriteString("BT\n")
			setFont(pdfPtFooter, false)
			cy := y - 28 // py-7
			for _, part := range footerParts {
				writeTextCentered(cy, pdfPtFooter, false, part, 0.40, 0.40, 0.40)
				cy -= 16 // leading-relaxed @ 12px
			}
			cy -= 12 // gap-3
			if op.logo != nil {
				imgName := registerImage(op.logo)
				dw, dh := fitLogo(float64(op.logo.Width), float64(op.logo.Height), pdfMarkSM, pdfMarkSM)
				content.WriteString("ET\n")
				content.WriteString(fmt.Sprintf("q\n%.2f 0 0 %.2f %.2f %.2f cm\n/%s Do\nQ\n", dw, dh, (pdfPageW-dw)/2, cy-dh, imgName))
				content.WriteString("BT\n")
				setFont(curSize, curBold)
				cy -= dh + 12 // gap-3
			}
			for _, part := range addrParts {
				writeTextCentered(cy, pdfPtLocality, false, part, 0.45, 0.45, 0.45)
				cy -= 14
			}
			y = contentBottom

		case "gap":
			h := op.h
			if h < 1 {
				h = 10
			}
			y -= h
		case "rule":
			content.WriteString("ET\n")
			content.WriteString(fmt.Sprintf("0.5 w\n0.910 0.910 0.910 RG\n%.2f %.2f m\n%.2f %.2f l\nS\n0 0 0 RG\n", pdfMarginL, y, pdfContentR, y))
			content.WriteString("BT\n")
			setFont(curSize, curBold)
			y -= 8
		case "two_col":
			// md:grid-cols-2 gap-10 (40px) → right column after mid + gap.
			const gap10 = 40.0
			colR := pdfMarginL + (pdfContentW+gap10)/2
			parseGap := func(s string) (float64, bool) {
				if !strings.HasPrefix(s, "#GAP:") {
					return 0, false
				}
				var n float64
				if _, err := fmt.Sscanf(s, "#GAP:%f", &n); err != nil || n < 1 {
					return 8, true
				}
				return n, true
			}
			for _, packed := range op.lines {
				parts := strings.SplitN(packed, "\x1f", 2)
				l, r := "", ""
				if len(parts) > 0 {
					l = parts[0]
				}
				if len(parts) > 1 {
					r = parts[1]
				}
				gL, okL := parseGap(l)
				gR, okR := parseGap(r)
				if okL || okR {
					g := gL
					if gR > g {
						g = gR
					}
					if g < 1 {
						g = 8
					}
					y -= g
					continue // spacer row (space-y-2 / pt-2 / pt-1)
				}
				lBold, rBold := false, false
				lVal, rVal := "", ""
				lHead, rHead := false, false
				if strings.HasPrefix(l, "#L:") {
					rest := strings.TrimPrefix(l, "#L:")
					kv := strings.SplitN(rest, "\x1e", 2)
					l = kv[0]
					if len(kv) > 1 {
						lVal = kv[1]
					}
					lBold = true
				} else if strings.HasPrefix(l, "#") {
					l = strings.TrimPrefix(l, "#")
					lBold = true
					lHead = true // h2 → print:text-zinc-900
				}
				if strings.HasPrefix(r, "#L:") {
					rest := strings.TrimPrefix(r, "#L:")
					kv := strings.SplitN(rest, "\x1e", 2)
					r = kv[0]
					if len(kv) > 1 {
						rVal = kv[1]
					}
					rBold = true
				} else if strings.HasPrefix(r, "#") {
					r = strings.TrimPrefix(r, "#")
					rBold = true
					rHead = true
				}
				if l == "" && r == "" && lVal == "" && rVal == "" {
					continue
				}
				// space-y-0.5 + leading-relaxed at text-[13px]
				y -= 17
				if y < 36 {
					continue
				}
				bodyR, bodyG, bodyB := 0.15, 0.15, 0.15 // print:text-zinc-800
				if l != "" {
					if lVal != "" {
						writeLabeled(pdfMarginL, y, pdfPtBody, l, lVal, bodyR, bodyG, bodyB)
					} else if lHead {
						writeTextAt(pdfMarginL, y, pdfPtBody, true, l, 0, 0, 0)
					} else {
						writeTextAt(pdfMarginL, y, pdfPtBody, lBold, l, bodyR, bodyG, bodyB)
					}
				}
				if r != "" {
					if rVal != "" {
						writeLabeled(colR, y, pdfPtBody, r, rVal, bodyR, bodyG, bodyB)
					} else if rHead {
						writeTextAt(colR, y, pdfPtBody, true, r, 0, 0, 0)
					} else {
						writeTextAt(colR, y, pdfPtBody, rBold, r, bodyR, bodyG, bodyB)
					}
				}
			}
		case "labeled":
			size := op.size
			if size < 1 {
				size = pdfPtBody
			}
			// InlineDetail: leading-relaxed + space-y-1
			y -= float64(size) + 6
			if y < 36 {
				continue
			}
			writeLabeled(pdfMarginL, y, size, op.left, op.right, 0, 0, 0)
		case "right_pair":
			size := op.size
			if size < 1 {
				size = pdfPtBody
			}
			// max-w-[240px] + py-2 + border-b (same hairline every row)
			totalsL := pdfContentR - 240
			y -= 7 // py-2 top
			y -= float64(size)
			if y < 36 {
				continue
			}
			writeTextAt(totalsL, y, size, op.bold, op.left, op.r, op.g, op.b)
			writeTextRight(pdfContentR, y, size, op.bold, op.right, op.r, op.g, op.b)
			y -= 7 // py-2 bottom
			content.WriteString("ET\n")
			content.WriteString(fmt.Sprintf("0.4 w\n0.910 0.910 0.910 RG\n%.2f %.2f m\n%.2f %.2f l\nS\n0 0 0 RG\n", totalsL, y, pdfContentR, y))
			content.WriteString("BT\n")
			setFont(curSize, curBold)
			y -= 2
		case "table_header":
			size := op.size
			if size < 1 {
				size = pdfPtTableH
			}
			y -= float64(size)
			if y < 48 {
				continue
			}
			for i, cell := range op.cells {
				if strings.TrimSpace(cell) == "" {
					continue
				}
				if i == 0 {
					writeTextAt(tableColX[i], y, size, true, cell, 0, 0, 0)
				} else {
					writeTextRight(tableColX[i]+tableColW, y, size, true, cell, 0, 0, 0)
				}
			}
			y -= 10 // pb-2.5
			content.WriteString("ET\n")
			content.WriteString(fmt.Sprintf("0.5 w\n0.910 0.910 0.910 RG\n%.2f %.2f m\n%.2f %.2f l\nS\n0 0 0 RG\n", pdfMarginL, y, pdfContentR, y))
			content.WriteString("BT\n")
			setFont(curSize, curBold)
			y -= 2
		case "tax_table":
			// max-w-[240px]: th text-[12px], body text-[13px]
			th := op.size
			if th < 1 {
				th = pdfPtTableH
			}
			body := pdfPtBody
			taxPctH, taxAmtH := op.cells[0], op.cells[1]
			taxPctV, taxAmtV := op.cells[2], op.cells[3]
			taxTotalL := op.cells[4]
			taxTotalV := op.right
			taxRight := pdfMarginL + 240
			y -= float64(th)
			writeTextAt(pdfMarginL, y, th, true, taxPctH, 0, 0, 0)
			writeTextRight(taxRight, y, th, true, taxAmtH, 0, 0, 0)
			y -= 8 // pb-2
			content.WriteString("ET\n")
			content.WriteString(fmt.Sprintf("0.4 w\n0.910 0.910 0.910 RG\n%.2f %.2f m\n%.2f %.2f l\nS\n0 0 0 RG\n", pdfMarginL, y, taxRight, y))
			content.WriteString("BT\n")
			setFont(curSize, curBold)
			y -= 8 // py-2 top
			y -= float64(body)
			writeTextAt(pdfMarginL, y, body, false, taxPctV, 0.15, 0.15, 0.15) // zinc-800
			writeTextRight(taxRight, y, body, false, taxAmtV, 0.15, 0.15, 0.15)
			y -= 8
			content.WriteString("ET\n")
			content.WriteString(fmt.Sprintf("0.4 w\n0.910 0.910 0.910 RG\n%.2f %.2f m\n%.2f %.2f l\nS\n0 0 0 RG\n", pdfMarginL, y, taxRight, y))
			content.WriteString("BT\n")
			setFont(curSize, curBold)
			if strings.TrimSpace(taxTotalL) != "" {
				y -= 10 // pt-2.5
				y -= float64(body)
				writeTextAt(pdfMarginL, y, body, true, taxTotalL, 0, 0, 0)
				writeTextRight(taxRight, y, body, true, taxTotalV, 0, 0, 0)
			}
			y -= 4
		case "product_row":
			size := op.size
			if size < 1 {
				size = pdfPtBody
			}
			stack := op.lines
			if len(stack) == 0 {
				stack = []string{op.cells[0]}
			}
			// py-3.5; sublines text-[12px] mt-0.5
			y -= 12
			y -= float64(size)
			topY := y
			for i, line := range stack {
				if y < 48 {
					break
				}
				if i == 0 {
					writeTextAt(tableColX[0], y, size, true, line, 0, 0, 0)
				} else if i == 1 {
					writeTextAt(tableColX[0], y, pdfPtTableH, false, line, 0.45, 0.45, 0.45)
				} else {
					writeTextAt(tableColX[0], y, pdfPtTableH, false, line, 0.012, 0.412, 0.631) // sky-700
				}
				if i+1 < len(stack) {
					y -= float64(pdfPtTableH) + 2 // mt-0.5
				}
			}
			for i := 1; i < 5; i++ {
				cell := op.cells[i]
				if strings.TrimSpace(cell) == "" {
					continue
				}
				r, g, b := 0.35, 0.35, 0.35 // zinc-700
				bold := false
				if i == 4 {
					r, g, b = 0, 0, 0
					bold = true
				}
				writeTextRight(tableColX[i]+tableColW, topY, size, bold, cell, r, g, b)
			}
			y -= 12 // py-3.5 bottom
			content.WriteString("ET\n")
			content.WriteString(fmt.Sprintf("0.4 w\n0.910 0.910 0.910 RG\n%.2f %.2f m\n%.2f %.2f l\nS\n0 0 0 RG\n", pdfMarginL, y, pdfContentR, y))
			content.WriteString("BT\n")
			setFont(curSize, curBold)
			y -= 2
		default: // text
			size := op.size
			if size < 1 {
				size = pdfPtBody
			}
			y -= float64(size + 4)
			if y < 36 {
				continue
			}
			x := op.x
			if x < 1 {
				x = pdfMarginL
			}
			writeTextAt(x, y, size, op.bold, op.text, op.r, op.g, op.b)
		}
	}
	_ = drawImage
	content.WriteString("ET\n")
	stream := content.Bytes()

	var buf bytes.Buffer
	write := func(s string) { buf.WriteString(s) }
	offsets := make([]int, 0, 20)
	offsets = append(offsets, 0) // unused index 0

	write("%PDF-1.4\n")
	offsets = append(offsets, buf.Len())
	write("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	offsets = append(offsets, buf.Len())
	write("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	offsets = append(offsets, buf.Len())

	// Pre-assign object numbers for images + soft masks after fonts (5,6).
	// Page resources reference them; objects written after content stream.
	nextObj := 7
	type imgObjNums struct {
		img  int
		mask int // 0 if none
		slot imgSlot
	}
	imgObjs := make([]imgObjNums, 0, len(images))
	xobjects := ""
	for i, slot := range images {
		io := imgObjNums{img: nextObj, slot: slot}
		nextObj++
		if len(slot.img.RGB) > 0 && len(slot.img.Mask) > 0 {
			io.mask = nextObj
			nextObj++
		}
		imgObjs = append(imgObjs, io)
		if i > 0 {
			xobjects += " "
		}
		xobjects += fmt.Sprintf("/%s %d 0 R", slot.name, io.img)
	}

	var resources string
	if xobjects != "" {
		resources = fmt.Sprintf("<< /Font << /F1 5 0 R /F2 6 0 R >> /XObject << %s >> >>", xobjects)
	} else {
		resources = "<< /Font << /F1 5 0 R /F2 6 0 R >> >>"
	}
	write(fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources %s >>\nendobj\n", resources))

	offsets = append(offsets, buf.Len())
	write(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n", len(stream)))
	buf.Write(stream)
	write("\nendstream\nendobj\n")

	offsets = append(offsets, buf.Len())
	write("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")
	offsets = append(offsets, buf.Len())
	write("6 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>\nendobj\n")

	for _, io := range imgObjs {
		img := io.slot.img
		offsets = append(offsets, buf.Len())
		if len(img.JPEG) > 0 {
			write(fmt.Sprintf(
				"%d 0 obj\n<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n",
				io.img, img.Width, img.Height, len(img.JPEG),
			))
			buf.Write(img.JPEG)
			write("\nendstream\nendobj\n")
			continue
		}
		compressed := zlibCompress(img.RGB)
		smaskRef := ""
		if io.mask > 0 {
			smaskRef = fmt.Sprintf(" /SMask %d 0 R", io.mask)
		}
		write(fmt.Sprintf(
			"%d 0 obj\n<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode%s /Length %d >>\nstream\n",
			io.img, img.Width, img.Height, smaskRef, len(compressed),
		))
		buf.Write(compressed)
		write("\nendstream\nendobj\n")
		if io.mask > 0 {
			offsets = append(offsets, buf.Len())
			mcomp := zlibCompress(img.Mask)
			write(fmt.Sprintf(
				"%d 0 obj\n<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n",
				io.mask, img.Width, img.Height, len(mcomp),
			))
			buf.Write(mcomp)
			write("\nendstream\nendobj\n")
		}
	}

	xrefStart := buf.Len()
	write(fmt.Sprintf("xref\n0 %d\n", len(offsets)))
	write("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		write(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	write(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xrefStart))
	return buf.Bytes(), nil
}

func zlibCompress(b []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

func approxTextWidth(s string, size int) float64 {
	return approxTextWidthFont(s, size, false)
}

// helveticaWidths / helveticaBoldWidths are Adobe AFM WinAnsi advances (units/1000).
// Missing glyphs fall back to average width. Accurate widths prevent labeled
// collisions like "Payment methodvisa" when bold labels sit beside regular values.
var helveticaWidths = [256]int{
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556,
	1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556,
	333, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556,
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 333, 556, 556, 167, 556, 556, 556, 556, 191, 333, 556, 333, 333, 500, 500,
	250, 556, 556, 556, 278, 250, 537, 350, 222, 333, 333, 556, 1000, 1000, 250, 611,
	250, 333, 333, 333, 333, 333, 333, 333, 333, 250, 333, 333, 250, 333, 333, 333,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
}

var helveticaBoldWidths = [256]int{
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	278, 333, 474, 556, 556, 889, 722, 238, 333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 333, 333, 584, 584, 584, 611,
	975, 722, 722, 722, 722, 667, 611, 778, 722, 278, 556, 722, 611, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 333, 278, 333, 584, 556,
	333, 556, 611, 556, 611, 556, 333, 611, 611, 278, 278, 556, 278, 889, 611, 611,
	611, 611, 389, 556, 333, 611, 556, 778, 556, 556, 500, 389, 280, 389, 584, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 333, 556, 556, 167, 556, 556, 556, 556, 238, 500, 556, 333, 333, 611, 611,
	250, 556, 556, 556, 278, 250, 556, 350, 278, 500, 500, 556, 1000, 1000, 250, 611,
	250, 333, 333, 333, 333, 333, 333, 333, 333, 250, 333, 333, 250, 333, 333, 333,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
	250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250, 250,
}

// approxTextWidthFont returns Helvetica / Helvetica-Bold advance using AFM widths.
func approxTextWidthFont(s string, size int, bold bool) float64 {
	if size < 1 {
		size = 10
	}
	table := helveticaWidths
	if bold {
		table = helveticaBoldWidths
	}
	sum := 0
	for _, r := range s {
		if r < 0 || r > 255 {
			sum += 600
			continue
		}
		w := table[r]
		if w <= 0 {
			w = 556
		}
		sum += w
	}
	return float64(sum) * float64(size) / 1000.0
}

func fitLogo(srcW, srcH, maxW, maxH float64) (float64, float64) {
	if srcW < 1 {
		srcW = 1
	}
	if srcH < 1 {
		srcH = 1
	}
	scale := maxW / srcW
	if srcH*scale > maxH {
		scale = maxH / srcH
	}
	return srcW * scale, srcH * scale
}

func truncateRunes(s string, max int) string {
	if max < 1 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "."
}

func wrapWords(s string, maxChars int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if maxChars < 8 {
		maxChars = 8
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
			continue
		}
		if cur.Len()+1+len(w) > maxChars {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(w)
			continue
		}
		cur.WriteByte(' ')
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

func pdfEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	var b strings.Builder
	for _, r := range s {
		if r > 255 {
			b.WriteRune('?')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
