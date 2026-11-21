package intl

// RightToLeft reports whether the selected writing system uses right-to-left layout.
// An explicit script, such as Latin in az-Latn, takes precedence over the language default.
func (l Locale) RightToLeft() bool {
	script, _ := l.Tag().Script()
	// Scripts with rtl=YES in Unicode CLDR 48.2 scriptMetadata.json.
	// https://github.com/unicode-org/cldr-json/blob/48.2.0/cldr-json/cldr-core/scriptMetadata.json
	switch script.String() {
	case "Adlm", "Arab", "Armi", "Avst", "Chrs", "Cprt", "Elym", "Gara", "Hatr", "Hebr",
		"Hung", "Khar", "Lydi", "Mand", "Mani", "Mend", "Merc", "Mero", "Narb", "Nbat",
		"Nkoo", "Orkh", "Ougr", "Palm", "Phli", "Phlp", "Phnx", "Prti", "Rohg", "Samr",
		"Sarb", "Sidt", "Sogd", "Sogo", "Syrc", "Thaa", "Yezi":
		return true
	default:
		return false
	}
}
