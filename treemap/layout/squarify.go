package layout

import (
	"context"
	"math"
	"sort"
	"go.opentelemetry.io/otel"
)

type Box struct {
	X	float64
	Y	float64
	W	float64
	H	float64
}

var NilBox Box = Box{}

type wrappedArea struct {
	i	int
	area	float64
}

func Squarify(ctx context.Context, box Box, areas []float64) []Box {
	ctx, span := otel.Tracer("my-service").Start(ctx, "Squarify")
	defer span.End()

	sortedAreas := make([]wrappedArea, len(areas))
	for i, s := range normalizeAreas(ctx, areas, (box.W * box.H)) {
		sortedAreas[i] = wrappedArea{i: i, area: s}
	}
	sort.Slice(sortedAreas, func(i, j int) bool { return sortedAreas[i].area > sortedAreas[j].area })

	cleanAreas := make([]float64, 0, len(areas))
	for _, v := range sortedAreas {
		if v.area > 0 {
			cleanAreas = append(cleanAreas, v.area)
		}
	}

	layout := squarifyBoxLayout{
		boxes:		nil,
		freeSpace:	box,
	}
	layout.squarify(ctx, cleanAreas, nil, math.Min(layout.freeSpace.W, layout.freeSpace.H))

	boxes := layout.boxes
	cutoffOverflows(ctx, box, layout.boxes)

	res := make([]Box, len(areas))
	for i, wr := range sortedAreas {
		if i < len(cleanAreas) && i < len(boxes) {

			res[wr.i] = boxes[i]
		} else {

			res[wr.i] = Box{}
		}
	}

	return res
}

func normalizeAreas(ctx context.Context, areas []float64, target float64) []float64 {
	ctx, span := otel.Tracer("my-service").Start(ctx, "normalizeAreas")
	defer span.End()
	var total float64
	for _, s := range areas {
		total += s
	}
	if total == target {
		return areas
	}
	n := make([]float64, len(areas))
	copy(n, areas)
	for i, s := range n {
		n[i] = target * s / total
	}
	return n
}

type squarifyBoxLayout struct {
	boxes		[]Box
	freeSpace	Box
}

func (l *squarifyBoxLayout) squarify(ctx context.Context, unassignedAreas []float64, stackAreas []float64, w float64) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "squarifyBoxLayout.squarify")
	defer span.End()
	if len(unassignedAreas) == 0 {
		l.stackBoxes(ctx, stackAreas)
		return
	}

	if len(stackAreas) == 0 {
		l.squarify(ctx, unassignedAreas[1:], []float64{unassignedAreas[0]}, w)
		return
	}

	c := unassignedAreas[0]
	if stackc := append(stackAreas, c); highestAspectRatio(ctx, stackAreas, w) > highestAspectRatio(ctx, stackc, w) {

		l.squarify(ctx, unassignedAreas[1:], stackc, w)
	} else {

		l.stackBoxes(ctx, stackAreas)
		l.squarify(ctx, unassignedAreas, nil, math.Min(l.freeSpace.W, l.freeSpace.H))
	}
}

func (l *squarifyBoxLayout) stackBoxes(ctx context.Context, stackAreas []float64) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "squarifyBoxLayout.stackBoxes")
	defer span.End()
	if l.freeSpace.W < l.freeSpace.H {
		l.stackBoxesHorizontal(ctx, stackAreas)
	} else {
		l.stackBoxesVertical(ctx, stackAreas)
	}
}

func (l *squarifyBoxLayout) stackBoxesVertical(ctx context.Context, areas []float64) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "squarifyBoxLayout.stackBoxesVertical")
	defer span.End()
	if len(areas) == 0 {
		return
	}

	stackArea := 0.0
	for _, s := range areas {
		stackArea += s
	}
	if stackArea == 0 {
		return
	}

	totalArea := l.freeSpace.W * l.freeSpace.H
	if totalArea == 0 {
		return
	}

	offset := l.freeSpace.Y
	for _, s := range areas {
		h := l.freeSpace.H * s / stackArea
		b := Box{
			X:	l.freeSpace.X,
			W:	l.freeSpace.W * stackArea / totalArea,
			Y:	offset,
			H:	h,
		}
		offset += h
		l.boxes = append(l.boxes, b)
	}

	l.freeSpace = Box{
		X:	l.freeSpace.X + (l.freeSpace.W * stackArea / totalArea),
		W:	l.freeSpace.W * (1 - (stackArea / totalArea)),
		Y:	l.freeSpace.Y,
		H:	l.freeSpace.H,
	}
}

func (l *squarifyBoxLayout) stackBoxesHorizontal(ctx context.Context, areas []float64) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "squarifyBoxLayout.stackBoxesHorizontal")
	defer span.End()
	if len(areas) == 0 {
		return
	}

	stackArea := 0.0
	for _, s := range areas {
		stackArea += s
	}
	if stackArea == 0 {
		return
	}

	totalArea := l.freeSpace.W * l.freeSpace.H
	if totalArea == 0 {
		return
	}

	offset := l.freeSpace.X
	for _, s := range areas {
		w := l.freeSpace.W * s / stackArea
		b := Box{
			X:	offset,
			W:	w,
			Y:	l.freeSpace.Y,
			H:	l.freeSpace.H * stackArea / totalArea,
		}
		offset += w
		l.boxes = append(l.boxes, b)
	}

	l.freeSpace = Box{
		X:	l.freeSpace.X,
		W:	l.freeSpace.W,
		Y:	l.freeSpace.Y + (l.freeSpace.H * stackArea / totalArea),
		H:	l.freeSpace.H * (1 - (stackArea / totalArea)),
	}
}

func highestAspectRatio(ctx context.Context, areas []float64, w float64) float64 {
	ctx, span := otel.Tracer("my-service").Start(ctx, "highestAspectRatio")
	defer span.End()
	var minArea, maxArea, totalArea float64
	for i, s := range areas {
		totalArea += s
		if i == 0 || s < minArea {
			minArea = s
		}
		if i == 0 || s > maxArea {
			maxArea = s
		}
	}

	v1 := w * w * maxArea / (totalArea * totalArea)
	v2 := totalArea * totalArea / (w * w * minArea)

	return math.Max(v1, v2)
}

func cutoffOverflows(ctx context.Context, boundingBox Box, boxes []Box) {
	ctx, span := otel.Tracer("my-service").Start(ctx, "cutoffOverflows")
	defer span.End()
	maxX := boundingBox.X + boundingBox.W
	maxY := boundingBox.Y + boundingBox.H

	for i, b := range boxes {
		if delta := (b.X + b.W) - maxX; delta > 0 {
			boxes[i].W -= delta
		}
		if delta := (b.Y + b.H) - maxY; delta > 0 {
			boxes[i].H -= delta
		}
	}
}
