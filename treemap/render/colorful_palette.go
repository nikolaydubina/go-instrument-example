package render

import (
	"context"
	_ "embed"
	"image/color"
	"log"
	"strconv"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
	"go.opentelemetry.io/otel"
)

type ColorfulPalette []struct {
	Col	colorful.Color
	Pos	float64
}

func (gt ColorfulPalette) GetInterpolatedColorFor(ctx context.Context, t float64) color.Color {
	ctx, span := otel.Tracer("my-service").Start(ctx, "ColorfulPalette.GetInterpolatedColorFor")
	defer span.End()
	for i := 0; i < len(gt)-1; i++ {
		c1 := gt[i]
		c2 := gt[i+1]
		if c1.Pos <= t && t <= c2.Pos {

			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			return c1.Col.BlendHcl(c2.Col, t).Clamped()
		}
	}

	return gt[len(gt)-1].Col
}

var paletteReBuCSV string

var paletteRdYlGnCSV string

func makePaletteFromCSV(ctx context.Context, csv string) ColorfulPalette {
	ctx, span := otel.Tracer("my-service").Start(ctx, "makePaletteFromCSV")
	defer span.End()
	rows := strings.Split(csv, "\n")
	palette := make(ColorfulPalette, len(rows))

	for i, row := range rows {
		parts := strings.Split(row, ",")
		if len(parts) != 2 {
			continue
		}

		c, err := colorful.Hex(parts[0])
		if err != nil {
			log.Fatal(err)
		}

		v, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Fatal(err)
		}

		palette[i].Col = c
		palette[i].Pos = v
	}

	return palette
}

func GetPalette(ctx context.Context, name string) (ColorfulPalette, bool) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "GetPalette")
	defer span.End()
	switch name {
	case "RdBu":
		return makePaletteFromCSV(ctx, paletteReBuCSV), true
	case "RdYlGn":
		return makePaletteFromCSV(ctx, paletteRdYlGnCSV), true
	default:
		return nil, false
	}
}
