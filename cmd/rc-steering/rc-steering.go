package main

import (
	"flag"
	"github.com/cyrilix/robocar-base/cli"
	"github.com/cyrilix/robocar-steering/part"
	"log"
	"os"
)

const (
	DefaultClientId = "robocar-steering"
)

func main() {
	var mqttBroker, username, password, clientId string
	var steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic string

	mqttQos := cli.InitIntFlag("MQTT_QOS", 0)
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	cli.InitMqttFlags(DefaultClientId, &mqttBroker, &username, &password, &clientId, &mqttQos, &mqttRetain)

	flag.StringVar(&steeringTopic, "mqtt-topic-steering", os.Getenv("MQTT_TOPIC_STEERING"), "Mqtt topic to publish steering result, use MQTT_TOPIC_STEERING if args not set")
	flag.StringVar(&rcSteeringTopic, "mqtt-topic-rc-steering", os.Getenv("MQTT_TOPIC_RC_STEERING"), "Mqtt topic that contains RC steering value, use MQTT_TOPIC_RC_STEERING if args not set")
	flag.StringVar(&tfSteeringTopic, "mqtt-topic-tf-steering", os.Getenv("MQTT_TOPIC_TF_STEERING"), "Mqtt topic that contains tenorflow steering value, use MQTT_TOPIC_TF_STEERING if args not set")
	flag.StringVar(&driveModeTopic, "mqtt-topic-drive-mode", os.Getenv("MQTT_TOPIC_DRIVE_MODE"), "Mqtt topic that contains DriveMode value, use MQTT_TOPIC_DRIVE_MODE if args not set")

	flag.Parse()
	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	client, err := cli.Connect(mqttBroker, username, password, clientId)
	if err != nil {
		log.Fatalf("unable to connect to mqtt bus: %v", err)
	}
	defer client.Disconnect(50)

	p := part.NewPart(client, steeringTopic, driveModeTopic, rcSteeringTopic, tfSteeringTopic)
	defer p.Stop()

	cli.HandleExit(p)

	err = p.Start()
	if err != nil {
		log.Fatalf("unable to start service: %v", err)
	}
}
