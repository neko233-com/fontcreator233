// Package unicodes classifies code points by the script buckets fontcreator233
// reports.
package unicodes

// ScriptOf returns latin, greek, cyrillic, cjk, kana, hangul, symbols, or other.
func ScriptOf(r rune) string {
	switch {
	case r >= 0x3040 && r <= 0x30FF, r >= 0x31F0 && r <= 0x31FF, r >= 0xFF66 && r <= 0xFF9F:
		return "kana"
	case isHangul(r):
		return "hangul"
	case isCJK(r):
		return "cjk"
	case (r >= 0x0370 && r <= 0x03FF) || (r >= 0x1F00 && r <= 0x1FFF):
		return "greek"
	case r >= 0x0400 && r <= 0x052F:
		return "cyrillic"
	case (r >= 0x0020 && r <= 0x024F) || (r >= 0x1E00 && r <= 0x1EFF) || r == 0x0131:
		return "latin"
	case (r >= 0x2000 && r <= 0x27BF) || (r >= 0xFE30 && r <= 0xFE4F) || (r >= 0xFF00 && r <= 0xFFEF):
		return "symbols"
	default:
		return "other"
	}
}

// IsCJK reports CJK ideographs and CJK punctuation/blocks.
func IsCJK(r rune) bool { return isCJK(r) }

func isCJK(r rune) bool {
	return (r >= 0x2E80 && r <= 0x303F) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0x20000 && r <= 0x2FA1F) ||
		(r >= 0x30000 && r <= 0x323AF)
}

// IsKana reports Hiragana/Katakana blocks.
func IsKana(r rune) bool {
	return (r >= 0x3040 && r <= 0x30FF) || (r >= 0x31F0 && r <= 0x31FF) || (r >= 0xFF66 && r <= 0xFF9F)
}

// IsHangul reports Hangul syllables and Jamo.
func IsHangul(r rune) bool { return isHangul(r) }

func isHangul(r rune) bool {
	return (r >= 0x1100 && r <= 0x11FF) ||
		(r >= 0x3130 && r <= 0x318F) ||
		(r >= 0xA960 && r <= 0xA97F) ||
		(r >= 0xAC00 && r <= 0xD7AF) ||
		(r >= 0xD7B0 && r <= 0xD7FF) ||
		(r >= 0xFFA0 && r <= 0xFFDC)
}

// IsSymbol reports the punctuation/symbol buckets we derive from the base.
func IsSymbol(r rune) bool {
	return (r >= 0x2000 && r <= 0x27BF) ||
		(r >= 0xFE30 && r <= 0xFE4F) ||
		(r >= 0xFF00 && r <= 0xFFEF)
}
