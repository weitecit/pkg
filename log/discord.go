package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

type HookChannel string

var (
	HookChannelNone   HookChannel = ""
	HookChannelPro    HookChannel = HookChannel(os.Getenv("DISCORD_CHANNEL_PRO"))
	HookChannelDemo   HookChannel = HookChannel(os.Getenv("DISCORD_CHANNEL_DEMO"))
	HookChannelODM    HookChannel = HookChannel(os.Getenv("DISCORD_CHANNEL_ODM"))
	HookChannelLog    HookChannel = HookChannel(os.Getenv("DISCORD_CHANNEL_LOG"))
	HookChannelErr    HookChannel = HookChannel(os.Getenv("DISCORD_CHANNEL_ERR"))
	HookChannelAgtech HookChannel = HookChannel(os.Getenv("DISCORD_CHANNEL_AGTECH"))
)

type Message struct {
	Username        *string          `json:"username,omitempty"`
	AvatarUrl       *string          `json:"avatar_url,omitempty"`
	Content         *string          `json:"content,omitempty"`
	Embeds          *[]Embed         `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
}

type Embed struct {
	Title       *string    `json:"title,omitempty"`
	Url         *string    `json:"url,omitempty"`
	Description *string    `json:"description,omitempty"`
	Color       *string    `json:"color,omitempty"`
	Author      *Author    `json:"author,omitempty"`
	Fields      *[]Field   `json:"fields,omitempty"`
	Thumbnail   *Thumbnail `json:"thumbnail,omitempty"`
	Image       *Image     `json:"image,omitempty"`
	Footer      *Footer    `json:"footer,omitempty"`
}

type Author struct {
	Name    *string `json:"name,omitempty"`
	Url     *string `json:"url,omitempty"`
	IconUrl *string `json:"icon_url,omitempty"`
}

type Field struct {
	Name   *string `json:"name,omitempty"`
	Value  *string `json:"value,omitempty"`
	Inline *bool   `json:"inline,omitempty"`
}

type Thumbnail struct {
	Url *string `json:"url,omitempty"`
}

type Image struct {
	Url *string `json:"url,omitempty"`
}

type Footer struct {
	Text    *string `json:"text,omitempty"`
	IconUrl *string `json:"icon_url,omitempty"`
}

type AllowedMentions struct {
	Parse *[]string `json:"parse,omitempty"`
	Users *[]string `json:"users,omitempty"`
	Roles *[]string `json:"roles,omitempty"`
}

func sendDiscordMessage(channel HookChannel, message Message) error {

	env := os.Getenv("ENVIRONMENT")

	if env == "local" {
		fmt.Println("Discord message not sent in local or test environment: " + *message.Content)
	}

	if env == "local" || env == "test" {
		return nil
	}

	if channel == "" {
		return errors.New("SendMessage: channel is required")
	}

	payload := new(bytes.Buffer)

	err := json.NewEncoder(payload).Encode(message)
	if err != nil {
		return err
	}

	resp, err := http.Post(string(channel), "application/json", payload)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		defer resp.Body.Close()

		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return errors.New(string(responseBody))
	}

	return nil
}

// sample
// func main() {
// 	var username = "BotUser"
// 	var content = "This is a test message"
// 	var url = "https://discord.com/api/webhooks/..."

// 	message := discordwebhook.Message{
// 		Username: &username,
// 		Content: &content,
// 	}

// 	err := discordwebhook.SendMessage(url, message)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//  }
