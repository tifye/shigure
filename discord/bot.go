package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/patrickmn/go-cache"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/stream"
)

const (
	useDefaultCacheTime     = -1
	discordMaxMessageLength = 2000

	siteChatsChannelCacheKey  = "site-chats"
	userChannelCacheKeyPrefix = "user-"
)

type ChatBot struct {
	logger *log.Logger

	sesh           *discordgo.Session
	guildID        string
	chatCategoryID string

	mu             sync.RWMutex
	userChannelIDs map[string]stream.ID

	mux            *stream.Mux
	muxMessageType string

	cache *cache.Cache
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

	b := &ChatBot{
		logger:         logger,
		sesh:           sesh,
		guildID:        guildID,
		chatCategoryID: chatCategoryID,
		userChannelIDs: map[string]stream.ID{},
		mux:            mux,
		muxMessageType: "chat",
		cache:          cache.New(30*time.Minute, 60*time.Minute),
	}

	sesh.AddHandler(b.handleDiscordMessage)

	return b, nil
}

func (b *ChatBot) handleDiscordMessage(s *discordgo.Session, i *discordgo.MessageCreate) {
	if i.Author.Bot {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	b.mu.RLock()
	userID, ok := b.userChannelIDs[i.ChannelID]
	b.mu.RUnlock()

	if !ok {
		return
	}

	ch, err := b.channel(ctx, i.ChannelID, channelIDFilter(i.ChannelID))
	if err != nil {
		b.logger.Error("get channel", "err", err, "channelID", i.ChannelID)
		return
	}
	assert.AssertNotNil(ch)

	msg := message{
		Type: "message",
		Payload: messagePayload{
			Message: i.Message.Content,
		},
	}
	msgb, err := json.Marshal(msg)
	if err != nil {
		b.logger.Error("marshal message", "err", err, "msg", msg)
	}

	err = b.mux.SendMessage(userID, b.muxMessageType, msgb)
	if err != nil {
		b.logger.Error("send message", "err", err, "id", userID, "msg", string(msgb))
	}
}

func (b *ChatBot) MessageType() string {
	return b.muxMessageType
}

func (b *ChatBot) Start() error {
	return b.sesh.Open()
}

func (b *ChatBot) Stop() error {
	return b.sesh.Close()
}

type messagePayload struct {
	Message string `json:"message"`
}

type message struct {
	Type    string         `json:"type"`
	Payload messagePayload `json:"payload"`
}

func (b *ChatBot) HandleMessage(id stream.ID, data []byte) error {
	b.logger.Debug("message", "id", id, "msg", string(data))

	var msg message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal data: %s", err)
	}

	if len(msg.Type) > 30 {
		return fmt.Errorf("message type too long: %d", len(msg.Type))
	}
	if len(msg.Payload.Message) > discordMaxMessageLength {
		return fmt.Errorf("message too long, expected at most %d but got %d", discordMaxMessageLength, len(msg.Payload.Message))
	}
	if len(msg.Payload.Message) == 0 {
		return fmt.Errorf("no message sent")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userCh, err := b.userChannel(ctx, id)
	if err != nil {
		return err
	}

	b.mu.Lock()
	b.userChannelIDs[userCh.ID] = id
	b.mu.Unlock()

	_, err = b.sesh.ChannelMessageSend(userCh.ID, msg.Payload.Message, discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("send message: %s", err)
	}

	return nil
}

func (b *ChatBot) siteChatsChannel(ctx context.Context) (*discordgo.Channel, error) {
	ch, err := b.channel(ctx, siteChatsChannelCacheKey, channelNameFilter("Site chats"))
	if err != nil {
		return nil, err
	}
	assert.AssertNotNil(ch)
	assert.Assert(ch.Type == discordgo.ChannelTypeGuildCategory, "expected channel of type GuildCategory")
	return ch, nil
}

func (b *ChatBot) userChannel(ctx context.Context, id stream.ID) (*discordgo.Channel, error) {
	cacheKey := userChannelCacheKey(id)
	ch, err := b.channel(ctx, cacheKey, channelNameFilter(cacheKey))
	if err != nil {
		return nil, err
	}

	siteChatsCh, err := b.siteChatsChannel(ctx)
	if err != nil {
		return nil, err
	}
	assert.Assert(siteChatsCh.Type == discordgo.ChannelTypeGuildCategory, "expected channel of type GuildCategory")

	if ch == nil {
		ch, err = b.sesh.GuildChannelCreateComplex(b.guildID, discordgo.GuildChannelCreateData{
			Name:     cacheKey,
			ParentID: siteChatsCh.ID,
		}, discordgo.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("create channel: %s", err)
		}
	}

	assert.Assert(ch.Type == discordgo.ChannelTypeGuildText, "expected channel of type GuildText")
	return ch, nil
}

func (b *ChatBot) channel(ctx context.Context, cacheKey string, filter func(c *discordgo.Channel) bool) (*discordgo.Channel, error) {
	pch, exists := b.cache.Get(cacheKey)
	if exists {
		ch, ok := pch.(*discordgo.Channel)
		assert.Assert(ok, "expected pointer to discordgo.Channel type")

		b.logger.Debug("cache hit on channel", "channelId", ch.ID, "channelName", ch.Name)
		return ch, nil
	}

	chans, err := b.sesh.GuildChannels(b.guildID, discordgo.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	var ch *discordgo.Channel
	for i := range chans {
		if filter(chans[i]) {
			ch = chans[i]
			break
		}
	}

	if ch != nil {
		b.cache.Set(cacheKey, ch, useDefaultCacheTime)
	}

	return ch, nil
}

func channelNameFilter(name string) func(c *discordgo.Channel) bool {
	return func(c *discordgo.Channel) bool {
		return c.Name == name
	}
}

func channelIDFilter(id string) func(c *discordgo.Channel) bool {
	return func(c *discordgo.Channel) bool {
		return c.ID == id
	}
}

func userChannelCacheKey(id stream.ID) string {
	return fmt.Sprintf("%s%d", userChannelCacheKeyPrefix, id)
}
