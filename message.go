package intl

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Values supplies named ICU arguments and rich-text callbacks for one translation.
// A caller sharing the map across goroutines must keep it read-only.
type Values map[string]any

// compiledMessage preserves the original pattern and its read-only rendering representation.
type compiledMessage struct {
	// raw preserves the catalog author's original ICU pattern.
	raw string
	// code stores instructions and nested alternatives in source order.
	code []instruction
	// plain holds the ordered substitutions for messages containing only text and ordinary arguments.
	plain *interpolation
}

// interpolation stores each substitution in source order followed by the trailing literal.
type interpolation struct {
	// args stores one entry per argument occurrence, in source order.
	args []interpolationArg
	// suffix is the decoded text after the last argument.
	suffix string
}

// interpolationArg pairs a named argument with the literal immediately before it.
type interpolationArg struct {
	// prefix contains the decoded text since the preceding argument.
	prefix string
	// key identifies the value to read at this position.
	key string
}

// instruction contains a literal, an argument name, or an alternative selector.
type instruction struct {
	// text references the source unless escaping required a new string.
	text string
	// end points just past the body of a branch, tag, or alternative.
	end int
	// kind selects the operation performed by the renderer.
	kind messageOp
}

// messageOp identifies an instruction in a compiled message.
type messageOp uint8

const (
	// opText emits a literal.
	opText messageOp = iota
	// opValue interpolates a named value.
	opValue
	// opSelect branches on an enum or boolean value.
	opSelect
	// opPlural selects a cardinal form.
	opPlural
	// opOrdinal selects an ordinal form.
	opOrdinal
	// opCase records a selector and the end of its body.
	opCase
	// opOffset holds the amount subtracted before plural categorization.
	opOffset
	// opPound writes the innermost plural's adjusted number.
	opPound
	// opTag applies a rich-text callback to the rendered body.
	opTag
	// opNumber formats a localized decimal.
	opNumber
	// opInteger rounds to a whole number.
	opInteger
	// opPercent formats a fraction as a percentage.
	opPercent
	// opDate formats a calendar date.
	opDate
	// opTime formats a local clock time.
	opTime
	// maxMessageDepth bounds stack use during compilation and rendering.
	maxMessageDepth = 64
)

// messageParser emits instructions while advancing once through the source.
type messageParser struct {
	// source contains the untranslated pattern.
	source string
	// pos is the next byte to consume.
	pos int
	// code holds the growing instruction stream.
	code []instruction
}

// compileMessage validates the pattern before it is made available to readers.
func compileMessage(source string) (compiledMessage, error) {
	if !strings.ContainsAny(source, "{}'<") {
		return compiledMessage{raw: source}, nil
	}
	p := messageParser{source: source}
	if err := p.body(0, false, ""); err != nil {
		return compiledMessage{}, fmt.Errorf("message at byte %d: %w", p.pos, err)
	}
	var count int
	for _, op := range p.code {
		switch op.kind {
		case opValue:
			count++
		case opText:
		default:
			return compiledMessage{raw: source, code: p.code}, nil
		}
	}
	if count > 0 {
		return compiledMessage{raw: source, plain: compileInterpolation(p.code, count)}, nil
	}
	if text := literalText(p.code); text != source {
		return compiledMessage{raw: source, code: []instruction{{kind: opText, text: text}}}, nil
	}
	return compiledMessage{raw: source}, nil
}

// compileInterpolation associates decoded literals with their following argument and preserves their order.
func compileInterpolation(code []instruction, count int) *interpolation {
	args := make([]interpolationArg, 0, count)
	start := 0
	for i, op := range code {
		if op.kind != opValue {
			continue
		}
		args = append(args, interpolationArg{prefix: literalText(code[start:i]), key: op.text})
		start = i + 1
	}
	return &interpolation{args: args, suffix: literalText(code[start:])}
}

// literalText joins adjacent decoded literals once, including fragments from ICU quoting.
func literalText(code []instruction) string {
	switch len(code) {
	case 0:
		return ""
	case 1:
		return code[0].text
	}
	var literal strings.Builder
	for _, op := range code {
		literal.WriteString(op.text)
	}
	return literal.String()
}

// emit returns the position at which the new instruction was written.
func (p *messageParser) emit(kind messageOp, text string) int {
	i := len(p.code)
	p.code = append(p.code, instruction{kind: kind, text: text})
	return i
}

