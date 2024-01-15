package steering

import (
	"github.com/cyrilix/robocar-protobuf/go/events"
	"gocv.io/x/gocv"
	"image"
)

func GroupBBoxes(bboxes []image.Rectangle) []image.Rectangle {
	if len(bboxes) == 0 {
		return []image.Rectangle{}
	}
	if len(bboxes) == 1 {
		return []image.Rectangle{bboxes[0]}
	}
	return gocv.GroupRectangles(bboxes, 1, 0.2)
}
func GroupObjects(objects []*events.Object, imgWidth, imgHeight int) []*events.Object {
	if len(objects) == 0 {
		return []*events.Object{}
	}
	if len(objects) == 1 {
		return []*events.Object{objects[0]}
	}

	rectangles := make([]image.Rectangle, 0, len(objects))
	for _, o := range objects {
		rectangles = append(rectangles, *objectToRect(o, imgWidth, imgHeight))
	}
	grp := gocv.GroupRectangles(rectangles, 1, 0.2)
	result := make([]*events.Object, 0, len(grp))
	for _, r := range grp {
		result = append(result, rectToObject(&r, imgWidth, imgHeight))
	}
	return result
}

func objectToRect(object *events.Object, imgWidth, imgHeight int) *image.Rectangle {
	r := image.Rect(
		int(object.Left*float32(imgWidth)),
		int(object.Top*float32(imgHeight)),
		int(object.Right*float32(imgWidth)),
		int(object.Bottom*float32(imgHeight)),
	)
	return &r
}

func sizeObject(object *events.Object, imgWidth, imgHeight int) float64 {
	r := objectToRect(object, imgWidth, imgHeight)
	return float64(r.Dx()) * float64(r.Dy())
}

func rectToObject(r *image.Rectangle, imgWidth, imgHeight int) *events.Object {
	return &events.Object{
		Type:       events.TypeObject_ANY,
		Left:       float32(r.Min.X) / float32(imgWidth),
		Top:        float32(r.Min.Y) / float32(imgHeight),
		Right:      float32(r.Max.X) / float32(imgWidth),
		Bottom:     float32(r.Max.Y) / float32(imgHeight),
		Confidence: -1,
	}
}
