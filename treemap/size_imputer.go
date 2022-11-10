package treemap

import (
	"context"
	"strings"
	"go.opentelemetry.io/otel"
)

type SumSizeImputer struct {
	EmptyLeafSize float64
}

func (s SumSizeImputer) ImputeSize(ctx context.Context, t Tree) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "SumSizeImputer.ImputeSize")
	defer span.End()
	s.ImputeSizeNode(ctx, t, t.Root)
}

func (s SumSizeImputer) ImputeSizeNode(ctx context.Context, t Tree, node string) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "SumSizeImputer.ImputeSizeNode")
	defer span.End()
	var sum float64
	for _, child := range t.To[node] {
		s.ImputeSizeNode(ctx, t, child)
		sum += t.Nodes[child].Size
	}

	if n, ok := t.Nodes[node]; !ok || n.Size == 0 {
		v := s.EmptyLeafSize
		if len(t.To[node]) > 0 {
			v = sum
		}

		var name string
		if parts := strings.Split(node, "/"); len(parts) > 0 {
			name = parts[len(parts)-1]
		}

		t.Nodes[node] = Node{
			Path:	node,
			Name:	name,
			Size:	v,
		}
	}
}
