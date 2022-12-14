package render

import (
	"context"
	"image/color"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/nikolaydubina/go-instrument-example/treemap"
)

var (
	DarkTextColor  color.Color = color.Black
	LightTextColor color.Color = color.White
)

type NoneColorer struct{}

func (s NoneColorer) ColorBox(ctx context.Context, tree treemap.Tree, node string) color.Color {
	return color.Transparent
}

func (s NoneColorer) ColorText(ctx context.Context, tree treemap.Tree, node string) color.Color {
	return DarkTextColor
}

// HeatColorer will use heat field of nodes.
// If not present, then will pick midrange.
// This is proxy for go-colorful palette.
type HeatColorer struct {
	Palette ColorfulPalette
}

func (s HeatColorer) ColorBox(ctx context.Context, tree treemap.Tree, node string) color.Color {
	n, ok := tree.Nodes[node]
	if !ok || !n.HasHeat {
		return s.Palette.GetInterpolatedColorFor(ctx, 0.5)
	}
	return s.Palette.GetInterpolatedColorFor(ctx, n.Heat)
}

func (s HeatColorer) ColorText(ctx context.Context, tree treemap.Tree, node string) color.Color {
	boxColor := s.ColorBox(ctx, tree, node).(colorful.Color)
	_, _, l := boxColor.Hcl()
	switch {
	case l > 0.5:
		return DarkTextColor
	default:
		return LightTextColor
	}
}
