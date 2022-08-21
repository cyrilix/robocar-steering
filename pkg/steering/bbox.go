package steering

import (
	"image"
)

func GroupBBoxes(bboxes []*image.Rectangle) []*image.Rectangle {
	resp := make([]*image.Rectangle, 0, len(bboxes))
	copy(bboxes, resp)
	return resp
}
