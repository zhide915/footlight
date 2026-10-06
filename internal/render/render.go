package render

import (
	"strings"

	"github.com/zhide915/footlight/internal/color"
	"github.com/zhide915/footlight/internal/widget"
)

var markerColor = color.MustParse("red")

func Render(lines []widget.BuiltLine, c widget.Ctx, cfgErr bool) string {
	out := make([]string, 0, len(lines))
	for _, bl := range lines {
		sep := bl.Sep
		if sep == "" {
			sep = " · "
		}
		parts := make([]string, 0, len(bl.Widgets))
		for _, w := range bl.Widgets {
			if t, ok := w.Render(c); ok {
				parts = append(parts, t)
			}
		}
		out = append(out, strings.Join(parts, sep))
	}
	res := strings.Join(out, "\n")
	if cfgErr {
		marker := color.FG(markerColor).Apply(c.Color, "⚠ cfg")
		if res == "" {
			return marker
		}
		res += " " + marker
	}
	return res
}
