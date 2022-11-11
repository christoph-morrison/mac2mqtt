# mac2mqtt

`mac2mqtt` is a program that allow viewing and controlling some aspects of computers running macOS via MQTT.

It publish to MQTT:

 * current volume
 * volume mute state
 * battery charge percent

You can send topics to:

 * change volume
 * mute/unmute
 * put computer to sleep
 * shutdown computer
 * turn off display

## Building

Use `make` for creating the binary. It takes care of the libraries and compiling, but not the installation.

## Installing the binary

After building, copy the binary to a location of your choice, for example to your home directory

    cp mac2mqtt $HOME/

## Configuration

### Config file path

Please refer to the sample configure file `mac2mqtt.yml` in `sample/`. Alter it to your needs and move it to a
directory of your choice. The directory where you put the binary `mac2mqtt` with the filename `mac2mqtt.yml` is the
default. 

Start `mac2mqtt` with the parameter `-c /path/to/your/config_file` to change this behavior.

### Config file syntax and options

The config file is a YAML file. The following options are supported:

| Name              | Purpose                                |
|:------------------|:---------------------------------------|
| `mqtt_ip`         | IP or Hostname of your MQTT broker     |
| `mqtt_port`       | Port where your mqtt broker listens to |
| `mqtt_user`       | Username for authentification          |
| `mqtt_password`   | Password for authentication            |
| `mqtt_base_topic` | MQTT topic to listen on                |

There are **no defaults**, mac2mqtt will just bail and exit, if something essential is missing.

## Running in the background

To run `mac2mqtt` in the background, you need to use a [Launch Daemon](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html#//apple_ref/doc/uid/10000172i-SW7-BCIEDDBJ).
In `sample/com.bessarabov.mac2mqtt.plist` there is a sample configuration file. Edit it to your needs and move it to `/Library/LaunchDaemons/`

    sudo cp sample/com.bessarabov.mac2mqtt.plist /Library/LaunchDaemons/

Load the daemon with 

    launchctl load /Library/LaunchDaemons/com.bessarabov.mac2mqtt.plist

Stop it if necessary with

    launchctl unload /Library/LaunchDaemons/com.bessarabov.mac2mqtt.plist

## Home Assistant sample config

![](https://user-images.githubusercontent.com/47263/114361105-753c4200-9b7e-11eb-833c-c26a2b7d0e00.png)

`configuration.yaml`:

```yaml
script:
  air2_sleep:
    icon: mdi:laptop
    sequence:
      - service: mqtt.publish
        data:
          topic: "mac2mqtt/bessarabov-osx/command"
          payload: "sleep"

  air2_shutdown:
    icon: mdi:laptop
    sequence:
      - service: mqtt.publish
        data:
          topic: "mac2mqtt/bessarabov-osx/command"
          payload: "shutdown"

  air2_displaysleep:
    icon: mdi:laptop
    sequence:
      - service: mqtt.publish
        data:
          topic: "mac2mqtt/bessarabov-osx/command"
          payload: "displaysleep"

sensor:
  - platform: mqtt
    name: air2_alive
    icon: mdi:laptop
    state_topic: "mac2mqtt/bessarabov-osx/status/alive"

  - platform: mqtt
    name: "air2_battery"
    icon: mdi:battery-high
    unit_of_measurement: "%"
    state_topic: "mac2mqtt/bessarabov-osx/status/battery"

switch:
  - platform: mqtt
    name: air2_mute
    icon: mdi:volume-mute
    state_topic: "mac2mqtt/bessarabov-osx/status/mute"
    command_topic: "mac2mqtt/bessarabov-osx/command/mute"
    payload_on: "true"
    payload_off: "false"

number:
  - platform: mqtt
    name: air2_volume
    icon: mdi:volume-medium
    state_topic: "mac2mqtt/bessarabov-osx/status/volume"
    command_topic: "mac2mqtt/bessarabov-osx/command/volume"
```

`ui-lovelace.yaml`:

```yaml
title: Home
views:
  - path: default_view
    title: Home
    cards:
      - type: entities
        entities:
          - sensor.air2_alive
          - sensor.air2_battery
          - type: 'custom:slider-entity-row'
            entity: number.air2_volume
            min: 0
            max: 100
          - switch.air2_mute
          - type: button
            name: air2
            entity: script.air2_sleep
            action_name: sleep
            tap_action:
              action: call-service
              service: script.air2_sleep
          - type: button
            name: air2
            entity: script.air2_shutdown
            action_name: shutdown
            tap_action:
              action: call-service
              service: script.air2_shutdown
          - type: button
            name: air2
            entity: script.air2_displaysleep
            action_name: displaysleep
            tap_action:
              action: call-service
              service: script.air2_displaysleep

      - type: history-graph
        hours_to_show: 48
        refresh_interval: 0
        entities:
          - sensor.air2_battery
```

## MQTT topic structure

The program is working with several MQTT topics. All topix are prefixed with the `mqtt_base_topic` from your `mac2mqtt.yaml` + `COMPUTER_NAME`.
Setting the `mqtt_base_topic` to `test/mac2mqtt` the topic will be `test/mac2mqtt/`.

`mac2mqtt` send info to the topics `$mqtt_base_topic/status/#` and listen for commands in topics
`$mqtt_base_topic/command/#`.

### `$mqtt_base_topic` + `/status/alive`

There can be `true` of `false` in this topic. If `mac2mqtt` is connected to MQTT server there is `true`.
If `mac2mqtt` is disconnected from MQTT there is `false`. This is the standard MQTT thing called Last Will and Testament (LWT).

### `$mqtt_base_topic` + `/status/volume`

The value is the numbers from 0 (inclusive) to 100 (inclusive). The current volume of computer.

The value of this topic is updated every 2 seconds.

### `$mqtt_base_topic` + `/status/mute`

There can be `true` of `false` in this topic. `true` means that the computer volume is muted (no sound),
`false` means that it is not multed.

### `$mqtt_base_topic` + `/status/battery`

The value is the number up to 100. The charge percent of the battery. 
If the device is powered with mains voltage, `mains` is issued (i.e. for a Mac Studio)

The value of this topic is updated every 60 seconds.

### `$mqtt_base_topic` + `/command/volume`

You can send integer numberf from 0 (inclusive) to 100 (inclusive) to this topic. It will set the volume on the computer.

### `$mqtt_base_topic` + `/command/mute`

You can send `true` of `false` to this topic. When you send `true` the computer is muted. When you send `false` the computer
is unmuted.

### `$mqtt_base_topic` + `/command/sleep`

You can send string `sleep` to this topic. It will put computer to sleep mode. Sending some other value will do nothing.

### `$mqtt_base_topic` + `/command/shutdown`

You can send string `shutdown` to this topic. It will try to shutdown the computer. The way it is done depends on
the user who run the program. If the program is run by `root` the computer will shutdown, but if it is run by ordinary user
the computer will not shut down if there is other user who logged in.

Sending some other value but `shutdown` will do nothing.

### `$mqtt_base_topic` + `/command/displaysleep`

You can send string `displaysleep` to this topic. It will turn off display. Sending some other value will do nothing.

