package steering

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"os"
	"reflect"
	"testing"
)

type ObjectsList struct {
	BBoxes []BBox `json:"bboxes"`
}
type BBox struct {
	Left       float32 `json:"left"`
	Top        float32 `json:"top"`
	Bottom     float32 `json:"bottom"`
	Right      float32 `json:"right"`
	Confidence float32 `json:"confidence"`
}

var (
	bboxes1 []image.Rectangle
	bboxes2 []image.Rectangle
	bboxes3 []image.Rectangle
	bboxes4 []image.Rectangle
)

func init() {
	img1, bb, err := load_data("01")
	if err != nil {
		zap.S().Panicf("unable to load data test: %w", err)
	}
	defer img1.Close()
	bboxes1 = bboxesToRectangles(bb, img1.Cols(), img1.Rows())

	img2, bb, err := load_data("02")
	if err != nil {
		zap.S().Panicf("unable to load data test: %w", err)
	}
	defer img2.Close()
	bboxes2 = bboxesToRectangles(bb, img2.Cols(), img2.Rows())

	img3, bb, err := load_data("03")
	if err != nil {
		zap.S().Panicf("unable to load data test: %w", err)
	}
	defer img3.Close()
	bboxes3 = bboxesToRectangles(bb, img3.Cols(), img3.Rows())

	img4, bb, err := load_data("04")
	if err != nil {
		zap.S().Panicf("unable to load data test: %w", err)
	}
	defer img4.Close()
	bboxes4 = bboxesToRectangles(bb, img4.Cols(), img4.Rows())
}

func bboxesToRectangles(bboxes []BBox, imgWidth, imgHeiht int) []image.Rectangle {
	rects := make([]image.Rectangle, 0, len(bboxes))
	for _, bb := range bboxes {
		rects = append(rects, bb.toRect(imgWidth, imgHeiht))
	}
	return rects
}

func (bb *BBox) toRect(imgWidth, imgHeight int) image.Rectangle {
	return image.Rect(
		int(bb.Left*float32(imgWidth)),
		int(bb.Top*float32(imgHeight)),
		int(bb.Right*float32(imgWidth)),
		int(bb.Bottom*float32(imgHeight)),
	)
}

func load_data(dataName string) (*gocv.Mat, []BBox, error) {
	contentBBoxes, err := os.ReadFile(fmt.Sprintf("test_data/bboxes-%s.json", dataName))
	if err != nil {
		return nil, []BBox{}, fmt.Errorf("unable to load json file for bbox of '%v': %w", dataName, err)
	}

	var obj ObjectsList
	err = json.Unmarshal(contentBBoxes, &obj)
	if err != nil {
		return nil, []BBox{}, fmt.Errorf("unable to unmarsh json file for bbox of '%v': %w", dataName, err)
	}

	imgContent, err := os.ReadFile(fmt.Sprintf("test_data/img-%s.jpg", dataName))
	if err != nil {
		return nil, []BBox{}, fmt.Errorf("unable to load jpg file of '%v': %w", dataName, err)
	}
	img, err := gocv.IMDecode(imgContent, gocv.IMReadUnchanged)
	if err != nil {
		return nil, []BBox{}, fmt.Errorf("unable to load jpg of '%v': %w", dataName, err)
	}
	return &img, obj.BBoxes, nil
}

func drawImage(img *gocv.Mat, bboxes []BBox) {
	for _, bb := range bboxes {
		gocv.Rectangle(img, bb.toRect(img.Cols(), img.Rows()), color.RGBA{R: 0, G: 255, B: 0, A: 0}, 2)
		gocv.PutText(
			img,
			fmt.Sprintf("%.2f", bb.Confidence),
			image.Point{
				X: int(bb.Left*float32(img.Cols()) + 10.),
				Y: int(bb.Top*float32(img.Rows()) + 10.),
			},
			gocv.FontHersheyTriplex,
			0.4,
			color.RGBA{R: 0, G: 0, B: 0, A: 0},
			1)
	}
}

func saveImage(name string, img *gocv.Mat) error {
	err := os.MkdirAll("test_result", os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directory for test result: %w", err)
	}
	jpg, err := gocv.IMEncode(gocv.JPEGFileExt, *img)
	if err != nil {
		return fmt.Errorf("unable to encode jpg image: %w", err)
	}
	defer jpg.Close()

	err = os.WriteFile(fmt.Sprintf("test_result/%s.jpg", name), jpg.GetBytes(), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to write jpeg file: %w", err)
	}
	return nil
}

func DisplayImageAndBBoxes(dataName string) error {
	img, bboxes, err := load_data(dataName)
	if err != nil {
		return fmt.Errorf("unable to load image and bboxes: %w", err)
	}
	drawImage(img, bboxes)
	err = saveImage(dataName, img)
	if err != nil {
		return fmt.Errorf("unable to save image: %w", err)
	}
	return nil
}

func TestDisplayBBox(t *testing.T) {

	type args struct {
		dataName string
	}
	tests := []struct {
		name string
		args args
		//want []*image.Rectangle
	}{
		{
			name: "default",
			args: args{dataName: "01"},
		},
		{
			name: "02",
			args: args{dataName: "02"},
		},
		{
			name: "03",
			args: args{dataName: "03"},
		},
		{
			name: "04",
			args: args{dataName: "04"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DisplayImageAndBBoxes(tt.args.dataName)
			if err != nil {
				t.Errorf("unable to draw image: %v", err)
			}
		})
	}

}

func TestGroupBBoxes(t *testing.T) {
	type args struct {
		bboxes []image.Rectangle
	}
	tests := []struct {
		name string
		args args
		want []image.Rectangle
	}{
		{
			name: "data-01",
			args: args{
				bboxes: bboxes1,
			},
			want: []image.Rectangle{image.Rectangle{Min: image.Point{X: 1, Y: 2}, Max: image.Point{X: 3, Y: 4}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GroupBBoxes(tt.args.bboxes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GroupBBoxes() = %v, want %v", got, tt.want)
			}
		})
	}
}
