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

	"github.com/go-chi/chi"
	chirender "github.com/go-chi/render"
	"golang.org/x/tools/cover"
	chitrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-chi/chi"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/nikolaydubina/go-instrument-example/go-cover-treemap/covertreemap"
	"github.com/nikolaydubina/go-instrument-example/treemap"
	"github.com/nikolaydubina/go-instrument-example/treemap/render"
)

var grey = color.RGBA{128, 128, 128, 255}

func makeCover(ctx context.Context, width float64, height float64, in io.Reader, out io.Writer) error {
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

func main() {
	tracer.Start(
		tracer.WithServiceName("go_cover_http_server"),
		tracer.WithEnv("prod"),
		tracer.WithSamplingRules([]tracer.SamplingRule{tracer.RateRule(1)}),
	)
	defer tracer.Stop()

	router := chi.NewRouter()

	router.Use(
		chitrace.Middleware(chitrace.WithServiceName("go_cover_http_server")),
	)

	router.Post("/cover", coverHandler)

	log.Fatal(http.ListenAndServe(":8080", router))
}
