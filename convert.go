package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var aliases = map[string]string{
	"usd": "USD", "dollar": "USD", "dollor": "USD", "$": "USD", "doll": "USD",
	"rub": "RUB", "rubl": "RUB", "rublь": "RUB", "rubel": "RUB", "rubl'": "RUB",
	"eur": "EUR", "yevro": "EUR", "evro": "EUR", "euro": "EUR", "€": "EUR",
	"uzs": "UZS", "som": "UZS", "so'm": "UZS", "sum": "UZS", "so‘m": "UZS", "so’m": "UZS",
}

var multipliers = map[string]float64{
	"k": 1e3, "ming": 1e3, "ming.": 1e3,
	"mln": 1e6, "million": 1e6, "mln.": 1e6,
	"mlrd": 1e9, "milliard": 1e9,
}

var queryRe = regexp.MustCompile(`^\s*([0-9]+(?:[.,][0-9]+)?)\s*(.*)$`)

// Query - foydalanuvchi yozgan "500 usd" kabi so'rov.
type Query struct {
	Amount   float64
	Currency string
}

// ParseQuery "500 usd", "2 mln so'm", "100$" kabi matnlarni tushunadi.
func ParseQuery(text string) (Query, bool) {
	m := queryRe.FindStringSubmatch(strings.ToLower(strings.TrimSpace(text)))
	if m == nil {
		return Query{}, false
	}
	amount, err := strconv.ParseFloat(strings.Replace(m[1], ",", ".", 1), 64)
	if err != nil {
		return Query{}, false
	}
	words := strings.Fields(m[2])
	if len(words) > 0 {
		if mult, ok := multipliers[words[0]]; ok {
			amount *= mult
			words = words[1:]
		}
	}
	if len(words) == 0 {
		return Query{}, false
	}
	cur, ok := aliases[words[0]]
	if !ok {
		return Query{}, false
	}
	return Query{Amount: amount, Currency: cur}, true
}

// Convert so'rovni javob matniga aylantiradi.
func Convert(q Query, rates map[string]Rate) string {
	if q.Currency == "UZS" {
		var b strings.Builder
		fmt.Fprintf(&b, "💵 %s so'm =\n", formatNum(q.Amount, 0))
		for _, c := range []string{"USD", "EUR", "RUB"} {
			if r, ok := rates[c]; ok {
				fmt.Fprintf(&b, "• %s %s\n", formatNum(q.Amount/r.UZS, 2), c)
			}
		}
		return b.String()
	}
	r, ok := rates[q.Currency]
	if !ok {
		return "Bu valyuta kursi topilmadi."
	}
	return fmt.Sprintf("💵 %s %s = %s so'm\n(CBU kursi: 1 %s = %s so'm)",
		formatNum(q.Amount, 2), q.Currency, formatNum(q.Amount*r.UZS, 0), q.Currency, formatNum(r.UZS, 2))
}

// RatesText /kurs buyrug'i uchun matn.
func RatesText(rates map[string]Rate) string {
	var b strings.Builder
	date := ""
	for _, c := range []string{"USD", "EUR", "RUB"} {
		r, ok := rates[c]
		if !ok {
			continue
		}
		date = r.Date
		arrow := "▪️"
		if r.Diff > 0 {
			arrow = "🔺"
		} else if r.Diff < 0 {
			arrow = "🔻"
		}
		fmt.Fprintf(&b, "%s 1 %s = %s so'm (%+.2f)\n", arrow, c, formatNum(r.UZS, 2), r.Diff)
	}
	return fmt.Sprintf("📊 Markaziy bank kursi (%s)\n\n%s", date, b.String())
}

// formatNum: 1234567.891 -> "1 234 567.89"
func formatNum(f float64, decimals int) string {
	s := strconv.FormatFloat(math.Abs(f), 'f', decimals, 64)
	intPart, frac := s, ""
	if i := strings.Index(s, "."); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	var out []byte
	for i, ch := range []byte(intPart) {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			out = append(out, ' ')
		}
		out = append(out, ch)
	}
	res := string(out) + frac
	if f < 0 {
		res = "-" + res
	}
	return res
}

