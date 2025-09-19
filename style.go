package termenv

import (
	"strings"

	"github.com/rivo/uniseg"
)

// Sequence definitions.
const (
	ResetSeq     = "0"
	BoldSeq      = "1"
	FaintSeq     = "2"
	ItalicSeq    = "3"
	UnderlineSeq = "4"
	BlinkSeq     = "5"
	ReverseSeq   = "7"
	CrossOutSeq  = "9"
	OverlineSeq  = "53"
)

// Style is a string that various rendering styles can be applied to.
type Style struct {
	profile Profile
	string
	styles []string
}

// String returns a new Style.
func String(s ...string) Style {
	return Style{
		profile: ANSI,
		string:  strings.Join(s, " "),
	}
}

func (t Style) String() string {
	return t.Styled(t.string)
}

// Styled renders s with all applied styles.
func (t Style) Styled(s string) string {
	if t.profile == Ascii {
		return s
	}
	if len(t.styles) == 0 {
		return s
	}

	seq := strings.Join(t.styles, ";")
	if seq == "" {
		return s
	}

	buf := make([]byte, 0, len(CSI)*2+len(seq)+len(s)+len(ResetSeq)+2)
	buf = append(buf, CSI...)
	buf = append(buf, seq...)
	buf = append(buf, "m"...)
	buf = append(buf, s...)
	buf = append(buf, CSI...)
	buf = append(buf, ResetSeq...)
	buf = append(buf, "m"...)
	return string(buf)
}

// Foreground sets a foreground color.
func (t Style) Foreground(c Color) Style {
	if c == nil {
		return t
	}

	var seq string
	if rgb, ok := c.(RGBColor); ok {
		cache := GetRGBSequenceCache()
		cacheKey := string(rgb)
		if s, ok := cache.Get(cacheKey); ok {
			if sequence, ok := s.(string); ok {
				t.styles = append(t.styles, sequence)
				seq = sequence
			} else {
				panic("rgbcache value type assertion failed")
			}
		} else {
			seq = rgb.Sequence(false)
			t.styles = append(t.styles, seq)
		}
		cache.Put(cacheKey, seq)
	} else {
		t.styles = append(t.styles, c.Sequence(false))
	}
	return t
}

// Background sets a background color.
func (t Style) Background(c Color) Style {
	if c != nil {
		t.styles = append(t.styles, c.Sequence(true))
	}
	return t
}

// Bold enables bold rendering.
func (t Style) Bold() Style {
	t.styles = append(t.styles, BoldSeq)
	return t
}

// Faint enables faint rendering.
func (t Style) Faint() Style {
	t.styles = append(t.styles, FaintSeq)
	return t
}

// Italic enables italic rendering.
func (t Style) Italic() Style {
	t.styles = append(t.styles, ItalicSeq)
	return t
}

// Underline enables underline rendering.
func (t Style) Underline() Style {
	t.styles = append(t.styles, UnderlineSeq)
	return t
}

// Overline enables overline rendering.
func (t Style) Overline() Style {
	t.styles = append(t.styles, OverlineSeq)
	return t
}

// Blink enables blink mode.
func (t Style) Blink() Style {
	t.styles = append(t.styles, BlinkSeq)
	return t
}

// Reverse enables reverse color mode.
func (t Style) Reverse() Style {
	t.styles = append(t.styles, ReverseSeq)
	return t
}

// CrossOut enables crossed-out rendering.
func (t Style) CrossOut() Style {
	t.styles = append(t.styles, CrossOutSeq)
	return t
}

// Width returns the width required to print all runes in Style.
func (t Style) Width() int {
	return uniseg.StringWidth(t.string)
}
