package steering

import (
	"fmt"
	"github.com/cyrilix/robocar-base/service"
	"github.com/cyrilix/robocar-protobuf/go/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
)

var (
	defaultGridMap = GridMap{
		DistanceSteps: []float64{0., 0.2, 0.4, 0.6, 0.8, 1.},
		SteeringSteps: []float64{-1., -0.66, -0.33, 0., 0.33, 0.66, 1.},
		Data: [][]float64{
			{0., 0., 0., 0., 0., 0.},
			{0., 0., 0., 0., 0., 0.},
			{0., 0., 0.25, -0.25, 0., 0.},
			{0., 0.25, 0.5, -0.5, -0.25, 0.},
			{0.25, 0.5, 1, -1, -0.5, -0.25},
		},
	}
	defaultObjectFactors = GridMap{
		DistanceSteps: []float64{0., 0.2, 0.4, 0.6, 0.8, 1.},
		SteeringSteps: []float64{-1., -0.66, -0.33, 0., 0.33, 0.66, 1.},
		Data: [][]float64{
			{0., 0., 0., 0., 0., 0.},
			{0., 0., 0., 0., 0., 0.},
			{0., 0., 0., 0., 0., 0.},
			{0., 0.25, 0, 0, -0.25, 0.},
			{0.5, 0.25, 0, 0, -0.5, -0.25},
		},
	}
)

type Option func(c *Controller)

func WithCorrector(c Corrector) Option {
	return func(ctrl *Controller) {
		ctrl.corrector = c
	}
}

func WithObjectsCorrectionEnabled(enabled, enabledOnUserDrive bool) Option {
	return func(ctrl *Controller) {
		ctrl.enableCorrection = enabled
		ctrl.enableCorrectionOnUser = enabledOnUserDrive
	}
}

func NewController(client mqtt.Client, steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic, objectsTopic string, options ...Option) *Controller {
	c := &Controller{
		client:          client,
		steeringTopic:   steeringTopic,
		driveModeTopic:  driveModeTopic,
		rcSteeringTopic: rcSteeringTopic,
		tfSteeringTopic: tfSteeringTopic,
		objectsTopic:    objectsTopic,
		driveMode:       events.DriveMode_USER,
		corrector:       NewGridCorrector(),
	}
	for _, o := range options {
		o(c)
	}
	return c
}

type Controller struct {
	client        mqtt.Client
	steeringTopic string

	muDriveMode sync.RWMutex
	driveMode   events.DriveMode

	cancel                                                         chan interface{}
	driveModeTopic, rcSteeringTopic, tfSteeringTopic, objectsTopic string

	muObjects sync.RWMutex
	objects   []*events.Object

	corrector              Corrector
	enableCorrection       bool
	enableCorrectionOnUser bool
}

func (c *Controller) Start() error {
	if err := registerCallbacks(c); err != nil {
		zap.S().Errorf("unable to register callbacks: %v", err)
		return err
	}

	c.cancel = make(chan interface{})
	<-c.cancel
	return nil
}

func (c *Controller) Stop() {
	close(c.cancel)
	service.StopService("throttle", c.client, c.driveModeTopic, c.rcSteeringTopic, c.tfSteeringTopic)
}

func (c *Controller) onObjects(_ mqtt.Client, message mqtt.Message) {
	var msg events.ObjectsMessage
	err := proto.Unmarshal(message.Payload(), &msg)
	if err != nil {
		zap.S().Errorf("unable to unmarshal protobuf %T message: %v", msg, err)
		return
	}

	c.muObjects.Lock()
	defer c.muObjects.Unlock()
	c.objects = msg.GetObjects()
}

func (c *Controller) onDriveMode(_ mqtt.Client, message mqtt.Message) {
	var msg events.DriveModeMessage
	err := proto.Unmarshal(message.Payload(), &msg)
	if err != nil {
		zap.S().Errorf("unable to unmarshal protobuf %T message: %v", msg, err)
		return
	}

	c.muDriveMode.Lock()
	defer c.muDriveMode.Unlock()
	c.driveMode = msg.GetDriveMode()
}

func (c *Controller) onRCSteering(_ mqtt.Client, message mqtt.Message) {
	c.muDriveMode.RLock()
	defer c.muDriveMode.RUnlock()

	if c.driveMode != events.DriveMode_USER {
		return
	}

	payload := message.Payload()
	evt := &events.SteeringMessage{}
	err := proto.Unmarshal(payload, evt)
	if err != nil {
		zap.S().Debugf("unable to unmarshal rc event: %v", err)
	} else {
		zap.S().Debugf("receive steering message from radio command: %0.00f", evt.GetSteering())
	}

	if c.enableCorrection && c.enableCorrectionOnUser {
		payload, err = c.adjustSteering(evt)
		if err != nil {
			zap.S().Errorf("unable to adjust steering, skip message: %v", err)
			return
		}
	}
	publish(c.client, c.steeringTopic, &payload)
}

func (c *Controller) onTFSteering(_ mqtt.Client, message mqtt.Message) {
	c.muDriveMode.RLock()
	defer c.muDriveMode.RUnlock()
	if c.driveMode != events.DriveMode_PILOT {
		// User mode, skip new message
		return
	}

	evt := &events.SteeringMessage{}
	err := proto.Unmarshal(message.Payload(), evt)
	if err != nil {
		zap.S().Errorf("unable to unmarshal tensorflow event: %v", err)
		return
	} else {
		zap.S().Debugf("receive steering message from tensorflow: %0.00f", evt.GetSteering())
	}

	payload := message.Payload()
	if c.enableCorrection {
		payload, err = c.adjustSteering(evt)
		if err != nil {
			zap.S().Errorf("unable to adjust steering, skip message: %v", err)
			return
		}
	}

	publish(c.client, c.steeringTopic, &payload)
}

func (c *Controller) adjustSteering(evt *events.SteeringMessage) ([]byte, error) {
	steering := float64(evt.GetSteering())
	steering = c.corrector.AdjustFromObjectPosition(steering, c.Objects())
	zap.S().Debugf("adjust steering to avoid objects: %v -> %v", evt.GetSteering(), steering)
	evt.Steering = float32(steering)
	// override payload content
	payload, err := proto.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal steering message with new value, skip message: %v", err)
	}
	return payload, nil
}

func (c *Controller) Objects() []*events.Object {
	c.muObjects.RLock()
	defer c.muObjects.RUnlock()
	res := make([]*events.Object, 0, len(c.objects))
	copy(res, c.objects)
	return res
}

var registerCallbacks = func(p *Controller) error {
	err := service.RegisterCallback(p.client, p.driveModeTopic, p.onDriveMode)
	if err != nil {
		return err
	}

	err = service.RegisterCallback(p.client, p.rcSteeringTopic, p.onRCSteering)
	if err != nil {
		return err
	}

	err = service.RegisterCallback(p.client, p.tfSteeringTopic, p.onTFSteering)
	if err != nil {
		return err
	}

	err = service.RegisterCallback(p.client, p.objectsTopic, p.onObjects)
	if err != nil {
		return err
	}
	return nil
}

var publish = func(client mqtt.Client, topic string, payload *[]byte) {
	client.Publish(topic, 0, false, *payload)
}
