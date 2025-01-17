package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"flag"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var hostname string
var basetopic string
var debug bool
var config_file string = "mac2mqtt.yml"

type config struct {
	Ip       	string `yaml:"mqtt_ip"`
	Port     	string `yaml:"mqtt_port"`
	User     	string `yaml:"mqtt_user"`
	Password 	string `yaml:"mqtt_password"`
	BaseTopic 	string `yaml:"mqtt_base_topic"`
}

func (c *config) getConfig(path string) *config {

	configContent, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(configContent, c)
	if err != nil {
		log.Fatal(err)
	}

	if c.Ip == "" {
		log.Fatal("Must specify mqtt_ip in mac2mqtt.yaml")
	}

	if c.Port == "" {
		log.Fatal("Must specify mqtt_port in mac2mqtt.yaml")
	}

	if c.BaseTopic == "" {
		log.Fatal("No MQTT base topic set, set mqtt_base_topic in mac2mqtt.yaml")
	}

	if c.Password == "" && debug {
	    log.Print("No password for MQTT used, consider to use username and password for security reasons.")
	}

	return c
}

func getHostname() string {

	hostname, err := os.Hostname()

	if err != nil {
		log.Fatal(err)
	}

	// "name.local" => "name"
	firstPart := strings.Split(hostname, ".")[0]

	// remove all symbols, but [a-zA-Z0-9_-]
	reg, err := regexp.Compile("[^a-zA-Z0-9_-]+")
	if err != nil {
		log.Fatal(err)
	}
	firstPart = reg.ReplaceAllString(firstPart, "")

	return firstPart
}

func getCommandOutput(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)

	stdout, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	stdoutStr := string(stdout)
	stdoutStr = strings.TrimSuffix(stdoutStr, "\n")

	return stdoutStr
}

func getMuteStatus() bool {
	output := getCommandOutput("/usr/bin/osascript", "-e", "output muted of (get volume settings)")

	b, err := strconv.ParseBool(output)
	if err != nil {
		log.Fatal(err)
	}

	return b
}

func getCurrentVolume() int {
	output := getCommandOutput("/usr/bin/osascript", "-e", "output volume of (get volume settings)")

	i, err := strconv.Atoi(output)
	if err != nil {
		log.Fatal(err)
	}

	return i
}

func runCommand(name string, arg ...string) {
	cmd := exec.Command(name, arg...)

	if (debug) {log.Printf("Executing command: '%s'\n", cmd)}

	o, err := cmd.Output()

	if (debug)  {
	    log.Printf("Command output: '%s'\n", o)
    }

    if (err != nil) {
        log.Fatalln(err)
    }
}

// from 0 to 100
func setVolume(i int) {
	runCommand("/usr/bin/osascript", "-e", "set volume output volume "+strconv.Itoa(i))
}

// true - turn mute on
// false - turn mute off
func setMute(b bool) {
	runCommand("/usr/bin/osascript", "-e", "set volume output muted "+strconv.FormatBool(b))
}

func commandSleep() {
	runCommand("pmset", "sleepnow")
}

func commandDisplaySleep() {
	runCommand("pmset", "displaysleepnow")
}

func commandShutdown() {

	if os.Getuid() == 0 {
		// if the program is run by root user we are doing the most powerfull shutdown - that always shuts down the computer
		runCommand("shutdown", "-h", "now")
	} else {
		// if the program is run by ordinary user we are trying to shutdown, but it may fail if the other user is logged in
		runCommand("/usr/bin/osascript", "-e", "tell app \"System Events\" to shut down")
	}

}

func commandAfk() {
    commandDisplaySleep()
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
    log.Println("Connected to MQTT")

	token := client.Publish(getTopicPrefix()+"/status/alive", 0, true, "true")
	token.Wait()

    if (debug) {
        log.Println("Sending 'true' to topic: " + getTopicPrefix() + "/status/alive")
    }

	listen(client, getTopicPrefix()+"/command/#")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Disconnected from MQTT: %v", err)
}

func getMQTTClient(ip, port, user, password string) mqtt.Client {

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%s", ip, port))
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	opts.SetWill(getTopicPrefix()+"/status/alive", "false", 0, true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return client
}

func getTopicPrefix() string {
	return basetopic
}

func listen(client mqtt.Client, topic string) {

	token := client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {

	    if (debug) {
	        log.Printf( "Message received on topic '%s' with payload '%s'", msg.Topic(), msg.Payload() )
        }

		if msg.Topic() == getTopicPrefix()+"/command/volume" {
			i, err := strconv.Atoi(string(msg.Payload()))
			if err == nil && i >= 0 && i <= 100 {
				setVolume(i)
				updateVolume(client)
				updateMute(client)

			} else {
				log.Println("Incorrect value")
			}
		}

		if msg.Topic() == getTopicPrefix()+"/command/mute" {
			b, err := strconv.ParseBool(string(msg.Payload()))
			if err == nil {
				setMute(b)
				updateVolume(client)
				updateMute(client)

			} else {
				log.Println("Incorrect value")
			}
		}

		if msg.Topic() == getTopicPrefix()+"/command/sleep" {
            commandSleep()
        }

        if msg.Topic() == getTopicPrefix()+"/command/displaysleep" {
            commandDisplaySleep()
        }

		if msg.Topic() == getTopicPrefix()+"/command/shutdown" {
            commandShutdown()
        }

        if msg.Topic() == getTopicPrefix()+"/command/afk" {
            commandAfk()
        }
	})

	token.Wait()
	if token.Error() != nil {
		log.Printf("Token error: %s\n", token.Error())
	}
}

func updateVolume(client mqtt.Client) {
	token := client.Publish(getTopicPrefix()+"/status/volume", 0, false, strconv.Itoa(getCurrentVolume()))
	token.Wait()
}

func updateMute(client mqtt.Client) {
	token := client.Publish(getTopicPrefix()+"/status/mute", 0, false, strconv.FormatBool(getMuteStatus()))
	token.Wait()
}

func getBatteryChargePercent() string {

	output := getCommandOutput("/usr/bin/pmset", "-g", "batt")

    matches_mains, _ := regexp.MatchString(".*AC.Power.*", output)
    if (matches_mains) {
        return "mains"
    }

    rPercent := regexp.MustCompile(`(\d+)%`)
	return rPercent.FindStringSubmatch(output)[1]
}

func updateBattery(client mqtt.Client) {
	token := client.Publish(getTopicPrefix()+"/status/battery", 0, false, getBatteryChargePercent())
	token.Wait()
}

func main() {

	log.Println("Started")

    // parse command line arguments
    flag.StringVar(&config_file, "c", "./mac2mqtt.yml", "Path to configuration file, defaults to ./mac2mqtt.yml")
    flag.BoolVar(&debug, "d", false, "Print debug output, default false")
    flag.Parse()

    if (debug) {
        log.Println("Printing debug information")
        log.Println("Using config file", config_file )
    }

	var c config
	c.getConfig(config_file)

	var wg sync.WaitGroup

	hostname = getHostname()
	basetopic = c.BaseTopic
	mqttClient := getMQTTClient(c.Ip, c.Port, c.User, c.Password)

	volumeTicker := time.NewTicker(2 * time.Second)
	batteryTicker := time.NewTicker(60 * time.Second)

	wg.Add(1)
	go func() {
		for {
			select {
			case _ = <-volumeTicker.C:
				updateVolume(mqttClient)
				updateMute(mqttClient)

			case _ = <-batteryTicker.C:
				updateBattery(mqttClient)
			}
		}
	}()

	wg.Wait()

}
