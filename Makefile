CONFIG_FILE_TARGET	:= $(HOME)/bin/mac2mqtt.yml
.DEFAULT_GOAL   	:= build
export PATH := /opt/local/bin:$(PATH)

build:

	$(MAKE) clean
	go mod init mac2mqtt
	go get gopkg.in/yaml.v2
	go get github.com/eclipse/paho.mqtt.golang
	go build

clean:

	-@rm go.mod go.sum mac2mqtt 2>/dev/null || true