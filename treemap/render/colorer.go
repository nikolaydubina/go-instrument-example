package render

import (
	"context"
	"image/color"

	"github.com/lucasb-eyer/go-colorful"
	"go.opentelemetry.io/otel"
	"github.com/nikolaydubina/go-instrument-example/treemap"
)

var (
	DarkTextColor	color.Color	= color.Black
	LightTextColor	color.Color	= color.White
)

type NoneColorer struct{}

func (s NoneColorer) ColorBox(ctx context.Context, tree treemap.Tree, node string) color.Color {
	ctx, span := otel.Tracer("my-service").Start(ctx, "NoneColorer.ColorBox")
	defer span.End()
	return color.Transparent
}

func (s NoneColorer) ColorText(ctx context.Context, tree treemap.Tree, node string) color.Color {
	ctx, span := otel.Tracer("my-service").Start(ctx, "NoneColorer.ColorText")
	defer span.End()
	return DarkTextColor
}

type HeatColorer struct {
	Palette ColorfulPalette
}

func (s HeatColorer) ColorBox(ctx context.Context, tree treemap.Tree, node string) color.Color {
	ctx, span := otel.Tracer("my-service").Start(ctx, "HeatColorer.ColorBox")
	defer span.End()
	n, ok := tree.Nodes[node]
	if !ok || !n.HasHeat {
		return s.Palette.GetInterpolatedColorFor(ctx, 0.5)
	}
	return s.Palette.GetInterpolatedColorFor(ctx, n.Heat)
}

func (s HeatColorer) ColorText(ctx context.Context, tree treemap.Tree, node string) color.Color {
	ctx, span := otel.Tracer("my-service").Start(ctx, "HeatColorer.ColorText")
	defer span.End()
	boxColor := s.ColorBox(ctx, tree, node).(colorful.Color)
	_, _, l := boxColor.Hcl()
	switch {
	case l > 0.5:
		return DarkTextColor
	default:
		return LightTextColor
	}
}
