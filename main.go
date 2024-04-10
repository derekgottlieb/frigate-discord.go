package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// DiscordWebhookRequest represents the JSON structure of the Discord webhook request
type DiscordWebhookRequest struct {
	Content   string `json:"content,omitempty"`
	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

func main() {
	envFile := ".env"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		// Do nothing if no .env file found
		log.Println("No .env file found\n")
	} else {
		// Load the .env file
		err := godotenv.Load(envFile)
		if err != nil {
			log.Fatalf("Error loading .env file: %s", err)
		}
	}

	mqtt_broker := os.Getenv("MQTT_BROKER")

	mqtt_opts := mqtt.NewClientOptions().AddBroker(mqtt_broker).SetClientID("hal9000-discord.go")
	mqtt_client := mqtt.NewClient(mqtt_opts)
	if token := mqtt_client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Failed to connect to MQTT broker: %v\n", token.Error())
		os.Exit(1)
	}

	if token := mqtt_client.Subscribe("frigate/events", 0, onFrigateEvent); token.Wait() && token.Error() != nil {
		fmt.Printf("Failed to subscribe to topic: %v\n", token.Error())
		os.Exit(1)
	}
	fmt.Println("Subscribed to topic frigate/events. Waiting for messages...")

	// Wait here until Ctrl-C or other term signal is received
	fmt.Println("Bot is now running. Press Ctrl-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	mqtt_client.Unsubscribe("frigate/events")
	mqtt_client.Disconnect(250)
	fmt.Println("Disconnected from MQTT broker.")
}

func onFrigateEvent(mqtt_client mqtt.Client, message mqtt.Message) {
	fmt.Printf("Received message on topic %s: %s\n", message.Topic(), message.Payload())

	var data map[string]interface{}
	err := json.Unmarshal([]byte(message.Payload()), &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	event_type := data["type"].(string)
	after := data["after"].(map[string]interface{})
	camera := after["camera"].(string)
	label := after["label"].(string)

	fmt.Printf("type %s, camera %s, label %s\n", event_type, camera, label)

	if label != "person" {
		return
	}

	frigate_api := os.Getenv("FRIGATE_API")
	imageURL := frigate_api + "/" + camera + "/" + label + "/snapshot.jpg?bbox=1"

	// Fetch the image from the HTTP endpoint
	resp, err := http.Get(imageURL)
	if err != nil {
		fmt.Printf("Failed to fetch image: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read the image data
	imageData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read image data: %v\n", err)
		return
	}

	// Discord webhook URL
	webhookURL := os.Getenv("DISCORD_WEBHOOK")

	// Prepare the webhook request payload
	requestBody := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(requestBody)

	// Add image file to the multipart request
	imagePart, err := multipartWriter.CreateFormFile("file", camera+"_"+label+".jpg")
	if err != nil {
		fmt.Printf("Failed to create form file: %v\n", err)
		return
	}
	_, err = imagePart.Write(imageData)
	if err != nil {
		fmt.Printf("Failed to write image data to multipart writer: %v\n", err)
		return
	}

	// Close the multipart writer
	multipartWriter.Close()

	// Create HTTP request
	request, err := http.NewRequest("POST", webhookURL, requestBody)
	if err != nil {
		fmt.Printf("Failed to create HTTP request: %v\n", err)
		return
	}
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	// Send HTTP request
	http_client := http.Client{}
	response, err := http_client.Do(request)
	if err != nil {
		fmt.Printf("Failed to send HTTP request: %v\n", err)
		return
	}
	defer response.Body.Close()

	// Read and print response body
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		return
	}
	fmt.Println("Response:", string(responseBody))

}
