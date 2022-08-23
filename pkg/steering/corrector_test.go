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
	objectOnRightNear = events.Object{
		Type:       events.TypeObject_ANY,
		Left:       0.7,
		Top:        0.8,
		Right:      0.9,
		Bottom:     0.9,
		Confidence: 0.9,
	}
	objectOnLeftNear = events.Object{
		Type:       events.TypeObject_ANY,
		Left:       0.1,
		Top:        0.8,
		Right:      0.3,
		Bottom:     0.9,
		Confidence: 0.9,
	}
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
			{0., 0., 0.25, -0.25, 0., 0.},
			{0., 0.25, 0.5, -0.5, -0.25, 0.},
			{0.25, 0.5, 1, -1, -0.5, -0.25},
		},
	}
)

func TestCorrector_AdjustFromObjectPosition(t *testing.T) {
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
			want: 1,
		},
		{
			name: "run to left with 1 near object",
			args: args{
				currentSteering: -0.9,
				objects:         []*events.Object{&objectOnMiddleNear},
			},
			want: -0.9,
		},
		{
			name: "run to right with 1 near object",
			args: args{
				currentSteering: 0.9,
				objects:         []*events.Object{&objectOnMiddleNear},
			},
			want: 0.9,
		},
		{
			name: "run to right with 1 near object on the right",
			args: args{
				currentSteering: 0.9,
				objects:         []*events.Object{&objectOnRightNear},
			},
			want: 0.1,
		},
		{
			name: "run to left with 1 near object on the left",
			args: args{
				currentSteering: -0.9,
				objects:         []*events.Object{&objectOnLeftNear},
			},
			want: -0.1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Corrector{
				gridMap: &defaultGridMap,
			}
			if got := c.AdjustFromObjectPosition(tt.args.currentSteering, tt.args.objects); got != tt.want {
				t.Errorf("AdjustFromObjectPosition() = %v, want %v", got, tt.want)
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

func TestNewGridMapFromJson(t *testing.T) {
	type args struct {
		fileName string
	}
	tests := []struct {
		name    string
		args    args
		want    *GridMap
		wantErr bool
	}{
		{
			name: "default config",
			args: args{
				fileName: "test_data/config.json",
			},
			want: &defaultGridMap,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGridMapFromJson(tt.args.fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGridMapFromJson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(*got, *tt.want) {
				t.Errorf("NewGridMapFromJson() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got.SteeringSteps, tt.want.SteeringSteps) {
				t.Errorf("NewGridMapFromJson(), bad steering limits: got = %v, want %v", got.SteeringSteps, tt.want.SteeringSteps)
			}
			if !reflect.DeepEqual(got.DistanceSteps, tt.want.DistanceSteps) {
				t.Errorf("NewGridMapFromJson(), bad distance limits: got = %v, want %v", got.DistanceSteps, tt.want.DistanceSteps)
			}
		})
	}
}

func TestGridMap_ValueOf(t *testing.T) {
	type fields struct {
		DistanceSteps []float64
		SteeringSteps []float64
		Data          [][]float64
	}
	type args struct {
		steering float64
		distance float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "nominal",
			fields: fields{
				DistanceSteps: defaultGridMap.DistanceSteps,
				SteeringSteps: defaultGridMap.SteeringSteps,
				Data:          defaultGridMap.Data,
			},
			args: args{
				steering: 0.,
				distance: 0.,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "limit distance <",
			fields: fields{
				DistanceSteps: defaultGridMap.DistanceSteps,
				SteeringSteps: defaultGridMap.SteeringSteps,
				Data:          defaultGridMap.Data,
			},
			args: args{
				steering: 0,
				distance: 0.39999,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "limit distance >",
			fields: fields{
				DistanceSteps: defaultGridMap.DistanceSteps,
				SteeringSteps: defaultGridMap.SteeringSteps,
				Data:          defaultGridMap.Data,
			},
			args: args{
				steering: 0,
				distance: 0.400001,
			},
			want:    -0.25,
			wantErr: false,
		},
		{
			name: "limit steering <",
			fields: fields{
				DistanceSteps: defaultGridMap.DistanceSteps,
				SteeringSteps: defaultGridMap.SteeringSteps,
				Data:          defaultGridMap.Data,
			},
			args: args{
				steering: -0.660001,
				distance: 0.85,
			},
			want:    0.25,
			wantErr: false,
		},
		{
			name: "limit steering >",
			fields: fields{
				DistanceSteps: defaultGridMap.DistanceSteps,
				SteeringSteps: defaultGridMap.SteeringSteps,
				Data:          defaultGridMap.Data,
			},
			args: args{
				steering: -0.66,
				distance: 0.85,
			},
			want:    0.5,
			wantErr: false,
		},
		{
			name: "steering < min value",
			fields: fields{
				DistanceSteps: defaultGridMap.DistanceSteps,
				SteeringSteps: defaultGridMap.SteeringSteps,
				Data:          defaultGridMap.Data,
			},
			args: args{
				steering: defaultGridMap.SteeringSteps[0] - 0.1,
				distance: 0.85,
			},
			wantErr: true,
		},
		{
			name: "steering  > max value",
			fields: fields{
				DistanceSteps: defaultGridMap.DistanceSteps,
				SteeringSteps: defaultGridMap.SteeringSteps,
				Data:          defaultGridMap.Data,
			},
			args: args{
				steering: defaultGridMap.SteeringSteps[len(defaultGridMap.SteeringSteps)-1] + 0.1,
				distance: 0.85,
			},
			wantErr: true,
		},
		{
			name: "distance < min value",
			fields: fields{
				DistanceSteps: defaultGridMap.DistanceSteps,
				SteeringSteps: defaultGridMap.SteeringSteps,
				Data:          defaultGridMap.Data,
			},
			args: args{
				steering: -0.65,
				distance: defaultGridMap.DistanceSteps[0] - 0.1,
			},
			wantErr: true,
		},
		{
			name: "distance  > max value",
			fields: fields{
				DistanceSteps: defaultGridMap.DistanceSteps,
				SteeringSteps: defaultGridMap.SteeringSteps,
				Data:          defaultGridMap.Data,
			},
			args: args{
				steering: -0.65,
				distance: defaultGridMap.DistanceSteps[len(defaultGridMap.DistanceSteps)-1] + 0.1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &GridMap{
				DistanceSteps: tt.fields.DistanceSteps,
				SteeringSteps: tt.fields.SteeringSteps,
				Data:          tt.fields.Data,
			}
			got, err := f.ValueOf(tt.args.steering, tt.args.distance)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValueOf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValueOf() = %v, want %v", got, tt.want)
			}
		})
	}
}
