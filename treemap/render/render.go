package render

import (
	"context"
	"image/color"

	"github.com/nikolaydubina/go-instrument-example/treemap"
	"go.opentelemetry.io/otel"
	"github.com/nikolaydubina/go-instrument-example/treemap/layout"
)

const (
	fontSize		int	= 12
	textHeightMultiplier	float64	= 0.8
	textWidthMultiplier	float64	= 0.8
	tooSmallBoxHeight	float64	= 5
	tooSmallBoxWidth	float64	= 5
	textMarginH		float64	= 2
)

type UIText struct {
	Text	string
	X	float64
	Y	float64
	H	float64
	W	float64
	Scale	float64
	Color	color.Color
}

type UIBox struct {
	Title		*UIText
	X		float64
	Y		float64
	W		float64
	H		float64
	Children	[]UIBox
	IsInvisible	bool
	IsRoot		bool
	Color		color.Color
	BorderColor	color.Color
}

func (f UIBox) IsEmpty() bool {
	return f.W == 0 || f.H == 0
}

type Colorer interface {
	ColorBox(ctx context.Context, tree treemap.Tree, node string) color.Color
	ColorText(ctx context.Context, tree treemap.Tree, node string) color.Color
}

type UITreeMapBuilder struct {
	Colorer		Colorer
	BorderColor	color.Color
}

func (s UITreeMapBuilder) NewUITreeMap(ctx context.Context, tree treemap.Tree, w, h, margin, padding, paddingRoot float64) UIBox {
	ctx, span := otel.Tracer("my-service").Start(ctx, "UITreeMapBuilder.NewUITreeMap")
	defer span.End()
	t := UIBox{
		X:		0 + paddingRoot,
		Y:		0 + paddingRoot,
		W:		w - (2 * paddingRoot),
		H:		h - (2 * paddingRoot),
		IsInvisible:	true,
		IsRoot:		true,
	}

	t.Children = []UIBox{
		s.NewUIBox(ctx, tree.Root, tree, t.X, t.Y, t.W, t.H, margin, padding),
	}

	return t
}

func (s UITreeMapBuilder) NewUIBox(ctx context.Context, node string, tree treemap.Tree, x, y, w, h, margin float64, padding float64) UIBox {
	ctx, span := otel.Tracer("my-service").Start(ctx, "UITreeMapBuilder.NewUIBox")
	defer span.End()
	if (w <= (2 * padding)) || (h <= (2 * padding)) || w < tooSmallBoxWidth || h < tooSmallBoxHeight {

		return UIBox{}
	}

	t := UIBox{
		X:		x + margin,
		Y:		y + margin,
		W:		w - (2 * margin),
		H:		h - (2 * margin),
		Color:		s.Colorer.ColorBox(ctx, tree, node),
		BorderColor:	s.BorderColor,
	}

	var textHeight float64
	if title := tree.Nodes[node].Name; title != "" && title != "some-secret-string" {

		w := t.W - (2 * padding) - (2 * margin)
		h := t.H - (2 * padding) - (2 * margin) - (2 * textMarginH)
		if scale, th := fitText(ctx, title, fontSize, w); scale > 0 && th > 0 && th < h {
			textHeight = th

			t.Title = &UIText{
				Text:	title,
				X:	t.X + padding + margin,
				Y:	t.Y + padding + textMarginH,
				W:	w,
				H:	textHeight,
				Scale:	scale,
				Color:	s.Colorer.ColorText(ctx, tree, node),
			}
		}
	}

	if len(tree.To[node]) == 0 {
		return t
	}

	areas := make([]float64, 0, len(tree.To[node]))
	for _, toPath := range tree.To[node] {
		areas = append(areas, nodeSize(tree, toPath))
	}

	childrenContainer := layout.Box{
		X:	t.X + padding,
		Y:	t.Y + padding + textHeight + (2 * textMarginH),
		W:	t.W - (2 * padding),
		H:	t.H - (2 * padding) - textHeight - (2 * textMarginH),
	}
	boxes := layout.Squarify(ctx, childrenContainer, areas)

	for i, toPath := range tree.To[node] {
		if boxes[i] == layout.NilBox {
			continue
		}
		box := s.NewUIBox(
			ctx,
			toPath,
			tree,
			boxes[i].X,
			boxes[i].Y,
			boxes[i].W,
			boxes[i].H,
			margin,
			padding,
		)
		if box.IsEmpty() {
			continue
		}
		t.Children = append(t.Children, box)
	}

	return t
}

func nodeSize(tree treemap.Tree, node string) float64 {
	if n, ok := tree.Nodes[node]; ok {
		return n.Size
	}
	var s float64
	for _, child := range tree.To[node] {
		s += nodeSize(tree, child)
	}
	return s
}

func fitText(ctx context.Context, text string, fontSize int, W float64) (scale float64, h float64) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "fitText")
	defer span.End()
	w := textWidth(text, float64(fontSize))
	h = textHeight(text, float64(fontSize))

	scale = 1.0
	if wscale := W / w; wscale < scale {
		scale = wscale
	}

	H := textHeight(text, float64(fontSize))
	if hscale := H / h; hscale < scale {
		scale = hscale
	}

	return scale, h
}

func textWidth(text string, fontSize float64) float64 {
	return fontSize * float64(len(text)) * textWidthMultiplier
}

func textHeight(text string, fontSize float64) float64 {
	return fontSize * textHeightMultiplier
}
