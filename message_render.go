package intl

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/number"
)

// render evaluates only the selected alternatives. An error discards partial output.
func (m *compiledMessage) render(locale *dictionary, values Values) (string, error) {
	if single := m.single; single != nil {
		value, ok := values[single.key]
		if !ok || value == nil {
			return "", fmt.Errorf("missing value for %q", single.key)
		}
		text := messageString(value)
		if single.prefix == "" && single.suffix == "" {
			return text, nil
		}
		return single.prefix + text + single.suffix, nil
	}
	if len(m.code) == 0 {
		return m.raw, nil
	}
	if len(m.code) == 1 && m.code[0].kind == opText {
		return m.code[0].text, nil
	}
	var b strings.Builder
	r := messageRender{code: m.code, locale: locale, values: values}
	if err := r.body(&b, 0, len(m.code), nil); err != nil {
		return "", err
	}
	return b.String(), nil
}

// messageRender shares immutable input across the active nested message bodies.
type messageRender struct {
	// code is the program shared by all translations of this message.
	code []instruction
	// locale holds the selected translation and its prebound formatters.
	locale *dictionary
	// values belong to the caller and are only read during rendering.
	values Values
}

// body traverses one contiguous range of instructions.
func (r *messageRender) body(b *strings.Builder, start, end int, pound any) error {
	for i := start; i < end; {
		op := r.code[i]
		i++
		switch op.kind {
		case opText:
			b.WriteString(op.text)
		case opPound:
			b.WriteString(localizedNumber(r.locale, pound, opNumber))
		default:
			value, ok := r.values[op.text]
			if !ok || value == nil {
				return fmt.Errorf("missing value for %q", op.text)
			}
			switch op.kind {
			case opTag:
				fn, ok := value.(func(string) string)
				if !ok || fn == nil {
					return fmt.Errorf("tag %q requires func(string) string", op.text)
				}
				var child strings.Builder
				if err := r.body(&child, i, op.end, pound); err != nil {
					return err
				}
				b.WriteString(fn(child.String()))
				i = op.end
			case opSelect, opPlural, opOrdinal:
				branch, adjusted, err := r.choose(i, op, value)
				if err != nil {
					return fmt.Errorf("argument %q: %w", op.text, err)
				}
				if err := r.body(b, branch+1, r.code[branch].end, adjusted); err != nil {
					return err
				}
				i = op.end
			default:
				s, err := r.format(op, value)
				if err != nil {
					return err
				}
				b.WriteString(s)
			}
		}
	}
	return nil
}

// format renders a single argument using the formatter chosen at compile time.
func (r *messageRender) format(op instruction, value any) (string, error) {
	switch op.kind {
	case opValue:
		return messageString(value), nil
	case opTime, opDate:
		t, ok := value.(time.Time)
		if !ok {
			return "", fmt.Errorf("argument %q requires time.Time", op.text)
		}
		kind := "date"
		if op.kind == opTime {
			kind = "time"
		}
		return formatDateTime(r.locale.dates, t, kind, dateStyles[op.end]), nil
	default:
		if _, err := numberText(value); err != nil {
			return "", fmt.Errorf("argument %q: %w", op.text, err)
		}
		return localizedNumber(r.locale, value, op.kind), nil
	}
}

// messageString returns string arguments directly and accepts Go enum and Stringer values.
func messageString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

// negativeMagnitude holds the exact result of subtracting an offset below math.MinInt64.
type negativeMagnitude uint64

// localizedNumber delegates grouping, digits, percent signs, and rounding to the locale.
func localizedNumber(locale *dictionary, value any, kind messageOp) string {
	if magnitude, ok := value.(negativeMagnitude); ok {
		// Obtain the locale's minus sign, bidi marks, and digit placement from -1.
		negative := localizedNumber(locale, -1, kind)
		positive := localizedNumber(locale, 1, kind)
		return strings.Replace(negative, positive, localizedNumber(locale, uint64(magnitude), kind), 1)
	}
	// Ungrouped ASCII integers below 1000 have identical output in these locales.
	if kind == opNumber && (locale.tag == language.English || locale.tag == language.AmericanEnglish) {
		if n, ok := value.(int); ok && n > -1000 && n < 1000 {
			return strconv.Itoa(n)
		}
	}
	var f number.Formatter
	switch kind {
	case opInteger:
		f = number.Decimal(value, number.MaxFractionDigits(0))
	case opPercent:
		f = number.Percent(value)
	default:
		f = number.Decimal(value)
	}
	return locale.printer.Sprint(f)
}

