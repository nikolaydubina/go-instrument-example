package treemap

import (
	"context"
	"strings"
	"go.opentelemetry.io/otel"
)

func CollapseLongPaths(ctx context.Context, t *Tree) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "CollapseLongPaths")
	defer span.End()
	if t == nil {
		return
	}
	CollapseLongPathsFromNode(ctx, t, t.Root)
}

func CollapseLongPathsFromNode(ctx context.Context, t *Tree, nodeName string) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "CollapseLongPathsFromNode")
	defer span.End()
	if t == nil {
		return
	}

	parts := []string{}
	q := nodeName
	for children := t.To[q]; len(children) == 1; children = t.To[q] {
		nextChild := children[0]

		parts = append(parts, t.Nodes[q].Name)
		delete(t.Nodes, q)
		delete(t.To, q)

		q = nextChild
	}

	if q != nodeName {

		t.To[nodeName] = make([]string, len(t.To[q]))
		copy(t.To[nodeName], t.To[q])

		node := t.Nodes[q]

		parts = append(parts, node.Name)

		t.Nodes[nodeName] = Node{
			Path:		node.Path,
			Name:		strings.Join(parts, "/"),
			Size:		node.Size,
			Heat:		node.Heat,
			HasHeat:	node.HasHeat,
		}

		delete(t.Nodes, q)
		delete(t.To, q)
	}

	for _, node := range t.To[nodeName] {
		CollapseLongPathsFromNode(ctx, t, node)
	}
}
