package steering

import (
	"github.com/cyrilix/robocar-base/testtools"
	"github.com/cyrilix/robocar-protobuf/go/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
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
	registerCallbacks = func(p *Controller) error {
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
	objectsTopic := "topic/objects"

	p := NewController(nil, steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic, objectsTopic)

	cases := []struct {
		driveMode        events.DriveModeMessage
		rcSteering       events.SteeringMessage
		tfSteering       events.SteeringMessage
		expectedSteering events.SteeringMessage
		objects          events.ObjectsMessage
	}{
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
			events.ObjectsMessage{},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
			events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
			events.ObjectsMessage{},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
			events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.7, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.7, Confidence: 1.0},
			events.ObjectsMessage{},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_COPILOT},
			events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
			events.ObjectsMessage{},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_COPILOT},
			events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.7, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.7, Confidence: 1.0},
			events.ObjectsMessage{},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.8, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
			events.ObjectsMessage{},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.9, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
			events.ObjectsMessage{},
		},
		{
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
			events.SteeringMessage{Steering: -0.3, Confidence: 1.0},
			events.SteeringMessage{Steering: 0.6, Confidence: 1.0},
			events.ObjectsMessage{},
		},
	}

	go p.Start()
	defer func() { close(p.cancel) }()

	for _, c := range cases {

		p.onDriveMode(nil, testtools.NewFakeMessageFromProtobuf(driveModeTopic, &c.driveMode))
		p.onRCSteering(nil, testtools.NewFakeMessageFromProtobuf(rcSteeringTopic, &c.rcSteering))
		p.onTFSteering(nil, testtools.NewFakeMessageFromProtobuf(tfSteeringTopic, &c.tfSteering))
		p.onObjects(nil, testtools.NewFakeMessageFromProtobuf(objectsTopic, &c.objects))

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
					c.driveMode.String(), msg.GetSteering(), c.expectedSteering.GetSteering())
			}
			if msg.GetConfidence() != 1. {
				t.Errorf("bad throtlle confidence: %v, wants %v", msg.GetConfidence(), 1.)
			}

			time.Sleep(1 * time.Millisecond)
		}
	}
}

type StaticCorrector struct {
	delta float64
}

func (s *StaticCorrector) AdjustFromObjectPosition(currentSteering float64, objects []*events.Object) float64 {
	return s.delta
}