// body compiles until the expected closing brace or rich-text tag.
func (p *messageParser) body(depth int, plural bool, tag string) error {
	if depth > maxMessageDepth {
		return fmt.Errorf("nesting exceeds %d levels", maxMessageDepth)
	}
	start := p.pos
	for p.pos < len(p.source) {
		ch := p.source[p.pos]
		special := strings.ContainsRune("{}'<", rune(ch)) || plural && ch == '#'
		if !special {
			p.pos++
			continue
		}
		if start < p.pos {
			p.emit(opText, p.source[start:p.pos])
		}
		p.pos++
		switch ch {
		case '{':
			if err := p.argument(depth); err != nil {
				return err
			}
		case '}':
			if tag != "" || depth == 0 {
				return fmt.Errorf("unexpected closing brace")
			}
			return nil
		case '#':
			p.emit(opPound, "")
		case '\'':
			p.emit(opText, p.quoted(plural))
		case '<':
			if strings.HasPrefix(p.source[p.pos:], "/") {
				closeTag := "/" + tag + ">"
				if tag == "" || !p.take(closeTag) {
					return fmt.Errorf("unexpected closing tag")
				}
				return nil
			}
			if err := p.richTag(depth, plural); err != nil {
				return err
			}
		}
		start = p.pos
	}
	if depth != 0 {
		return fmt.Errorf("unclosed message body")
	}
	if start < p.pos {
		p.emit(opText, p.source[start:p.pos])
	}
	return nil
}

// richTag recognizes paired tags and leaves ordinary comparison signs as text.
func (p *messageParser) richTag(depth int, plural bool) error {
	nameStart := p.pos
	if nameStart == len(p.source) || !isTagLetter(p.source[nameStart]) {
		p.emit(opText, "<")
		return nil
	}
	for p.pos < len(p.source) {
		ch := p.source[p.pos]
		digit := ch >= '0' && ch <= '9'
		if !isTagLetter(ch) && !digit && ch != '-' && ch != '_' {
			break
		}
		p.pos++
	}
	name := p.source[nameStart:p.pos]
	if p.take("/>") {
		p.emit(opText, p.source[nameStart-1:p.pos])
		return nil
	}
	if !p.take(">") {
		return fmt.Errorf("expected > after tag %q", name)
	}
	i := p.emit(opTag, name)
	if err := p.body(depth+1, plural, name); err != nil {
		return err
	}
	p.code[i].end = len(p.code)
	return nil
}

// isTagLetter recognizes an ASCII rich-text element name's first character.
func isTagLetter(ch byte) bool {
	return ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z'
}

// quoted handles ICU's apostrophe-friendly quoting, including an implied final quote.
func (p *messageParser) quoted(plural bool) string {
	if p.take("'") || p.pos == len(p.source) {
		return "'"
	}
	next := p.source[p.pos]
	escapable := strings.ContainsRune("{}<>", rune(next)) || plural && next == '#'
	if !escapable {
		return "'"
	}
	start := p.pos
	for p.pos < len(p.source) {
		if !p.take("'") {
			p.pos++
			continue
		}
		if !p.take("'") {
			return strings.ReplaceAll(p.source[start:p.pos-1], "''", "'")
		}
	}
	return strings.ReplaceAll(p.source[start:], "''", "'")
}

// take consumes a specific delimiter, if present.
func (p *messageParser) take(s string) bool {
	if !strings.HasPrefix(p.source[p.pos:], s) {
		return false
	}
	p.pos += len(s)
	return true
}

// space consumes whitespace between tokens without touching literal message text.
func (p *messageParser) space() {
	p.pos = len(p.source) - len(strings.TrimLeftFunc(p.source[p.pos:], unicode.IsSpace))
}

// token reads one argument name, style, or selector and its surrounding whitespace.
func (p *messageParser) token() string {
	p.space()
	start := p.pos
	for p.pos < len(p.source) {
		ch, size := utf8.DecodeRuneInString(p.source[p.pos:])
		if unicode.IsSpace(ch) || strings.ContainsRune("{},", ch) {
			break
		}
		p.pos += size
	}
	word := p.source[start:p.pos]
	p.space()
	return word
}

