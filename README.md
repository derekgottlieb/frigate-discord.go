# frigate-discord.go

App connects to an MQTT broker, listens for new messages on the frigate/events
topic. For events with a label of "person", retrieves latest snapshot for that
camera/label via the frigate API and sends to a Discord webhook.

Env vars:

* DISCORD_WEBHOOK
* FRIGATE_API
* MQTT_BROKER

References:

* https://frigate.video
