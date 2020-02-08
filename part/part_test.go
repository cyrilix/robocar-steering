package part

import (
	"github.com/cyrilix/robocar-base/testtools"
	"github.com/cyrilix/robocar-protobuf/go/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/protobuf/proto"
	"sync"
	"testing"
	"time"
)

func TestDefaultSteering(t *testing.T) {
	oldRegister := registerCallbacks
	oldPublish := publish
	defer func() {
		registerCallbacks = oldRegister
		publish = oldPublish
	}()
	registerCallbacks = func(p *SteeringPart) error {
		return nil
	}

	var muEventsPublished sync.Mutex
	eventsPublished := make(map[string][]byte)
	publish = func(client mqtt.Client, topic string, payload *[]byte) {
		muEventsPublished.Lock()
		defer muEventsPublished.Unlock()
		eventsPublished[topic] = *payload
	}

	steeringTopic := "topic/steering"
	driveModeTopic := "topic/driveMode"
	rcSteeringTopic := "topic/rcSteering"
	tfSteeringTopic := "topic/tfSteering"

	p := NewPart(nil, steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic)

	cases := []struct {
		driveMode        events.DriveModeMessage
		rcSteering       events.SteeringMessage
		tfSteering       events.SteeringMessage
		expectedSteering events.SteeringMessage
	}{
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
			events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
			events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.7, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.7, Confidence: 1.0},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.8, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.9, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
			events.SteeringMessage{Steering: -0.3, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
		},
	}

	go p.Start()
	defer func() { close(p.cancel) }()

	for _, c := range cases {

		p.onDriveMode(nil, testtools.NewFakeMessageFromProtobuf(driveModeTopic, &c.driveMode))
		p.onRCSteering(nil, testtools.NewFakeMessageFromProtobuf(rcSteeringTopic, &c.rcSteering))
		p.onTFSteering(nil, testtools.NewFakeMessageFromProtobuf(tfSteeringTopic, &c.tfSteering))

		time.Sleep(10 * time.Millisecond)

		for i := 3; i >= 0; i-- {

			var msg events.SteeringMessage
			muEventsPublished.Lock()
			err := proto.Unmarshal(eventsPublished[steeringTopic], &msg)
			if err != nil {
				t.Errorf("unable to unmarshall response: %v", err)
				t.Fail()
			}
			muEventsPublished.Unlock()

			if msg.GetSteering() != c.expectedSteering.GetSteering() {
				t.Errorf("bad msg value for mode %v: %v, wants %v",
					c.driveMode, msg.GetSteering(), c.expectedSteering.GetSteering())
			}
			if msg.GetConfidence() != 1. {
				t.Errorf("bad throtlle confidence: %v, wants %v", msg.GetConfidence(), 1.)
			}

			time.Sleep(1 * time.Millisecond)
		}
	}
}
