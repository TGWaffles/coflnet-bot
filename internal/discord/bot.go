package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Coflnet/coflnet-bot/internal/metrics"
	"github.com/Coflnet/coflnet-bot/internal/mongo"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

var (
	session    *discordgo.Session
	coflChatId string
)

func InitDiscord() {
	session = getSession()
	session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	coflChatId = os.Getenv("DISCORD_COFLCHAT_ID")

	go ObserveMessages()
	err := session.Open()

	if err != nil {
		log.Error().Err(err).Msgf("error in discord session")
	}
}

func getSession() *discordgo.Session {
	if session != nil {
		return session
	}
	var err error
	log.Info().Msgf("login: %s", "Bot "+os.Getenv("DISCORD_BOT_TOKEN"))
	session, err = discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		log.Error().Err(err).Msgf("error getting discord session")
	}
	return session
}

func ObserveMessages() {
	log.Info().Msgf("adding message handler")
	session.AddHandler(messageCreate)
}

func messageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	log.Info().Msgf("received discord message: %s", m.Content)

	err := mongo.InsertMessage(m.Message)

	if err != nil {
		log.Error().Err(err).Msgf("error when inserting message")
		metrics.ErrorOccurred()
	}

	err = SendMessageToChatApi(m)
	if err != nil {
		log.Error().Err(err).Msgf("error when sending message to chat api")
		metrics.ErrorOccurred()
	}

	metrics.MessageProcessed()
}

type AllowedMentions struct {
	Parse []string `json:"parse"`
}

type WebhookRequest struct {
	Content   string `json:"content"`
	Username  string `json:"username"`
	AvatarUrl string `json:"avatar_url"`
	AllowedMentionsData AllowedMentions `json:"allowed_mentions"`
}

func SendMessageToDiscordChat(message *mongo.ChatMessage) error {

	if message.UUID == "" {
		return fmt.Errorf("no icon url is set")
	}

	iconUrl := fmt.Sprintf("https://crafatar.com/avatars/%s", message.UUID)
	url := os.Getenv("CHAT_WEBHOOK")
	data := &WebhookRequest{
		Content:   message.Message,
		Username:  message.Name,
		AvatarUrl: iconUrl,
		AllowedMentionsData: AllowedMentions{Parse: make([]string, 0)}
	}

	body, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msgf("can not marshal webhook request")
	}

	_, err = http.DefaultClient.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Error().Err(err).Msgf("error sending message to discord chat")
		return err
	}

	return nil
}

func sendInvalidUUIDMessageToDiscord(message *discordgo.Message) {
	_, err := session.ChannelMessageSendReply(message.ChannelID, " minecraft account not found / validated "+message.Author.Username, &discordgo.MessageReference{
		MessageID: message.ID,
		ChannelID: message.ChannelID,
		GuildID:   message.GuildID,
	})

	if err != nil {
		log.Error().Err(err).Msgf("there was an error when sending the message to discord")
	}
}