// ParseAmount foydalanuvchidan kelgan oddiy summani o'qiydi.
// Masalan: 100, 500.5, 1 000, 1,000.50
func ParseAmount(text string) (float64, bool) {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " ", "")
	text = strings.ReplaceAll(text, "_", "")

	// O'zbek foydalanuvchilari verguldan ham foydalanishi mumkin.
	text = strings.Replace(text, ",", ".", 1)

	if text == "" {
		return 0, false
	}

	amount, err := strconv.ParseFloat(text, 64)
	if err != nil || amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, false
	}

	return amount, true
}

type ConversionQuery struct {
	Amount float64
	From   string
	To     string
}

// ParseConversionQuery quyidagilarni tushunadi:
//
// 100 USD -> UZS
// 100 USD → UZS
// 500 eur to uzs
// 100 usd uzs
func ParseConversionQuery(text string) (ConversionQuery, bool) {
	text = strings.ToLower(strings.TrimSpace(text))

	text = strings.ReplaceAll(text, "→", "->")
	text = strings.ReplaceAll(text, "=>", "->")

	// "100 usd to uzs"
	text = strings.ReplaceAll(text, " to ", " -> ")

	parts := strings.Split(text, "->")

	if len(parts) != 2 {
		return ConversionQuery{}, false
	}

	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	// Chap tomondan summa + source currency ajratamiz.
	m := queryRe.FindStringSubmatch(left)
	if m == nil {
		return ConversionQuery{}, false
	}

	amount, err := strconv.ParseFloat(
		strings.Replace(m[1], ",", ".", 1),
		64,
	)
	if err != nil || amount < 0 {
		return ConversionQuery{}, false
	}

	fromWords := strings.Fields(m[2])
	toWords := strings.Fields(right)

	if len(fromWords) == 0 || len(toWords) == 0 {
		return ConversionQuery{}, false
	}

	from, ok := aliases[fromWords[0]]
	if !ok {
		return ConversionQuery{}, false
	}

	to, ok := aliases[toWords[0]]
	if !ok {
		return ConversionQuery{}, false
	}

	return ConversionQuery{
		Amount: amount,
		From:   from,
		To:     to,
	}, true
}

// ConvertPair ikki valyuta o'rtasidagi konvertatsiyani hisoblaydi.
func ConvertPair(
	amount float64,
	from string,
	to string,
	rates map[string]Rate,
) string {
	if amount < 0 {
		return "❌ Summa manfiy bo‘lishi mumkin emas."
	}

	if from == to {
		return fmt.Sprintf(
			"🔄 Konvertatsiya\n\n%s %s = %s %s",
			formatNum(amount, 2),
			from,
			formatNum(amount, 2),
			to,
		)
	}

	// Avval source valyutani UZS ga o'tkazamiz.
	var amountUZS float64

	if from == "UZS" {
		amountUZS = amount
	} else {
		r, ok := rates[from]
		if !ok {
			return "❌ " + from + " kursi topilmadi."
		}

		amountUZS = amount * r.UZS
	}

	// UZS dan target valyutaga o'tkazamiz.
	var result float64

	if to == "UZS" {
		result = amountUZS
	} else {
		r, ok := rates[to]
		if !ok {
			return "❌ " + to + " kursi topilmadi."
		}

		if r.UZS == 0 {
			return "❌ " + to + " kursi noto‘g‘ri."
		}

		result = amountUZS / r.UZS
	}

	return fmt.Sprintf(
		"🔄 Konvertatsiya\n\n"+
			"💰 %s %s\n"+
			"⬇️\n"+
			"💵 %s %s",
		formatNum(amount, 2),
		from,
		formatNum(result, 2),
		to,
	)
}