func TestController_Start(t *testing.T) {
	oldRegister := registerCallbacks
	oldPublish := publish
	defer func() {
		registerCallbacks = oldRegister
		publish = oldPublish
	}()
	registerCallbacks = func(p *Controller) error {
		return nil
	}

	waitPublish := sync.WaitGroup{}
	var muEventsPublished sync.Mutex
	eventsPublished := make(map[string][]byte)
	publish = func(client mqtt.Client, topic string, payload *[]byte) {
		muEventsPublished.Lock()
		defer muEventsPublished.Unlock()
		eventsPublished[topic] = *payload
		waitPublish.Done()
	}

	steeringTopic := "topic/steering"
	driveModeTopic := "topic/driveMode"
	rcSteeringTopic := "topic/rcSteering"
	tfSteeringTopic := "topic/tfSteering"
	objectsTopic := "topic/objects"

	type fields struct {
		driveMode              events.DriveMode
		enableCorrection       bool
		enableCorrectionOnUser bool
	}
	type msgEvents struct {
		driveMode        events.DriveModeMessage
		rcSteering       events.SteeringMessage
		tfSteering       events.SteeringMessage
		expectedSteering events.SteeringMessage
		objects          events.ObjectsMessage
	}

	tests := []struct {
		name               string
		fields             fields
		msgEvents          msgEvents
		correctionOnObject float64
		want               events.SteeringMessage
		wantErr            bool
	}{
		{
			name: "On user drive mode, none correction",
			fields: fields{
				driveMode:              events.DriveMode_USER,
				enableCorrection:       false,
				enableCorrectionOnUser: false,
			},
			msgEvents: msgEvents{
				driveMode:  events.DriveModeMessage{DriveMode: events.DriveMode_USER},
				rcSteering: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
				tfSteering: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
				objects:    events.ObjectsMessage{Objects: []*events.Object{&objectOnMiddleNear}},
			},
			correctionOnObject: 0.5,
			// Get rc value without correction
			want: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
		},
		{
			name: "On pilot drive mode, none correction",
			fields: fields{
				driveMode:              events.DriveMode_PILOT,
				enableCorrection:       false,
				enableCorrectionOnUser: false,
			},
			msgEvents: msgEvents{
				driveMode:  events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
				rcSteering: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
				tfSteering: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
				objects:    events.ObjectsMessage{Objects: []*events.Object{&objectOnMiddleNear}},
			},
			correctionOnObject: 0.5,
			// Get rc value without correction
			want: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
		},
		{
			name: "On copilot drive mode, none correction",
			fields: fields{
				driveMode:              events.DriveMode_COPILOT,
				enableCorrection:       false,
				enableCorrectionOnUser: false,
			},
			msgEvents: msgEvents{
				driveMode:  events.DriveModeMessage{DriveMode: events.DriveMode_COPILOT},
				rcSteering: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
				tfSteering: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
				objects:    events.ObjectsMessage{Objects: []*events.Object{&objectOnMiddleNear}},
			},
			correctionOnObject: 0.5,
			// Get rc value without correction
			want: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
		},
		{
			name: "On pilot drive mode, correction enabled",
			fields: fields{
				driveMode:              events.DriveMode_PILOT,
				enableCorrection:       true,
				enableCorrectionOnUser: false,
			},
			msgEvents: msgEvents{
				driveMode:  events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
				rcSteering: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
				tfSteering: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
				objects:    events.ObjectsMessage{Objects: []*events.Object{&objectOnMiddleNear}},
			},
			correctionOnObject: 0.5,
			// Get rc value without correction
			want: events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
		},
		{
			name: "On pilot drive mode, all corrections enabled",
			fields: fields{
				driveMode:              events.DriveMode_PILOT,
				enableCorrection:       true,
				enableCorrectionOnUser: true,
			},
			msgEvents: msgEvents{
				driveMode:  events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
				rcSteering: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
				tfSteering: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
				objects:    events.ObjectsMessage{Objects: []*events.Object{&objectOnMiddleNear}},
			},
			correctionOnObject: 0.5,
			// Get rc value without correction
			want: events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
		},
		{
			name: "On copilot drive mode, correction enabled",
			fields: fields{
				driveMode:              events.DriveMode_COPILOT,
				enableCorrection:       true,
				enableCorrectionOnUser: false,
			},
			msgEvents: msgEvents{
				driveMode:  events.DriveModeMessage{DriveMode: events.DriveMode_COPILOT},
				rcSteering: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
				tfSteering: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
				objects:    events.ObjectsMessage{Objects: []*events.Object{&objectOnMiddleNear}},
			},
			correctionOnObject: 0.5,
			// Get rc value without correction
			want: events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
		},
		{
			name: "On copilot drive mode, all corrections enabled",
			fields: fields{
				driveMode:              events.DriveMode_COPILOT,
				enableCorrection:       true,
				enableCorrectionOnUser: true,
			},
			msgEvents: msgEvents{
				driveMode:  events.DriveModeMessage{DriveMode: events.DriveMode_COPILOT},
				rcSteering: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
				tfSteering: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
				objects:    events.ObjectsMessage{Objects: []*events.Object{&objectOnMiddleNear}},
			},
			correctionOnObject: 0.5,
			// Get rc value without correction
			want: events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
		},
		{
			name: "On user drive mode, only correction PILOT enabled",
			fields: fields{
				driveMode:              events.DriveMode_PILOT,
				enableCorrection:       true,
				enableCorrectionOnUser: false,
			},
			msgEvents: msgEvents{
				driveMode:  events.DriveModeMessage{DriveMode: events.DriveMode_USER},
				rcSteering: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
				tfSteering: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
				objects:    events.ObjectsMessage{Objects: []*events.Object{&objectOnMiddleNear}},
			},
			correctionOnObject: 0.5,
			// Get rc value without correction
			want: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
		},
		{
			name: "On user drive mode, all corrections enabled",
			fields: fields{
				driveMode:              events.DriveMode_USER,
				enableCorrection:       true,
				enableCorrectionOnUser: true,
			},
			msgEvents: msgEvents{
				driveMode:  events.DriveModeMessage{DriveMode: events.DriveMode_USER},
				rcSteering: events.SteeringMessage{Steering: 0.3, Confidence: 1.0},
				tfSteering: events.SteeringMessage{Steering: 0.4, Confidence: 1.0},
				objects:    events.ObjectsMessage{Objects: []*events.Object{&objectOnMiddleNear}},
			},
			correctionOnObject: 0.5,
			// Get rc value without correction
			want: events.SteeringMessage{Steering: 0.5, Confidence: 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewController(nil,
				steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic, objectsTopic,
				WithObjectsCorrectionEnabled(tt.fields.enableCorrection, tt.fields.enableCorrectionOnUser),
				WithCorrector(&StaticCorrector{delta: tt.correctionOnObject}),
			)
			go c.Start()
			time.Sleep(1 * time.Millisecond)

			// Publish events and wait generation of new steering message
			waitPublish.Add(1)
			c.onDriveMode(nil, testtools.NewFakeMessageFromProtobuf(driveModeTopic, &tt.msgEvents.driveMode))
			c.onRCSteering(nil, testtools.NewFakeMessageFromProtobuf(rcSteeringTopic, &tt.msgEvents.rcSteering))
			c.onTFSteering(nil, testtools.NewFakeMessageFromProtobuf(tfSteeringTopic, &tt.msgEvents.tfSteering))
			c.onObjects(nil, testtools.NewFakeMessageFromProtobuf(objectsTopic, &tt.msgEvents.objects))
			waitPublish.Wait()

			var msg events.SteeringMessage
			muEventsPublished.Lock()
			err := proto.Unmarshal(eventsPublished[steeringTopic], &msg)
			if err != nil {
				t.Errorf("unable to unmarshall response: %v", err)
				t.Fail()
			}
			muEventsPublished.Unlock()

			if msg.GetSteering() != tt.want.GetSteering() {
				t.Errorf("bad msg value for mode %v: %v, wants %v", c.driveMode.String(), msg.GetSteering(), tt.want.GetSteering())
			}

		})
	}
}
