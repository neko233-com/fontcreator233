package derive

import (
	"fmt"
	"sort"
)

const (
	sampleASCII   = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 .,;:!?@#$%&*+-=()[]{}<>/\\|~^\"'"
	sampleLatin   = "ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÑÒÓÔÕÖØÙÚÛÜÝÞßàáâãäåæçèéêëìíîïñòóôõöøùúûüýþÿ"
	sampleGreek   = "ΑΒΓΔΕΖΗΘΙΚΛΜΝΞΟΠΡΣΤΥΦΧΨΩαβγδεζηθικλμνξοπρστυφχψωάέήίόύώ"
	sampleCyr     = "АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯабвгдеёжзийклмнопрстуфхцчшщъыьэюя"
	sampleCJK     = "天地玄黄宇宙洪荒日月盈昃辰宿列张寒来暑往秋收冬藏闰余成岁律吕调阳云腾致雨露结为霜金生丽水玉出昆冈剑号巨阙珠称夜光果珍李柰菜重芥姜海咸河淡鳞潜羽翔龙师火帝鸟官人皇始制文字乃服衣裳推位让国有虞陶唐中文你好的世界测试字体创造器"
	sampleKana    = "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんアイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン"
	sampleHangul  = "가나다라마바사아자차카타파하한국어글꼴만들기"
	sampleSymbols = "€£¥¢©®™°±×÷→←↑↓•…—–“”‘’"
)

// CoverageNames returns the supported coverage levels in order.
func CoverageNames() []string { return []string{"none", "sample", "common", "full"} }

// CoverageRunes returns the requested code points for a coverage level.
// Actual presence is determined by the embedded fonts.
func CoverageRunes(level string) ([]rune, error) {
	var runes []rune
	switch level {
	case "none":
		return nil, nil
	case "sample":
		runes = sampleRunes()
	case "common":
		runes = sampleRunes()
		for _, rg := range commonRanges {
			for r := rg[0]; r <= rg[1]; r++ {
				runes = append(runes, r)
			}
		}
	case "full":
		runes = sampleRunes()
		for _, rg := range append(append([][2]rune{}, commonRanges...), fullRanges...) {
			for r := rg[0]; r <= rg[1]; r++ {
				runes = append(runes, r)
			}
		}
	default:
		return nil, fmt.Errorf("unknown coverage level %q", level)
	}
	seen := make(map[rune]struct{}, len(runes))
	out := runes[:0]
	for _, r := range runes {
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func sampleRunes() []rune {
	var out []rune
	for _, s := range []string{
		sampleASCII, sampleLatin, sampleGreek, sampleCyr, sampleCJK,
		sampleKana, sampleHangul, sampleSymbols,
	} {
		out = append(out, []rune(s)...)
	}
	return out
}

var commonRanges = [][2]rune{
	{0x0020, 0x024F},
	{0x02B0, 0x02FF},
	{0x0300, 0x036F},
	{0x0370, 0x052F},
	{0x1E00, 0x1EFF},
	{0x2000, 0x206F},
	{0x20A0, 0x20CF},
	{0x2100, 0x214F},
	{0x2190, 0x23FF},
	{0x2500, 0x27BF},
	{0x2E80, 0x303F},
	{0x3040, 0x31EF},
	{0x3400, 0x4DBF},
	{0x4E00, 0x9FFF},
	{0xA960, 0xA97F},
	{0xAC00, 0xD7A3},
	{0xF900, 0xFAFF},
	{0xFE30, 0xFE4F},
	{0xFF00, 0xFFEF},
}

var fullRanges = [][2]rune{
	{0x20000, 0x2FA1F},
	{0x30000, 0x323AF},
}
