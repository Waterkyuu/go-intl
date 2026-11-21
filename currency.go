package intl

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/waterkyuu/go-intl/internal/cldrtext"
	"golang.org/x/text/currency"
)

// Currency formats major units using ISO 4217 precision, localized digits and
// CLDR standard monetary patterns. Decimal ties round away from zero.
func (l Locale) Currency(amount float64, code string) (string, error) {
	unit, err := currency.ParseISO(code)
	if err != nil {
		return "", fmt.Errorf("currency %q: %w", code, err)
	}
	d := &l.catalog.locales[l.chain[0]]
	symbol := d.printer.Sprint(currency.Symbol(unit))
	form := cldrtext.Currency(d.tag)
	sign := 0
	if math.Signbit(amount) && !math.IsNaN(amount) {
		sign = 1
	}
	pattern := form.Patterns[sign]
	if form.Patterns[sign+2] != pattern && currencyTouchesLetter(pattern, symbol) {
		pattern = form.Patterns[sign+2]
	}
	before, after, _ := strings.Cut(pattern, "#")
	var out strings.Builder
	out.WriteString(strings.Replace(before, "¤", symbol, 1))
	switch {
	case math.IsNaN(amount):
		out.WriteString(form.NaN)
	case math.IsInf(amount, 0):
		out.WriteString(form.Infinity)
	default:
		precision, _ := currency.Standard.Rounding(unit)
		writeCurrencyNumber(&out, form, math.Abs(amount), precision)
	}
	out.WriteString(strings.Replace(after, "¤", symbol, 1))
	return out.String(), nil
}

// currencyTouchesLetter identifies the symbol edge that faces the monetary amount.
func currencyTouchesLetter(pattern, symbol string) bool {
	if strings.Index(pattern, "¤") < strings.IndexByte(pattern, '#') {
		last, _ := utf8.DecodeLastRuneInString(strings.TrimRightFunc(symbol, unicode.IsMark))
		return unicode.IsLetter(last)
	}
	first, _ := utf8.DecodeRuneInString(symbol)
	return unicode.IsLetter(first)
}

// writeCurrencyNumber rounds a nonnegative finite magnitude and emits its localized digits.
func writeCurrencyNumber(out *strings.Builder, form *cldrtext.CurrencyFormat, amount float64, precision int) {
	var scratch [64]byte
	// Reserve one leading zero for a carry such as 9.995 -> 10.00.
	digits := strconv.AppendFloat(scratch[:1], amount, 'f', -1, 64)
	digits[0] = '0'
	point := bytes.IndexByte(digits, '.')
	if point < 0 {
		point = len(digits)
	} else {
		digits = append(digits[:point], digits[point+1:]...)
	}
	end := point + precision
	if end < len(digits) && digits[end] >= '5' {
		i := end - 1
		for digits[i] == '9' {
			digits[i] = '0'
			i--
		}
		digits[i]++
	}
	for len(digits) < end {
		digits = append(digits, '0')
	}
	start := 0
	if digits[0] == '0' {
		start = 1
	}
	group := point-start >= form.Primary+form.Minimum
	for i := start; i < end; i++ {
		remaining := point - i
		switch {
		case i == point:
			out.WriteString(form.Decimal)
		case group && i > start && remaining >= form.Primary && (remaining-form.Primary)%form.Secondary == 0:
			out.WriteString(form.Group)
		}
		out.WriteRune(form.Digits[digits[i]-'0'])
	}
}
