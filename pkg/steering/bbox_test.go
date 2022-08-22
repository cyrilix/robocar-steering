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
	dataBBoxes map[string][]image.Rectangle
	dataImages map[string]*gocv.Mat
)

func init() {
	// TODO: empty img without bbox
	dataNames := []string{"01", "02", "03", "04"}
	dataBBoxes = make(map[string][]image.Rectangle, len(dataNames))
	dataImages = make(map[string]*gocv.Mat, len(dataNames))

	for _, dataName := range dataNames {
		img, bb, err := load_data(dataName)
		if err != nil {
			zap.S().Panicf("unable to load data test: %v", err)
		}
		dataBBoxes[dataName] = bboxesToRectangles(bb, img.Cols(), img.Rows())
		dataImages[dataName] = img
	}
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

func drawRectangles(img *gocv.Mat, rects []image.Rectangle, c color.RGBA) {
	for _, r := range rects {
		gocv.Rectangle(img, r, c, 2)
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
		dataName string
	}
	tests := []struct {
		name string
		args args
		want []image.Rectangle
	}{
		{
			name: "groupbbox-01",
			args: args{
				dataName: "01",
			},
			want: []image.Rectangle{{Min: image.Point{X: 42, Y: 20}, Max: image.Point{X: 84, Y: 57}}},
		},
		{
			name: "groupbbox-02",
			args: args{
				dataName: "02",
			},
			want: []image.Rectangle{{Min: image.Point{X: 25, Y: 13}, Max: image.Point{X: 110, Y: 80}}},
		},
		{
			name: "groupbbox-03",
			args: args{
				dataName: "03",
			},
			want: []image.Rectangle{{Min: image.Point{X: 0, Y: 17}, Max: image.Point{X: 35, Y: 77}}},
		},
		{
			name: "groupbbox-04",
			args: args{
				dataName: "04",
			},
			want: []image.Rectangle{{Min: image.Point{X: 129, Y: 10}, Max: image.Point{X: 159, Y: 64}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupBBoxes(dataBBoxes[tt.args.dataName])
			img := gocv.NewMat()
			defer img.Close()
			dataImages[tt.args.dataName].CopyTo(&img)
			drawRectangles(&img, got, color.RGBA{R: 0, G: 0, B: 255, A: 0})
			saveImage(tt.name, &img)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GroupBBoxes() = %v, want %v", got, tt.want)
			}
		})
	}
}
