#! /bin/bash

go mod init mac2mqtt
go get gopkg.in/yaml.v2
go get github.com/eclipse/paho.mqtt.golang
go build