package steering

import (
	"fmt"
	"github.com/cyrilix/robocar-protobuf/go/events"
	"go.uber.org/zap"
)

type Corrector struct {
}

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