// choose gives exact matches precedence over plural categories, then falls back to other.
func (r *messageRender) choose(start int, op instruction, value any) (int, any, error) {
	var selector, original string
	var adjusted any
	if op.kind == opSelect {
		selector = messageString(value)
	} else {
		n, err := numberText(value)
		if err != nil {
			return 0, nil, err
		}
		original = n
		adjusted = value
		if offset := r.code[start]; offset.kind == opOffset {
			start++
			n, adjusted = subtractOffset(n, offset.text)
		}
		rules := plural.Cardinal
		if op.kind == opOrdinal {
			rules = plural.Ordinal
		}
		selector = pluralCategory(r.locale.tag, rules, n)
	}
	var choice, fallback int
	for i := start; i < op.end; i = r.code[i].end {
		label := r.code[i].text
		switch {
		case original != "" && label[0] == '=' && label[1:] == original:
			return i, adjusted, nil
		case label == selector:
			choice = i
		case label == "other":
			fallback = i
		}
	}
	if choice != 0 {
		return choice, adjusted, nil
	}
	return fallback, adjusted, nil
}

// numberText preserves the full integer domain for exact selectors and CLDR operands.
func numberText(value any) (string, error) {
	switch n := value.(type) {
	case int:
		return strconv.Itoa(n), nil
	case int8:
		return strconv.FormatInt(int64(n), 10), nil
	case int16:
		return strconv.FormatInt(int64(n), 10), nil
	case int32:
		return strconv.FormatInt(int64(n), 10), nil
	case int64:
		return strconv.FormatInt(n, 10), nil
	case uint:
		return strconv.FormatUint(uint64(n), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(n), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(n), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(n), 10), nil
	case uint64:
		return strconv.FormatUint(n, 10), nil
	case float32:
		return floatText(float64(n), 32)
	case float64:
		return floatText(n, 64)
	default:
		return "", fmt.Errorf("expected a number, got %T", value)
	}
}

// floatText converts finite floats to a short decimal spelling with no exponent.
func floatText(n float64, bits int) (string, error) {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return "", fmt.Errorf("expected a finite number")
	}
	if n == 0 {
		return "0", nil
	}
	return strconv.FormatFloat(n, 'f', -1, bits), nil
}

// subtractOffset uses exact decimal arithmetic on the uncommon offset path.
func subtractOffset(n, offset string) (string, any) {
	value, _ := new(big.Rat).SetString(n)
	amount, _ := new(big.Rat).SetString(offset)
	_, frac, _ := strings.Cut(n, ".")
	adjusted := value.Sub(value, amount).FloatString(len(frac))
	if i, err := strconv.ParseInt(adjusted, 10, 64); err == nil {
		return adjusted, i
	}
	if i, err := strconv.ParseUint(adjusted, 10, 64); err == nil {
		return adjusted, i
	}
	if magnitude, negative := strings.CutPrefix(adjusted, "-"); negative {
		if i, err := strconv.ParseUint(magnitude, 10, 64); err == nil {
			return adjusted, negativeMagnitude(i)
		}
	}
	f, _ := strconv.ParseFloat(adjusted, 64)
	return adjusted, f
}

// pluralCategory derives the operands for the selected locale from an exact decimal.
func pluralCategory(tag language.Tag, rules *plural.Rules, s string) string {
	integer, fraction, _ := strings.Cut(strings.TrimPrefix(s, "-"), ".")
	i := pluralOperand(integer)
	f := pluralOperand(fraction)
	form := rules.MatchPlural(tag, i, len(fraction), len(fraction), f, f)
	switch form {
	case plural.Zero:
		return "zero"
	case plural.One:
		return "one"
	case plural.Two:
		return "two"
	case plural.Few:
		return "few"
	case plural.Many:
		return "many"
	default:
		return "other"
	}
}

// pluralOperand preserves the last seven digits and keeps large values away from exact rules.
func pluralOperand(s string) int {
	s = strings.TrimLeft(s, "0")
	var n int
	if len(s) > 7 {
		n = 1
		s = s[len(s)-7:]
	}
	for i := range len(s) {
		n = 10*n + int(s[i]-'0')
	}
	return n
}
