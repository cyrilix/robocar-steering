package steering

import (
	"github.com/cyrilix/robocar-protobuf/go/events"
	"reflect"
	"testing"
)

var (
	objectOnMiddleDistant = events.Object{
		Type:       events.TypeObject_ANY,
		Left:       0.4,
		Top:        0.1,
		Right:      0.6,
		Bottom:     0.2,
		Confidence: 0.9,
	}
	objectOnLeftDistant = events.Object{
		Type:       events.TypeObject_ANY,
		Left:       0.1,
		Top:        0.09,
		Right:      0.3,
		Bottom:     0.19,
		Confidence: 0.9,
	}
	objectOnRightDistant = events.Object{
		Type:       events.TypeObject_ANY,
		Left:       0.7,
		Top:        0.21,
		Right:      0.9,
		Bottom:     0.11,
		Confidence: 0.9,
	}
	objectOnMiddleNear = events.Object{
		Type:       events.TypeObject_ANY,
		Left:       0.4,
		Top:        0.8,
		Right:      0.6,
		Bottom:     0.9,
		Confidence: 0.9,
	}
)

func TestCorrector_FixFromObjectPosition(t *testing.T) {
	type args struct {
		currentSteering float64
		objects         []*events.Object
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "run straight without objects",
			args: args{
				currentSteering: 0.,
				objects:         []*events.Object{},
			},
			want: 0.,
		},
		{
			name: "run to left without objects",
			args: args{
				currentSteering: -0.9,
				objects:         []*events.Object{},
			},
			want: -0.9,
		},
		{
			name: "run to right without objects",
			args: args{
				currentSteering: 0.9,
				objects:         []*events.Object{},
			}, want: 0.9,
		},

		{
			name: "run straight with 1 distant object",
			args: args{
				currentSteering: 0.,
				objects:         []*events.Object{&objectOnMiddleDistant},
			},
			want: 0.,
		},
		{
			name: "run to left with 1 distant object",
			args: args{
				currentSteering: -0.9,
				objects:         []*events.Object{&objectOnMiddleDistant},
			},
			want: -0.9,
		},
		{
			name: "run to right with 1 distant object",
			args: args{
				currentSteering: 0.9,
				objects:         []*events.Object{&objectOnMiddleDistant},
			},
			want: 0.9,
		},

		{
			name: "run straight with 1 near object",
			args: args{
				currentSteering: 0.,
				objects:         []*events.Object{&objectOnMiddleNear},
			},
			want: 0.5,
		},
		{
			name: "run to left with 1 near object",
			args: args{
				currentSteering: -0.9,
				objects:         []*events.Object{&objectOnMiddleNear},
			},
			want: -0.4,
		},
		{
			name: "run to right with 1 near object",
			args: args{
				currentSteering: 0.9,
				objects:         []*events.Object{&objectOnMiddleNear},
			},
			want: 0.4,
		},

		// Todo Object on left/right near/distant
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Corrector{}
			if got := c.FixFromObjectPosition(tt.args.currentSteering, tt.args.objects); got != tt.want {
				t.Errorf("FixFromObjectPosition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCorrector_nearObject(t *testing.T) {
	type args struct {
		objects []*events.Object
	}
	tests := []struct {
		name    string
		args    args
		want    *events.Object
		wantErr bool
	}{
		{
			name: "List object is empty",
			args: args{
				objects: []*events.Object{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "List with only one object",
			args: args{
				objects: []*events.Object{&objectOnMiddleNear},
			},
			want:    &objectOnMiddleNear,
			wantErr: false,
		},
		{
			name: "List with many objects",
			args: args{
				objects: []*events.Object{&objectOnLeftDistant, &objectOnMiddleNear, &objectOnRightDistant, &objectOnMiddleDistant},
			},
			want:    &objectOnMiddleNear,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Corrector{}
			got, err := c.nearObject(tt.args.objects)
			if (err != nil) != tt.wantErr {
				t.Errorf("nearObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("nearObject() got = %v, want %v", got, tt.want)
			}
		})
	}
}
