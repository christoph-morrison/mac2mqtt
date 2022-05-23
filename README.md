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
 * logs you out

## Building

Use `build.sh` for creating the binary. It takes care of the libraries and compiling.

## Running

To run this program you need to put 2 files in a directory (`/Users/USERNAME/mac2mqtt/`):

    mac2mqtt
    mac2mqtt.yaml

Edit `mac2mqtt.yaml` (the sample file `mac2mqtt.yaml.sample` is in this repository), make binary executable (`chmod +x mac2mqtt`) and run `./mac2mqtt`:

    $ ./mac2mqtt
    2021/04/12 10:37:28 Started
    2021/04/12 10:37:29 Connected to MQTT
    2021/04/12 10:37:29 Sending 'true' to topic: mac2mqtt/bessarabov-osx/status/alive

## Running in the background

You need `mac2mqtt.yaml` and `mac2mqtt` to be placed in the directory `/Users/USERNAME/mac2mqtt/`,
then you need to create file `/Library/LaunchDaemons/com.bessarabov.mac2mqtt.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>Label</key>
        <string>com.bessarabov.mac2mqtt</string>
        <key>Program</key>
        <string>/Users/USERNAME/mac2mqtt/mac2mqtt</string>
        <key>WorkingDirectory</key>
        <string>/Users/USERNAME/mac2mqtt/</string>
        <key>RunAtLoad</key>
        <true/>
        <key>KeepAlive</key>
        <true/>
    </dict>
</plist>
```

And run:

    launchctl load /Library/LaunchDaemons/com.bessarabov.mac2mqtt.plist

(To stop you need to run `launchctl unload /Library/LaunchDaemons/com.bessarabov.mac2mqtt.plist`)

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
          topic: "mac2mqtt/bessarabov-osx/command/sleep"
          payload: "sleep"

  air2_shutdown:
    icon: mdi:laptop
    sequence:
      - service: mqtt.publish
        data:
          topic: "mac2mqtt/bessarabov-osx/command/shutdown"
          payload: "shutdown"

  air2_displaysleep:
    icon: mdi:laptop
    sequence:
      - service: mqtt.publish
        data:
          topic: "mac2mqtt/bessarabov-osx/command/displaysleep"
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

## MQTT topics structure

The program is working with several MQTT topics. All topix are prefixed with the `mqtt_base_topic` from your `mac2mqtt.yaml` + `COMPUTER_NAME`.
Setting the `mqtt_base_topic` to `test/mac2mqtt` and the computer hostname is `WOPR`, the topic will be `test/mac2mqtt/WOPR`.

`mac2mqtt` send info to the topics `PREFIX/COMPUTER_NAME/status/#` and listen for commands in topics
`PREFIX/COMPUTER_NAME/command/#`.

### PREFIX + `/status/alive`

There can be `true` of `false` in this topic. If `mac2mqtt` is connected to MQTT server there is `true`.
If `mac2mqtt` is disconnected from MQTT there is `false`. This is the standard MQTT thing called Last Will and Testament (LWT).

### PREFIX + `/status/volume`

The value is the numbers from 0 (inclusive) to 100 (inclusive). The current volume of computer.

The value of this topic is updated every 2 seconds.

### PREFIX + `/status/mute`

There can be `true` of `false` in this topic. `true` means that the computer volume is muted (no sound),
`false` means that it is not multed.

### PREFIX + `/status/battery`

The value is the nuber up to 100. The charge percent of the battery.

The value of this topic is updated every 60 seconds.

### PREFIX + `/command/volume`

You can send integer numberf from 0 (inclusive) to 100 (inclusive) to this topic. It will set the volume on the computer.

### PREFIX + `/command/mute`

You can send `true` of `false` to this topic. When you send `true` the computer is muted. When you send `false` the computer
is unmuted.

### PREFIX + `/command/sleep`

You can send string `sleep` to this topic. It will put computer to sleep mode. Sending some other value will do nothing.

### PREFIX + `/command/shutdown`

You can send string `shutdown` to this topic. It will try to shutdown the computer. The way it is done depends on
the user who run the program. If the program is run by `root` the computer will shutdown, but if it is run by ordinary user
the computer will not shut down if there is other user who logged in.

Sending some other value but `shutdown` will do nothing.

### PREFIX + `/command/displaysleep`

You can send string `displaysleep` to this topic. It will turn off display. Sending some other value will do nothing.


### PREFIX + `/command/afk`

Lock the screen by sending `afk` to this topic.