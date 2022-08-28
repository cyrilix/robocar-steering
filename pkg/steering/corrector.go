package steering

import (
	"encoding/json"
	"fmt"
	"github.com/cyrilix/robocar-protobuf/go/events"
	"go.uber.org/zap"
	"os"
)

type OptionCorrector func(c *Corrector)

func WithGridMap(configPath string) OptionCorrector {
	var gm *GridMap
	if configPath == "" {
		zap.S().Warnf("no configuration defined for grid map, use default")
		gm = &defaultGridMap
	} else {
		var err error
		gm, err = loadConfig(configPath)
		if err != nil {
			zap.S().Panicf("unable to load grid-map config from file '%v': %w", configPath, err)
		}
	}
	return func(c *Corrector) {
		c.gridMap = gm
	}
}

func WithObjectMoveFactors(configPath string) OptionCorrector {
	var omf *GridMap
	if configPath == "" {
		zap.S().Warnf("no configuration defined for objects move factors, use default")
		omf = &defaultObjectFactors
	} else {
		var err error
		omf, err = loadConfig(configPath)
		if err != nil {
			zap.S().Panicf("unable to load objects move factors config from file '%v': %w", configPath, err)
		}
	}
	return func(c *Corrector) {
		c.objectMoveFactors = omf
	}
}

func loadConfig(configPath string) (*GridMap, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to load grid-map config from file '%v': %w", configPath, err)
	}
	var gm GridMap
	err = json.Unmarshal(content, &gm)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json config '%s': %w", configPath, err)
	}
	return &gm, nil
}

func WithImageSize(width, height int) OptionCorrector {
	return func(c *Corrector) {
		c.imgWidth = width
		c.imgHeight = height
	}
}

func WidthDeltaMiddle(d float64) OptionCorrector {
	return func(c *Corrector) {
		c.deltaMiddle = d
	}

}
func NewCorrector(options ...OptionCorrector) *Corrector {
	c := &Corrector{
		gridMap:           &defaultGridMap,
		objectMoveFactors: &defaultObjectFactors,
		deltaMiddle:       0.1,
		imgWidth:          160,
		imgHeight:         120,
	}
	for _, o := range options {
		o(c)
	}
	return c
}

type Corrector struct {
	gridMap             *GridMap
	objectMoveFactors   *GridMap
	deltaMiddle         float64
	imgWidth, imgHeight int
}

/*
AdjustFromObjectPosition modify steering value according object positions

 1. To compute steering correction, split in image in zones and define correction value for each zone

    Steering computed
    :  -1   -0.66 -0.33   0    0.33  0.66   1
    0%  |-----|-----|-----|-----|-----|-----|
    :   |  0  |  0  |  0  |  0  |  0  |  0  |
    20% |-----|-----|-----|-----|-----|-----|
    :   |  0  |  0  |  0  |  0  |  0  |  0  |
    40% |-----|-----|-----|-----|-----|-----|
    :   |  0  |  0  | 0.25|-0.25|  0  |  0  |
    60% |-----|-----|-----|-----|-----|-----|
    :   |  0  | 0.25| 0.5 |-0.5 |-0.25|  0  |
    80% |-----|-----|-----|-----|-----|-----|
    :   | 0.25| 0.5 |  1  |  -1 |-0.5 |-0.25|
    100%|-----|-----|-----|-----|-----|-----|

 2. For straight (current steering near of 0), search nearest object and if:

    * left and right values < 0: use correction from right value according image splitting
    * left and right values > 0: use correction from left value according image splitting
    * left < 0 and right values > 0: use (right + (right - left) / 2) value

 3. If current steering != 0 (turn on left or right), shift right and left values proportionnaly to current steering and
    apply 2.

    :  -1   -0.66 -0.33   0    0.33  0.66   1
    0%  |-----|-----|-----|-----|-----|-----|
    :   |  0  |  0  |  0  |  0  |  0  |  0  |
    20% |-----|-----|-----|-----|-----|-----|
    :   | 0.2 | 0.1 |  0  |  0  |-0.1 |-0.2 |
    40% |-----|-----|-----|-----|-----|-----|
    :   | ... | ... | ... | ... | ... | ... |
*/
func (c *Corrector) AdjustFromObjectPosition(currentSteering float64, objects []*events.Object) float64 {
	if len(objects) == 0 {
		return currentSteering
	}
	grpObjs := GroupObjects(objects, c.imgWidth, c.imgHeight)

	// get nearest object
	nearest, err := c.nearObject(grpObjs)
	if err != nil {
		zap.S().Warnf("unexpected error on nearest seach object, ignore objects: %v", err)
		return currentSteering
	}

	if currentSteering > -1*c.deltaMiddle && currentSteering < c.deltaMiddle {
		// Straight
		return currentSteering + c.computeDeviation(nearest)
	} else {
		// Turn to right or left, so search to avoid collision with objects on the right
		// Apply factor to object to move it at middle. This factor is function of distance
		factor, err := c.objectMoveFactors.ValueOf(float64(nearest.Right), float64(nearest.Bottom))
		if err != nil {
			zap.S().Warnf("unable to compute factor to apply to object: %v", err)
			return currentSteering
		}
		objMoved := events.Object{
			Type:       nearest.Type,
			Left:       nearest.Left + float32(currentSteering*factor),
			Top:        nearest.Top,
			Right:      nearest.Right + float32(currentSteering*factor),
			Bottom:     nearest.Bottom,
			Confidence: nearest.Confidence,
		}
		result := currentSteering + c.computeDeviation(&objMoved)
		if result < -1. {
			result = -1.
		}
		if result > 1. {
			result = 1.
		}
		return result
	}
}

