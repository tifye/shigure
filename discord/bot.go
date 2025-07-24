package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/stream"
)

type ChatBot struct {
	logger *log.Logger

	sesh           *discordgo.Session
	guildID        string
	chatCategoryID string

	mux            *stream.Mux
	muxMessateType string
}

func NewChatBot(
	logger *log.Logger,
	token, guildID, chatCategoryID string,
	mux *stream.Mux,
) (*ChatBot, error) {
	assert.AssertNotNil(logger)
	assert.AssertNotEmpty(token)
	assert.AssertNotEmpty(guildID)
	assert.AssertNotEmpty(chatCategoryID)
	assert.AssertNotNil(mux)

	sesh, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}

	return &ChatBot{
		logger:         logger,
		sesh:           sesh,
		guildID:        guildID,
		chatCategoryID: chatCategoryID,
		mux:            mux,
		muxMessateType: "chat",
	}, nil
}

func (b *ChatBot) MessageType() string {
	return b.muxMessateType
}

func (b *ChatBot) Start() error {
	return b.sesh.Open()
}

func (b *ChatBot) Stop() error {
	return b.sesh.Close()
}

func (b *ChatBot) HandleMessage(id stream.ID, data []byte) error {
	b.logger.Debug("message", "id", id, "msg", string(data))
	return nil
}
