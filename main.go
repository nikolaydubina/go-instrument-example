package main

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	chirender "github.com/go-chi/render"
	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/tools/cover"

	"github.com/nikolaydubina/go-instrument-example/go-cover-treemap/covertreemap"
	"github.com/nikolaydubina/go-instrument-example/treemap"
	"github.com/nikolaydubina/go-instrument-example/treemap/render"
)

var grey = color.RGBA{128, 128, 128, 255}

func makeCover(ctx context.Context, width float64, height float64, in io.Reader, out io.Writer) error {
	// manual instrumentation
	_, span := tracer.Start(ctx, "makeCover")
	defer span.End()

	profiles, err := cover.ParseProfilesFromReader(in)
	if err != nil {
		return fmt.Errorf("can not parse file: %w", err)
	}

	treemapBuilder := covertreemap.NewCoverageTreemapBuilder(true)
	tree, err := treemapBuilder.CoverageTreemapFromProfiles(profiles)
	if err != nil {
		return fmt.Errorf("can not build tree: %w", err)
	}

	sizeImputer := treemap.SumSizeImputer{EmptyLeafSize: 1}
	sizeImputer.ImputeSize(*tree)
	treemap.SetNamesFromPaths(tree)
	treemap.CollapseLongPaths(tree)

	heatImputer := treemap.WeightedHeatImputer{EmptyLeafHeat: 0.5}
	heatImputer.ImputeHeat(*tree)

	palette, ok := render.GetPalette("RdYlGn")
	if !ok {
		return errors.New("can not get palette")
	}
	uiBuilder := render.UITreeMapBuilder{
		Colorer:     render.HeatColorer{Palette: palette},
		BorderColor: grey,
	}
	spec := uiBuilder.NewUITreeMap(*tree, width, height, 4, 4, 16)
	renderer := render.SVGRenderer{}

	out.Write(renderer.Render(spec, width, height))
	return nil
}

func coverHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var width, height int

	query := r.URL.Query()
	if width, _ = strconv.Atoi(query.Get("w")); width == 0 {
		width = 600
	}
	if height, _ = strconv.Atoi(query.Get("h")); height == 0 {
		height = 600
	}
	if err := r.ParseForm(); err != nil {
		chirender.Status(r, 400)
		chirender.JSON(w, r, err.Error())
		return
	}
	profile, _, err := r.FormFile("profile")
	if err != nil {
		chirender.Status(r, 400)
		chirender.JSON(w, r, err.Error())
		return
	}

	if err := makeCover(ctx, float64(width), float64(height), profile, w); err != nil {
		chirender.Status(r, 400)
		chirender.JSON(w, r, err.Error())
		return
	}
}

var (
	tracer = otel.GetTracerProvider().Tracer(
		"github.com/nikolaydubina/go-instrument-example",
		trace.WithInstrumentationVersion("v0.1.0"),
		trace.WithSchemaURL(semconv.SchemaURL),
	)
)

func main() {
	client := otlptracehttp.NewClient()
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		panic(err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("go_cover_http_server"),
			semconv.ServiceVersionKey.String("0.0.1"),
		)),
	)
	otel.SetTracerProvider(tracerProvider)
	defer tracerProvider.Shutdown(context.Background())

	router := chi.NewRouter()

	router.Use(
		otelchi.Middleware("go_cover_http_server", otelchi.WithChiRoutes(router)),
	)

	router.Post("/cover", coverHandler)

	log.Fatal(http.ListenAndServe(":8080", router))
}
