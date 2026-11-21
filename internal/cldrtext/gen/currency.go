package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// numberDoc contains the dynamic CLDR keys for each numbering system.
type numberDoc struct {
	// Numbers holds both locale defaults and system-specific formatting rules.
	Numbers map[string]json.RawMessage
}

// currencyPattern locates the grouping and fractional digits in a CLDR pattern.
var currencyPattern = regexp.MustCompile(`[#,0]+\.00`)

// loadDigits reads the decimal digits for each numbering system from cldr-core.
func loadDigits(license []byte) (map[string]string, error) {
	var doc struct {
		// Supplemental contains shared numbering-system definitions.
		Supplemental struct {
			// NumberingSystems maps system names to their decimal digits.
			NumberingSystems map[string]struct {
				// Digits lists characters from zero through nine in order.
				Digits string `json:"_digits"`
			}
		}
	}
	err := readArchive("cldr-core", "numberingSystems.json", license, func(r io.Reader) error {
		return json.NewDecoder(r).Decode(&doc)
	})
	if err != nil {
		return nil, err
	}
	digits := make(map[string]string, len(doc.Supplemental.NumberingSystems))
	for system, data := range doc.Supplemental.NumberingSystems {
		digits[system] = data.Digits
	}
	return digits, nil
}

// currencyRecord encodes a system's digits, signed patterns, separators and grouping widths.
func currencyRecord(numbers map[string]json.RawMessage, system, digits string, minimum int) (string, error) {
	var formats struct {
		// Standard places the currency symbol for positive and negative amounts.
		Standard string
		// Alpha adds spacing when a letter in a symbol touches the number.
		Alpha string `json:"standard-alphaNextToNumber"`
	}
	if err := json.Unmarshal(numbers["currencyFormats-numberSystem-"+system], &formats); err != nil {
		return "", err
	}
	var symbols map[string]string
	if err := json.Unmarshal(numbers["symbols-numberSystem-"+system], &symbols); err != nil {
		return "", err
	}
	if formats.Alpha == "" {
		formats.Alpha = formats.Standard
	}
	affixes := make([]string, 0, 4)
	for _, pattern := range []string{formats.Standard, formats.Alpha} {
		positive, negative, ok := strings.Cut(pattern, ";")
		if !ok {
			negative = "-" + positive
		}
		for _, signed := range []string{positive, negative} {
			if len(currencyPattern.FindAllStringIndex(signed, 2)) != 1 || strings.Count(signed, "¤") != 1 {
				return "", fmt.Errorf("unsupported currency pattern %q", signed)
			}
			normalized := currencyPattern.ReplaceAllString(signed, "#")
			affixes = append(affixes, quote(strings.ReplaceAll(normalized, "-", symbols["minusSign"])))
		}
	}
	integer := strings.TrimSuffix(currencyPattern.FindString(formats.Standard), ".00")
	groups := strings.Split(integer, ",")
	primary := len(groups[len(groups)-1])
	secondary := primary
	if len(groups) >= 3 {
		secondary = len(groups[len(groups)-2])
	}
	runes := make([]string, 0, 10)
	for _, r := range digits {
		runes = append(runes, strconv.Itoa(int(r)))
	}
	if len(runes) != 10 {
		return "", fmt.Errorf("numbering system %s must have ten digits", system)
	}
	group, ok := symbols["currencyGroup"]
	if !ok {
		group = symbols["group"]
	}
	decimal, ok := symbols["currencyDecimal"]
	if !ok {
		decimal = symbols["decimal"]
	}
	return fmt.Sprintf("{[10]rune{%s},[4]string{%s},%s,%s,%s,%s,%d,%d,%d}",
		strings.Join(runes, ","), strings.Join(affixes, ","), quote(group), quote(decimal),
		quote(symbols["infinity"]), quote(symbols["nan"]), primary, secondary, minimum), nil
}

// writeCurrencies indexes both explicit numbering systems and each locale's default.
func writeCurrencies(out *bytes.Buffer, license []byte) error {
	digits, err := loadDigits(license)
	if err != nil {
		return err
	}
	docs, err := loadLocales[numberDoc]("cldr-numbers-full", "numbers.json", license)
	if err != nil {
		return err
	}
	var rows pool
	locales := make(map[string]int)
	for _, tag := range slices.Sorted(maps.Keys(docs)) {
		numbers := docs[tag].Numbers
		var defaultSystem, minDigits string
		if err := json.Unmarshal(numbers["defaultNumberingSystem"], &defaultSystem); err != nil {
			return fmt.Errorf("%s default numbering system: %w", tag, err)
		}
		if err := json.Unmarshal(numbers["minimumGroupingDigits"], &minDigits); err != nil {
			return fmt.Errorf("%s minimum grouping digits: %w", tag, err)
		}
		minimum, err := strconv.Atoi(minDigits)
		if err != nil {
			return fmt.Errorf("%s minimum grouping digits: %w", tag, err)
		}
		for _, key := range slices.Sorted(maps.Keys(numbers)) {
			system, ok := strings.CutPrefix(key, "currencyFormats-numberSystem-")
			if !ok {
				continue
			}
			record, err := currencyRecord(numbers, system, digits[system], minimum)
			if err != nil {
				return fmt.Errorf("%s/%s: %w", tag, system, err)
			}
			idx := rows.add(record)
			locales[tag+"-u-nu-"+system] = idx
			if system == defaultSystem {
				locales[tag] = idx
			}
		}
	}
	if len(rows.literals) > 1<<16 {
		return errors.New("currency data exceeds uint16 index width")
	}
	tags := slices.Sorted(maps.Keys(locales))
	indexes := make([]string, len(tags))
	for i, tag := range tags {
		indexes[i] = strconv.Itoa(locales[tag])
		tags[i] = quote(tag)
	}
	writeArray(out, "currencyLocales", "string", tags)
	writeArray(out, "currencyRows", "uint16", indexes)
	writeArray(out, "currencies", "CurrencyFormat", rows.literals)
	return nil
}
