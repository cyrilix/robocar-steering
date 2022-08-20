package steering

import (
	"github.com/cyrilix/robocar-base/service"
	"github.com/cyrilix/robocar-protobuf/go/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
)

func NewController(client mqtt.Client, steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic string, debug bool) *SteeringController {
	return &SteeringController{
		client:          client,
		steeringTopic:   steeringTopic,
		driveModeTopic:  driveModeTopic,
		rcSteeringTopic: rcSteeringTopic,
		tfSteeringTopic: tfSteeringTopic,
		driveMode:       events.DriveMode_USER,
	}

}

type SteeringController struct {
	client        mqtt.Client
	steeringTopic string

	muDriveMode sync.RWMutex
	driveMode   events.DriveMode

	cancel                                           chan interface{}
	driveModeTopic, rcSteeringTopic, tfSteeringTopic string

	debug bool
}

func (p *SteeringController) Start() error {
	if err := registerCallbacks(p); err != nil {
		zap.S().Errorf("unable to rgeister callbacks: %v", err)
		return err
	}

	p.cancel = make(chan interface{})
	<-p.cancel
	return nil
}

func (p *SteeringController) Stop() {
	close(p.cancel)
	service.StopService("throttle", p.client, p.driveModeTopic, p.rcSteeringTopic, p.tfSteeringTopic)
}

func (p *SteeringController) onDriveMode(_ mqtt.Client, message mqtt.Message) {
	var msg events.DriveModeMessage
	err := proto.Unmarshal(message.Payload(), &msg)
	if err != nil {
		zap.S().Errorf("unable to unmarshal protobuf %T message: %v", msg, err)
		return
	}

	p.muDriveMode.Lock()
	defer p.muDriveMode.Unlock()
	p.driveMode = msg.GetDriveMode()
}

func (p *SteeringController) onRCSteering(_ mqtt.Client, message mqtt.Message) {
	p.muDriveMode.RLock()
	defer p.muDriveMode.RUnlock()
	if p.debug {
		var evt events.SteeringMessage
		err := proto.Unmarshal(message.Payload(), &evt)
		if err != nil {
			zap.S().Debugf("unable to unmarshal rc event: %v", err)
		} else {
			zap.S().Debugf("receive steering message from radio command: %0.00f", evt.GetSteering())
		}
	}
	if p.driveMode == events.DriveMode_USER {
		// Republish same content
		payload := message.Payload()
		publish(p.client, p.steeringTopic, &payload)
	}
}
func (p *SteeringController) onTFSteering(_ mqtt.Client, message mqtt.Message) {
	p.muDriveMode.RLock()
	defer p.muDriveMode.RUnlock()
	if p.debug {
		var evt events.SteeringMessage
		err := proto.Unmarshal(message.Payload(), &evt)
		if err != nil {
			zap.S().Debugf("unable to unmarshal tensorflow event: %v", err)
		} else {
			zap.S().Debugf("receive steering message from tensorflow: %0.00f", evt.GetSteering())
		}
	}
	if p.driveMode == events.DriveMode_PILOT {
		// Republish same content
		payload := message.Payload()
		publish(p.client, p.steeringTopic, &payload)
	}
}

var registerCallbacks = func(p *SteeringController) error {
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
	return nil
}

var publish = func(client mqtt.Client, topic string, payload *[]byte) {
	client.Publish(topic, 0, false, *payload)
}
