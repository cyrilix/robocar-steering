package steering

import (
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
