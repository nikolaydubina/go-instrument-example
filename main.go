package main

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"math/rand"
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
	"golang.org/x/tools/cover"

	"github.com/nikolaydubina/go-instrument-example/go-cover-treemap/covertreemap"
	"github.com/nikolaydubina/go-instrument-example/treemap"
	"github.com/nikolaydubina/go-instrument-example/treemap/render"
)

var grey = color.RGBA{128, 128, 128, 255}

func makeCover(ctx context.Context, width float64, height float64, in io.Reader, out io.Writer) (err error) {
	profiles, err := cover.ParseProfilesFromReader(in)
	if err != nil {
		return fmt.Errorf("can not parse file: %w", err)
	}

	treemapBuilder := covertreemap.NewCoverageTreemapBuilder(true)
	tree, err := treemapBuilder.CoverageTreemapFromProfiles(ctx, profiles)
	if err != nil {
		return fmt.Errorf("can not build tree: %w", err)
	}

	sizeImputer := treemap.SumSizeImputer{EmptyLeafSize: 1}
	sizeImputer.ImputeSize(ctx, *tree)
	treemap.SetNamesFromPaths(ctx, tree)
	treemap.CollapseLongPaths(ctx, tree)

	heatImputer := treemap.WeightedHeatImputer{EmptyLeafHeat: 0.5}
	heatImputer.ImputeHeat(ctx, *tree)

	palette, ok := render.GetPalette(ctx, "RdYlGn")
	if !ok {
		return errors.New("can not get palette")
	}
	uiBuilder := render.UITreeMapBuilder{
		Colorer:     render.HeatColorer{Palette: palette},
		BorderColor: grey,
	}
	spec := uiBuilder.NewUITreeMap(ctx, *tree, width, height, 4, 4, 16)
	renderer := render.SVGRenderer{}

	out.Write(renderer.Render(ctx, spec, width, height))
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

func fib(ctx context.Context, n int) (v int, err error) {
	if n == 0 || n == 1 {
		return 1, nil
	}
	if n < 0 {
		return 0, fmt.Errorf("got n(%+v) < 0", n)
	}

	if v := rand.Float32(); v < 0.05 {
		return 0, fmt.Errorf("got random error(%#v)", v)
	}

	a, _ := fib(ctx, n-1)
	b, _ := fib(ctx, n-2)

	return a + b, nil
}

func fibHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	n, _ := strconv.Atoi(chi.URLParam(r, "n"))
	v, err := fib(ctx, n)
	if err != nil {
		chirender.Status(r, 400)
		chirender.JSON(w, r, err.Error())
		return
	}

	w.Write([]byte(strconv.Itoa(v)))
}

func main() {
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithInsecure(),
		),
	)
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
	router.Get("/fib/{n}", fibHandler)

	log.Fatal(http.ListenAndServe(":8080", router))
}
