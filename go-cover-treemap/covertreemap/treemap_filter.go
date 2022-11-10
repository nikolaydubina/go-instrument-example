package covertreemap

import (
	"context"
	"strings"

	"github.com/nikolaydubina/go-instrument-example/treemap"
	"go.opentelemetry.io/otel"
)

func RemoveGoFilesTreemapFilter(ctx context.Context, tree *treemap.Tree) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "RemoveGoFilesTreemapFilter")
	defer span.End()
	for path := range tree.Nodes {
		if strings.HasSuffix(path, ".go") {
			delete(tree.Nodes, path)
		}
	}

	for parent, children := range tree.To {
		childrenNew := make([]string, 0, len(children))

		for _, child := range children {
			if !strings.HasSuffix(child, ".go") {
				childrenNew = append(childrenNew, child)
			}
		}

		tree.To[parent] = childrenNew
	}
}

func AggregateGoFilesTreemapFilter(ctx context.Context, tree *treemap.Tree) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "AggregateGoFilesTreemapFilter")
	defer span.End()

	aggcov := make(map[string]float64)

	for path, node := range tree.Nodes {
		if !strings.HasSuffix(path, ".go") {
			continue
		}

		parent := parent(path)
		aggPath := parent + "/" + "*"

		hasNewNode := false
		for _, to := range tree.To[parent] {
			if to == aggPath {
				hasNewNode = true
				break
			}
		}
		if !hasNewNode {
			tree.To[parent] = append(tree.To[parent], aggPath)
		}

		if _, ok := tree.Nodes[aggPath]; !ok {
			tree.Nodes[aggPath] = treemap.Node{
				Path:		aggPath,
				Name:		"*",
				Size:		0,
				Heat:		0,
				HasHeat:	true,
			}
		}
		aggNode := tree.Nodes[aggPath]
		aggNode.Size += node.Size
		tree.Nodes[aggPath] = aggNode

		aggcov[aggPath] += node.Size * node.Heat
	}

	for aggPath, cov := range aggcov {
		aggNode := tree.Nodes[aggPath]
		aggNode.Heat = cov / aggNode.Size
		tree.Nodes[aggPath] = aggNode
	}
}

func CollapseRootsWithoutNameTreemapFilter(ctx context.Context, tree *treemap.Tree) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "CollapseRootsWithoutNameTreemapFilter")
	defer span.End()
	for path := range tree.Nodes {
		parent := parent(path)
		if len(tree.To[parent]) == 1 {
			tree.To[parent] = nil
			delete(tree.Nodes, path)
		}
	}
}

func parent(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], "/")
}
