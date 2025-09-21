package termenv

import (
	"fmt"
	"testing"
)

func BenchmarkSequence(b *testing.B) {
	colors := map[string]Color{
		"ANSI":    ANSI.Color("7"),
		"ANSI256": ANSI256.Color("91"),
		"RGB":     TrueColor.Color("#abcdef"),
	}

	var title string
	for _, color := range colors {
		title = fmt.Sprintf("%T:fmt.Sprintf", color)
		b.Run(title, func(b *testing.B) {
			for b.Loop() {
				color.Sequence(true)
			}
		})

		if _, ok := color.(ANSIColor); ok {
			title = fmt.Sprintf("%T:strconv.FormatInt", color)
			b.Run(title, func(b *testing.B) {
				for b.Loop() {
					color.SequenceBuf(true)
				}
			})
		} else {
			title = fmt.Sprintf("%T:strings.Builder", color)
			b.Run(title, func(b *testing.B) {
				for b.Loop() {
					color.SequenceBuilder(true)
				}
			})

			title = fmt.Sprintf("%T:[]byte", color)
			b.Run(title, func(b *testing.B) {
				for b.Loop() {
					color.SequenceBuf(true)
				}
			})
		}
		fmt.Println()
	}
}