func (c *Corrector) computeDeviation(nearest *events.Object) float64 {
	var delta float64
	var err error

	if nearest.Left < 0 && nearest.Right < 0 {
		delta, err = c.gridMap.ValueOf(float64(nearest.Right)*2-1., float64(nearest.Bottom))
	}
	if nearest.Left > 0 && nearest.Right > 0 {
		delta, err = c.gridMap.ValueOf(float64(nearest.Left)*2-1., float64(nearest.Bottom))
	} else {
		delta, err = c.gridMap.ValueOf(float64(float64(nearest.Left)+(float64(nearest.Right)-float64(nearest.Left))/2.)*2.-1., float64(nearest.Bottom))
	}
	if err != nil {
		zap.S().Warnf("unable to compute delta to apply to steering, skip correction: %v", err)
		delta = 0
	}
	return delta
}

func (c *Corrector) nearObject(objects []*events.Object) (*events.Object, error) {
	if len(objects) == 0 {
		return nil, fmt.Errorf("list objects must contain at least one object")
	}
	if len(objects) == 1 {
		return objects[0], nil
	}

	var result *events.Object
	for _, obj := range objects {
		if result == nil || obj.Bottom > result.Bottom {
			result = obj
			continue
		}
	}
	return result, nil
}

func NewGridMapFromJson(fileName string) (*GridMap, error) {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to read content from %s file: %w", fileName, err)
	}
	var ft GridMap
	err = json.Unmarshal(content, &ft)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json content from %s file: %w", fileName, err)
	}
	// TODO: check structure is valid
	return &ft, nil
}

type GridMap struct {
	DistanceSteps []float64   `json:"distance_steps"`
	SteeringSteps []float64   `json:"steering_steps"`
	Data          [][]float64 `json:"data"`
}

func (f *GridMap) ValueOf(steering float64, distance float64) (float64, error) {
	if steering < f.SteeringSteps[0] || steering > f.SteeringSteps[len(f.SteeringSteps)-1] {
		return 0., fmt.Errorf("invalid steering value: %v, must be between %v and %v", steering, f.SteeringSteps[0], f.SteeringSteps[len(f.SteeringSteps)-1])
	}
	if distance < f.DistanceSteps[0] || distance > f.DistanceSteps[len(f.DistanceSteps)-1] {
		return 0., fmt.Errorf("invalid distance value: %v, must be between %v and %v", steering, f.DistanceSteps[0], f.DistanceSteps[len(f.DistanceSteps)-1])
	}
	// search column index
	var idxCol int
	// Start loop at 1 because first column should be skipped
	for i := 1; i < len(f.SteeringSteps); i++ {
		if steering < f.SteeringSteps[i] {
			idxCol = i - 1
			break
		}
	}

	var idxRow int
	// Start loop at 1 because first column should be skipped
	for i := 1; i < len(f.DistanceSteps); i++ {
		if distance < f.DistanceSteps[i] {
			idxRow = i - 1
			break
		}
	}

	return f.Data[idxRow][idxCol], nil
}