// argument compiles an interpolation, a formatter, or a conditional.
func (p *messageParser) argument(depth int) error {
	name := p.token()
	if name == "" {
		return fmt.Errorf("missing argument name")
	}
	if p.take("}") {
		p.emit(opValue, name)
		return nil
	}
	if !p.take(",") {
		return fmt.Errorf("expected comma or closing brace after %q", name)
	}
	format := p.token()
	var kind messageOp
	switch format {
	case "number":
		return p.numberStyle(name)
	case "date", "time":
		return p.dateStyle(name, format)
	case "select":
		kind = opSelect
	case "plural":
		kind = opPlural
	case "selectordinal":
		kind = opOrdinal
	default:
		return fmt.Errorf("unsupported formatter %q", format)
	}
	if !p.take(",") {
		return fmt.Errorf("expected alternatives for %q", name)
	}
	return p.alternatives(depth, kind, name)
}

// dateStyles maps the encoded style index to its CLDR width.
var dateStyles = [...]string{"medium", "short", "long", "full"}

// dateStyle validates the requested date or time width.
func (p *messageParser) dateStyle(name, format string) error {
	style := 0
	if p.take(",") {
		word := p.token()
		style = slices.Index(dateStyles[:], word)
		if style < 0 {
			return fmt.Errorf("unsupported %s style %q", format, word)
		}
	}
	if !p.take("}") {
		return fmt.Errorf("expected closing brace after %s", format)
	}
	kind := opDate
	if format == "time" {
		kind = opTime
	}
	i := p.emit(kind, name)
	p.code[i].end = style
	return nil
}

// numberStyle resolves the style at load time.
func (p *messageParser) numberStyle(name string) error {
	kind := opNumber
	if p.take(",") {
		switch style := p.token(); style {
		case "integer":
			kind = opInteger
		case "percent":
			kind = opPercent
		default:
			return fmt.Errorf("unsupported number style %q", style)
		}
	}
	if !p.take("}") {
		return fmt.Errorf("expected closing brace after number")
	}
	p.emit(kind, name)
	return nil
}

// alternatives stores sibling jumps so lookup can skip unselected bodies in one step.
func (p *messageParser) alternatives(depth int, kind messageOp, name string) error {
	i := p.emit(kind, name)
	p.space()
	hasPlural := kind != opSelect
	if hasPlural && p.take("offset:") {
		text := p.token()
		offset, err := strconv.ParseUint(text, 10, 63)
		if err != nil {
			return fmt.Errorf("invalid plural offset %q: %w", text, err)
		}
		p.emit(opOffset, strconv.FormatUint(offset, 10))
	}
	seen := make(map[string]struct{})
	for !p.take("}") {
		label := p.token()
		if label == "" || !p.take("{") {
			return fmt.Errorf("expected selector and body for %q", name)
		}
		if hasPlural {
			var err error
			label, err = normalizeSelector(label)
			if err != nil {
				return err
			}
		}
		if _, ok := seen[label]; ok {
			return fmt.Errorf("duplicate selector %q", label)
		}
		seen[label] = struct{}{}
		branch := p.emit(opCase, label)
		if err := p.body(depth+1, hasPlural, ""); err != nil {
			return err
		}
		p.code[branch].end = len(p.code)
		p.space()
	}
	if _, ok := seen["other"]; !ok {
		return fmt.Errorf("missing other alternative for %q", name)
	}
	p.code[i].end = len(p.code)
	return nil
}

// normalizeSelector validates plural categories and canonicalizes exact decimal matches.
func normalizeSelector(label string) (string, error) {
	switch label {
	case "zero", "one", "two", "few", "many", "other":
		return label, nil
	}
	if n, ok := strings.CutPrefix(label, "="); ok {
		integer, fraction, dot := strings.Cut(strings.TrimPrefix(n, "-"), ".")
		if allDigits(integer) && (!dot || allDigits(fraction)) {
			return "=" + canonicalDecimal(n), nil
		}
	}
	return "", fmt.Errorf("invalid plural selector %q", label)
}

// allDigits checks the ASCII decimal syntax used by exact selectors.
func allDigits(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

// canonicalDecimal drops insignificant zeroes without rounding integers through float64.
func canonicalDecimal(s string) string {
	unsigned := strings.TrimPrefix(s, "-")
	integer, fraction, _ := strings.Cut(unsigned, ".")
	fraction = strings.TrimRight(fraction, "0")
	integer = strings.TrimLeft(integer, "0")
	if integer == "" {
		integer = "0"
	}
	if fraction != "" {
		integer += "." + fraction
	}
	if unsigned != s && integer != "0" {
		return "-" + integer
	}
	return integer
}
