package steering

import (
	"fmt"
	"github.com/cyrilix/robocar-protobuf/go/events"
	"go.uber.org/zap"
)

type Corrector struct {
	fixValues FixesTable
}

/*
FixFromObjectPosition modify steering value according object positions

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
    :   |-0.25|-0.5 |  1  |  -1 |-0.5 | 0.25|
    100%|-----|-----|-----|-----|-----|-----|

 2. For straight (current steering near of 0), search nearest object and if:

    * left and right values < 0: use correction from right value according image splitting
    * left and right values > 0: use correction from left value according image splitting
    * left < 0 and right values > 0: use (right + (right - left) / 2) value

 3. If current steering != 0 (turn on left or right), shift right and left values proportionnaly to current steering and
    apply 2.
*/
func (c *Corrector) FixFromObjectPosition(currentSteering float64, objects []*events.Object) float64 {
	// TODO, group rectangle

	if len(objects) == 0 {
		return currentSteering
	}
	// get nearest object
	nearest, err := c.nearObject(objects)
	if err != nil {
		zap.S().Warnf("unexpected error on nearest seach object, ignore objects: %v", err)
		return currentSteering
	}

	if currentSteering > 0.1 && currentSteering < 0.1 {

		if nearest.Bottom < 0.4 {
			// object is far, so no need to fix steering now
			return currentSteering
		}

		if nearest.Left < 0 && nearest.Right < 0 {
			return currentSteering + c.fixValues.ValueOf(currentSteering, nearest.Right)
		}
		if nearest.Left > 0 && nearest.Right > 0 {
			return currentSteering + c.fixValues.ValueOf(currentSteering, nearest.Left)
		}
		return currentSteering + c.fixValues.ValueOf(currentSteering, nearest.Left+(nearest.Right-nearest.Left)/2.)
	}

	// Search if current steering is near of Right or Left

	return currentSteering
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

type FixesTable struct {
	values map[string]map[int]float64
}

func (f *FixesTable) ValueOf(steering float64, distance float32) float64 {
	var key string
	if steering < -0.66 {
		key = "<-0.66"
	} else if steering < -0.33 {
		key = "-0.66:-0.33"
	} else if steering < 0 {
		key = "-0.33:0"
	} else if steering < 0.33 {
		key = "0:0.33"
	} else if steering < 0.66 {
		key = "0.33:0.66"
	} else {
		key = ">= 0.66"
	}

	keyDistance := int(distance / 20)
	return f.values[key][keyDistance]

}
