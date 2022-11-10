package parser

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/nikolaydubina/go-instrument-example/treemap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type CSVTreeParser struct{}

func (s CSVTreeParser) ParseString(ctx context.Context, in string) (tree *treemap.Tree, err error) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "CSVTreeParser.ParseString")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, "error")
			span.RecordError(err)
		}
	}()
	nodes, err := parseNodes(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("can not parse nodes: %w", err)
	}

	tree, err = makeTree(ctx, nodes)
	if err != nil {
		return nil, fmt.Errorf("can not make tree: %w", err)
	}

	return tree, nil
}

func parseNodes(ctx context.Context, in string) (node []treemap.Node, err error) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "parseNodes")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, "error")
			span.RecordError(err)
		}
	}()
	var nodes []treemap.Node
	r := csv.NewReader(strings.NewReader(in))
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("can not parse: %w", err)
		}

		if len(record) == 0 {
			return nil, errors.New("no values in row")
		}

		node := treemap.Node{Path: record[0]}

		if len(record) >= 2 {
			v, err := strconv.ParseFloat(record[1], 64)
			if err != nil {
				return nil, fmt.Errorf("size(%s) is not float: %w", record[1], err)
			}
			node.Size = v
		}

		if len(record) >= 3 {
			v, err := strconv.ParseFloat(record[2], 64)
			if err != nil {
				return nil, fmt.Errorf("heat(%s) is not float: %w", record[2], err)
			}
			node.Heat = v
			node.HasHeat = true
		}

		nodes = append(nodes, node)
	}
	return nodes, nil
}

func makeTree(ctx context.Context, nodes []treemap.Node) (*treemap.Tree, error) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "makeTree")
	defer span.End()
	tree := treemap.Tree{
		Nodes:	map[string]treemap.Node{},
		To:	map[string][]string{},
	}

	hasParent := map[string]bool{}

	for _, node := range nodes {
		if existingNode, ok := tree.Nodes[node.Path]; ok {
			tree.Nodes[node.Path] = treemap.Node{
				Path:		existingNode.Path,
				Name:		existingNode.Name,
				Size:		existingNode.Size + node.Size,
				Heat:		math.Max(existingNode.Heat, node.Heat),
				HasHeat:	existingNode.HasHeat || node.HasHeat,
			}
		}
		tree.Nodes[node.Path] = node

		parts := strings.Split(node.Path, "/")
		hasParent[parts[0]] = false

		for parent, i := parts[0], 1; i < len(parts); i++ {
			child := parent + "/" + parts[i]

			if _, ok := tree.Nodes[parent]; !ok {
				tree.Nodes[parent] = treemap.Node{
					Path:		parent,
					HasHeat:	false,
				}
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
