package treemap

import (
	"context"
	"strings"
	"go.opentelemetry.io/otel"
)

const minHeatDifferenceForHeatmap float64 = 0.0000001

type Node struct {
	Path	string
	Name	string
	Size	float64
	Heat	float64
	HasHeat	bool
}

type Tree struct {
	Nodes	map[string]Node
	To	map[string][]string
	Root	string
}

func (t Tree) HasHeat(ctx context.Context) bool {
	ctx, span := otel.Tracer("my-service").Start(ctx, "Tree.HasHeat")
	defer span.End()
	minHeat, maxHeat := t.HeatRange(ctx)
	return (maxHeat - minHeat) > minHeatDifferenceForHeatmap
}

func (t Tree) HeatRange(ctx context.Context) (minHeat float64, maxHeat float64) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "Tree.HeatRange")
	defer span.End()
	first := true
	for _, node := range t.Nodes {
		if !node.HasHeat {
			continue
		}
		h := node.Heat

		if first {
			minHeat = h
			maxHeat = h
			first = false
			continue
		}

		if h > maxHeat {
			maxHeat = h
		}
		if h < minHeat {
			minHeat = h
		}
	}
	return minHeat, maxHeat
}

func (t Tree) NormalizeHeat(ctx context.Context) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "Tree.NormalizeHeat")
	defer span.End()
	minHeat, maxHeat := t.HeatRange(ctx)

	if (maxHeat - minHeat) < minHeatDifferenceForHeatmap {
		return
	}

	for path, node := range t.Nodes {
		if !node.HasHeat {
			continue
		}

		n := Node{
			Path:		node.Path,
			Name:		node.Name,
			Size:		node.Size,
			Heat:		(node.Heat - minHeat) / (maxHeat - minHeat),
			HasHeat:	true,
		}
		t.Nodes[path] = n
	}
}

func SetNamesFromPaths(ctx context.Context, t *Tree) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "SetNamesFromPaths")
	defer span.End()
	if t == nil {
		return
	}
	for path, node := range t.Nodes {
		parts := strings.Split(node.Path, "/")
		if len(parts) == 0 {
			continue
		}

		t.Nodes[path] = Node{
			Path:		node.Path,
			Name:		parts[len(parts)-1],
			Size:		node.Size,
			Heat:		node.Heat,
			HasHeat:	node.HasHeat,
		}
	}
}
