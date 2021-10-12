package part

import (
	"github.com/cyrilix/robocar-base/service"
	"github.com/cyrilix/robocar-protobuf/go/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"sync"
)

func NewPart(client mqtt.Client, steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic string) *SteeringPart {
	return &SteeringPart{
		client:          client,
		steeringTopic:   steeringTopic,
		driveModeTopic:  driveModeTopic,
		rcSteeringTopic: rcSteeringTopic,
		tfSteeringTopic: tfSteeringTopic,
		driveMode:       events.DriveMode_USER,
	}

}

type SteeringPart struct {
	client        mqtt.Client
	steeringTopic string

	muDriveMode sync.RWMutex
	driveMode   events.DriveMode

	cancel                                           chan interface{}
	driveModeTopic, rcSteeringTopic, tfSteeringTopic string
}

func (p *SteeringPart) Start() error {
	if err := registerCallbacks(p); err != nil {
		zap.S().Errorf("unable to rgeister callbacks: %v", err)
		return err
	}

	p.cancel = make(chan interface{})
	<-p.cancel
	return nil
}

func (p *SteeringPart) Stop() {
	close(p.cancel)
	service.StopService("throttle", p.client, p.driveModeTopic, p.rcSteeringTopic, p.tfSteeringTopic)
}

func (p *SteeringPart) onDriveMode(_ mqtt.Client, message mqtt.Message) {
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

func (p *SteeringPart) onRCSteering(_ mqtt.Client, message mqtt.Message) {
	p.muDriveMode.RLock()
	defer p.muDriveMode.RUnlock()
	zap.S().Debugf("receive steering message from radio command: %v",message)
	if p.driveMode == events.DriveMode_USER {
		// Republish same content
		payload := message.Payload()
		publish(p.client, p.steeringTopic, &payload)
	}
}
func (p *SteeringPart) onTFSteering(_ mqtt.Client, message mqtt.Message) {
	p.muDriveMode.RLock()
	defer p.muDriveMode.RUnlock()
	zap.S().Debugf("receive steering message from tensorflow: %v",message)
	if p.driveMode == events.DriveMode_PILOT {
		// Republish same content
		payload := message.Payload()
		publish(p.client, p.steeringTopic, &payload)
	}
}

var registerCallbacks = func(p *SteeringPart) error {
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
