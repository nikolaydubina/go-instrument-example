package render

import (
	"context"
	"image/color"
	"math"

	"github.com/lucasb-eyer/go-colorful"
	"go.opentelemetry.io/otel"
	"github.com/nikolaydubina/go-instrument-example/treemap"
)

type TreeHueColorer struct {
	Hues	map[string]float64
	C	float64
	L	float64
	Offset	float64
	DeltaH	float64
	DeltaC	float64
	DeltaL	float64
}

func (s TreeHueColorer) ColorBox(ctx context.Context, tree treemap.Tree, node string) color.Color {
	ctx, span := otel.Tracer("my-service").Start(ctx, "TreeHueColorer.ColorBox")
	defer span.End()
	if len(s.Hues) == 0 {
		for k, v := range TreeHues(ctx, tree, s.Offset) {
			s.Hues[k] = v
		}
	}

	f := func(l, a, b float64) bool {

		th, tc, tl := s.Hues[node], s.C, s.L

		h, c, l := colorful.LabToHcl(l, a, b)

		return (math.Abs(h-th) < s.DeltaH) && (math.Abs(c-tc) < s.DeltaC) && (math.Abs(l-tl) < s.DeltaL)
	}
	palette, err := colorful.SoftPaletteEx(1, colorful.SoftPaletteSettings{CheckColor: f, Iterations: 500, ManySamples: true})
	if err != nil {

		return colorful.Hcl(0, 0, 1)
	}

	return palette[0]
}

func (s TreeHueColorer) ColorText(ctx context.Context, tree treemap.Tree, node string) color.Color {
	ctx, span := otel.Tracer("my-service").Start(ctx, "TreeHueColorer.ColorText")
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

func TreeHues(ctx context.Context, tree treemap.Tree, offset float64) map[string]float64 {
	ctx, span := otel.Tracer("my-service").Start(ctx, "TreeHues")
	defer span.End()
	ranges := map[string][2]float64{tree.Root: {offset, 360 + offset}}

	que := []string{tree.Root}
	var q string
	for len(que) > 0 {
		q, que = que[0], que[1:]
		children := tree.To[q]
		que = append(que, children...)

		if len(children) == 0 {
			continue
		}

		if len(children) == 1 {

			ranges[children[0]] = ranges[q]
			continue
		}

		if len(children) > 1 {

			minH, maxH := ranges[q][0], ranges[q][1]

			split := minH
			w := math.Abs(maxH-minH) / float64(len(children))
			for i, child := range children {
				if i == (len(children) - 1) {
					ranges[child] = [2]float64{split, maxH}
					continue
				}
				ranges[child] = [2]float64{split, split + w}
				split += w
			}
		}
	}

	hues := map[string]float64{}

	for node := range tree.To {
		minH, maxH := ranges[node][0], ranges[node][1]
		hues[node] = math.Mod(((minH + maxH) / 2), 360)
	}

	for node := range tree.Nodes {
		minH, maxH := ranges[node][0], ranges[node][1]
		hues[node] = math.Mod(((minH + maxH) / 2), 360)
	}

	return hues
}
