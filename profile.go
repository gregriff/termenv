package termenv

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
)

// Profile is a color profile: Ascii, ANSI, ANSI256, or TrueColor.
type Profile int

const (
	// TrueColor, 24-bit color profile.
	TrueColor = Profile(iota)
	// ANSI256, 8-bit color profile.
	ANSI256
	// ANSI, 4-bit color profile.
	ANSI
	// Ascii, uncolored profile.
	Ascii //nolint:revive
)

// Name returns the profile name as a string.
func (p Profile) Name() string {
	switch p {
	case Ascii:
		return "Ascii"
	case ANSI:
		return "ANSI"
	case ANSI256:
		return "ANSI256"
	case TrueColor:
		return "TrueColor"
	}
	return "Unknown"
}

// String returns a new Style.
func (p Profile) String(s ...string) Style {
	return Style{
		profile: p,
		string:  strings.Join(s, " "),
	}
}

// Convert transforms a given Color to a Color supported within the Profile.
func (p Profile) Convert(c Color) Color {
	if p == Ascii {
		return NoColor{}
	}

	switch v := c.(type) {
	case ANSIColor:
		return v

	case ANSI256Color:
		if p == ANSI {
			return ansi256ToANSIColor(v)
		}
		return v

	case RGBColor:
		h, err := colorful.Hex(string(v))
		if err != nil {
			return nil
		}
		if p != TrueColor {
			ac := hexToANSI256Color(h)
			if p == ANSI {
				return ansi256ToANSIColor(ac)
			}
			return ac
		}
		return v
	}

	return c
}

// ConvertRGB transforms an RGBColor to a Color supported within the Profile.
// It avoids unnessicary string conversions compared to Convert
func (p Profile) ConvertRGB(c RGBColor, s string) Color {
	if p == Ascii {
		return NoColor{}
	}

	var (
		h   colorful.Color
		err error
	)
	cache := GetSRGBCache()
	if sRGB, present := cache.Get(c); present {
		h = sRGB
	} else {
		h, err = colorful.Hex(s)
		if err != nil {
			return nil
		}
		cache.Put(c, h)
	}

	if p != TrueColor {
		ac := hexToANSI256Color(h)
		if p == ANSI {
			return ansi256ToANSIColor(ac)
		}
		return ac
	}
	return c
}

// Color creates a Color from a string. Valid inputs are hex colors, as well as
// ANSI color codes (0-15, 16-255).
func (p Profile) Color(s string) Color {
	if len(s) == 0 {
		return nil
	}

	if strings.HasPrefix(s, "#") {
		return p.ConvertRGB(RGBColor(s), s)
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}

	var c Color
	if i < 16 { //nolint:mnd
		c = ANSIColor(i)
	} else {
		c = ANSI256Color(i)
	}

	return p.Convert(c)
}

// FromColor creates a Color from a color.Color.
func (p Profile) FromColor(c color.Color) Color {
	col, _ := colorful.MakeColor(c)
	return p.Color(col.Hex())
}
