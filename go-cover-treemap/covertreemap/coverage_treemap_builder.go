package covertreemap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nikolaydubina/go-instrument-example/treemap"
	"go.opentelemetry.io/otel"
	"golang.org/x/tools/cover"
)

type CoverageTreemapBuilder struct {
	countStatements bool
}

func NewCoverageTreemapBuilder(
	countStatements bool,
) CoverageTreemapBuilder {
	return CoverageTreemapBuilder{
		countStatements: countStatements,
	}
}

func (s CoverageTreemapBuilder) CoverageTreemapFromProfiles(ctx context.Context, profiles []*cover.Profile) (*treemap.Tree, error) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "CoverageTreemapBuilder.CoverageTreemapFromProfiles")
	defer span.End()
	if len(profiles) == 0 {
		return nil, errors.New("no profiles passed")
	}
	tree := treemap.Tree{
		Nodes:	map[string]treemap.Node{},
		To:	map[string][]string{},
	}

	hasParent := map[string]bool{}

	for _, profile := range profiles {
		if profile == nil {
			return nil, fmt.Errorf("got nil profile")
		}

		if _, ok := tree.Nodes[profile.FileName]; ok {
			return nil, fmt.Errorf("duplicate node(%s)", profile.FileName)
		}

		var size int = 1
		if s.countStatements {
			size = numStatements(ctx, profile)
			if size == 0 {

				size = 1
			}
		}

		parts := strings.Split(profile.FileName, "/")
		hasParent[parts[0]] = false

		tree.Nodes[profile.FileName] = treemap.Node{
			Path:		profile.FileName,
			Size:		float64(size),
			Heat:		percentCovered(ctx, profile),
			HasHeat:	true,
		}

		for parent, i := parts[0], 1; i < len(parts); i++ {
			child := parent + "/" + parts[i]

			tree.Nodes[parent] = treemap.Node{
				Path: parent,
			}

			tree.To[parent] = append(tree.To[parent], child)
			hasParent[child] = true

			parent = child
		}
	}

	for node, v := range tree.To {
		tree.To[node] = unique(v)
	}

	var roots []string
	for node, has := range hasParent {
		if !has {
			roots = append(roots, node)
		}
	}

	switch {
	case len(roots) == 0:
		return nil, errors.New("no roots, possible cycle in graph")
	case len(roots) > 1:
		tree.Root = "some-secret-string"
		tree.To[tree.Root] = roots
	default:
		tree.Root = roots[0]
	}

	return &tree, nil
}

func unique(a []string) []string {
	u := map[string]bool{}
	var b []string
	for _, q := range a {
		if _, ok := u[q]; !ok {
			u[q] = true
			b = append(b, q)
		}
	}
	return b
}

func percentCovered(ctx context.Context, p *cover.Profile) float64 {
	ctx, span := otel.Tracer("my-service").Start(ctx, "percentCovered")
	defer span.End()
	var total, covered int64
	for _, b := range p.Blocks {
		total += int64(b.NumStmt)
		if b.Count > 0 {
			covered += int64(b.NumStmt)
		}
	}
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total)
}

func numStatements(ctx context.Context, p *cover.Profile) int {
	ctx, span := otel.Tracer("my-service").Start(ctx, "numStatements")
	defer span.End()
	var total int
	for _, b := range p.Blocks {
		total += b.NumStmt
	}
	return total
}
