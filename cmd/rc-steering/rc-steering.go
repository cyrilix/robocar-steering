package main

import (
	"flag"
	"github.com/cyrilix/robocar-base/cli"
	"github.com/cyrilix/robocar-steering/pkg/steering"
	"go.uber.org/zap"
	"log"
	"os"
)

const (
	DefaultClientId = "robocar-steering"
)

func main() {
	var mqttBroker, username, password, clientId string
	var steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic, objectsTopic string
	var enableObjectsCorrection, enableObjectsCorrectionOnUserMode bool
	var gridMapConfig, objectsMoveFactorsConfig string
	var deltaMiddle float64

	mqttQos := cli.InitIntFlag("MQTT_QOS", 0)
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	cli.InitMqttFlags(DefaultClientId, &mqttBroker, &username, &password, &clientId, &mqttQos, &mqttRetain)

	flag.StringVar(&steeringTopic, "mqtt-topic-steering", os.Getenv("MQTT_TOPIC_STEERING"), "Mqtt topic to publish steering result, use MQTT_TOPIC_STEERING if args not set")
	flag.StringVar(&rcSteeringTopic, "mqtt-topic-rc-steering", os.Getenv("MQTT_TOPIC_RC_STEERING"), "Mqtt topic that contains RC steering value, use MQTT_TOPIC_RC_STEERING if args not set")
	flag.StringVar(&tfSteeringTopic, "mqtt-topic-tf-steering", os.Getenv("MQTT_TOPIC_TF_STEERING"), "Mqtt topic that contains tenorflow steering value, use MQTT_TOPIC_TF_STEERING if args not set")
	flag.StringVar(&driveModeTopic, "mqtt-topic-drive-mode", os.Getenv("MQTT_TOPIC_DRIVE_MODE"), "Mqtt topic that contains DriveMode value, use MQTT_TOPIC_DRIVE_MODE if args not set")
	flag.StringVar(&objectsTopic, "mqtt-topic-objects", os.Getenv("MQTT_TOPIC_OBJECTS"), "Mqtt topic that contains Objects from object detection value, use MQTT_TOPIC_OBJECTS if args not set")
	flag.BoolVar(&enableObjectsCorrection, "enable-objects-correction", false, "Adjust steering to avoid objects")
	flag.BoolVar(&enableObjectsCorrectionOnUserMode, "enable-objects-correction-user", false, "Adjust steering to avoid objects on user mode driving")
	flag.StringVar(&gridMapConfig, "grid-map-config", "", "Json file path to configure grid object correction")
	flag.StringVar(&objectsMoveFactorsConfig, "objects-move-factors-config", "", "Json file path to configure objects move corrections")
	flag.Float64Var(&deltaMiddle, "delta-middle", 0.1, "Half Percent zone to interpret as straight")
	logLevel := zap.LevelFlag("log", zap.InfoLevel, "log level")

	flag.Parse()

	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(*logLevel)
	lgr, err := config.Build()
	if err != nil {
		log.Fatalf("unable to init logger: %v", err)
	}
	defer func() {
		if err := lgr.Sync(); err != nil {
			log.Printf("unable to Sync logger: %v\n", err)
		}
	}()
	zap.ReplaceGlobals(lgr)

	zap.S().Infof("steering topic                  : %s", steeringTopic)
	zap.S().Infof("rc topic                        : %s", rcSteeringTopic)
	zap.S().Infof("tflite steering topic           : %s", tfSteeringTopic)
	zap.S().Infof("drive mode topic                : %s", driveModeTopic)
	zap.S().Infof("objects topic                   : %s", objectsTopic)
	zap.S().Infof("objects correction enabled      : %v", enableObjectsCorrection)
	zap.S().Infof("objects correction on user mode : %v", enableObjectsCorrectionOnUserMode)
	zap.S().Infof("grid map file config            : %v", gridMapConfig)
	zap.S().Infof("objects move factors grid config: %v", objectsMoveFactorsConfig)

	client, err := cli.Connect(mqttBroker, username, password, clientId)
	if err != nil {
		log.Fatalf("unable to connect to mqtt bus: %v", err)
	}
	defer client.Disconnect(50)

	p := steering.NewController(
		client,
		steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic, objectsTopic,
		steering.WithCorrector(
			steering.NewGridCorrector(
				steering.WidthDeltaMiddle(deltaMiddle),
				steering.WithGridMap(gridMapConfig),
				steering.WithObjectMoveFactors(objectsMoveFactorsConfig),
			),
		),
		steering.WithObjectsCorrectionEnabled(enableObjectsCorrection, enableObjectsCorrectionOnUserMode),
	)
	defer p.Stop()

	cli.HandleExit(p)

	err = p.Start()
	if err != nil {
		zap.S().Fatalf("unable to start service: %v", err)
	}
}
