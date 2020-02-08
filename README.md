# robocar-steering

Microservice part to manage steering
     
## Usage
```
rc-steering <OPTIONS>
  -mqtt-broker string
        Broker Uri, use MQTT_BROKER env if arg not set (default "tcp://127.0.0.1:1883")
  -mqtt-client-id string
        Mqtt client id, use MQTT_CLIENT_ID env if args not set (default "robocar-steering")
  -mqtt-password string
        Broker Password, MQTT_PASSWORD env if args not set
  -mqtt-qos int
        Qos to pusblish message, use MQTT_QOS env if arg not set
  -mqtt-retain
        Retain mqtt message, if not set, true if MQTT_RETAIN env variable is set
  -mqtt-topic-drive-mode string
        Mqtt topic that contains DriveMode value, use MQTT_TOPIC_DRIVE_MODE if args not set
  -mqtt-topic-rc-steering string
        Mqtt topic that contains RC steering value, use MQTT_TOPIC_RC_STEERING if args not set
  -mqtt-topic-steering string
        Mqtt topic to publish steering result, use MQTT_TOPIC_STEERING if args not set
  -mqtt-topic-tf-steering string
        Mqtt topic that contains tenorflow steering value, use MQTT_TOPIC_TF_STEERING if args not set
  -mqtt-username string
        Broker Username, use MQTT_USERNAME env if arg not set
```

## Docker build

```bash
export DOCKER_CLI_EXPERIMENTAL=enabled
docker buildx build . --platform linux/amd64,linux/arm/7,linux/arm64 -t cyrilix/robocar-steering
```
