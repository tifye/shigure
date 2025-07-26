package discord

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/patrickmn/go-cache"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/mux"
)

const (
	useDefaultCacheTime     = -1
	discordMaxMessageLength = 2000

	siteChatsChannelCacheKey = "site-chats"
)

type ChatBot struct {
	logger *log.Logger

	sesh           *discordgo.Session
	guildID        string
	chatCategoryID string

	mux            *mux.Mux
	muxMessageType string

	cache *cache.Cache
}

func NewChatBot(
	logger *log.Logger,
	token, guildID, chatCategoryID string,
	mx *mux.Mux,
) (*ChatBot, error) {
	assert.AssertNotNil(logger)
	assert.AssertNotEmpty(token)
	assert.AssertNotEmpty(guildID)
	assert.AssertNotEmpty(chatCategoryID)
	assert.AssertNotNil(mx)

	sesh, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}

	b := &ChatBot{
		logger:         logger,
		sesh:           sesh,
		guildID:        guildID,
		chatCategoryID: chatCategoryID,
		mux:            mx,
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

	ch, err := b.channel(ctx, i.ChannelID, channelIDFilter(i.ChannelID))
	if err != nil {
		b.logger.Error("get channel", "err", err, "channelID", i.ChannelID)
		return
	}
	assert.AssertNotNil(ch)
	if ch.ParentID != b.chatCategoryID {
		return
	}

	muxSessionID, err := decodeChannelNameToMuxID(ch.Name)
	if err != nil {
		b.logger.Warn("failed to decode Discord channel name to a mux.ID", "channelName", ch.Name)
		return
	}

	payload, _ := json.Marshal(chatMessage{
		Actor:   "joshua",
		Message: i.Message.Content,
	})
	msg := message{
		Type:    "message",
		Payload: payload,
	}
	msgb, _ := json.Marshal(msg)

	err = b.mux.SendSession(muxSessionID, b.muxMessageType, msgb, nil)
	if err != nil {
		b.logger.Error("send message", "err", err, "channelID", muxSessionID, "msg", string(msgb))
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

type chatMessage struct {
	Actor   string `json:"actor"`
	Message string `json:"message"`
}

type message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (b *ChatBot) HandleMessage(c *mux.Channel, data []byte) error {
	muxID := c.Session().ID()
	b.logger.Debug("message", "id", muxID, "msg", string(data))

	var msg message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal data: %s", err)
	}

	if len(msg.Type) > 30 {
		return fmt.Errorf("message type too long: %d", len(msg.Type))
	}

	if msg.Type != "message" {
		return nil
	}

	var chatMessage chatMessage
	if err := json.Unmarshal(msg.Payload, &chatMessage); err != nil {
		return fmt.Errorf("unmarshal payload: %s", err)
	}

	if chatMessage.Actor != "user" {
		return nil
	}

	if len(chatMessage.Message) > discordMaxMessageLength {
		return fmt.Errorf("message too long, expected at most %d but got %d", discordMaxMessageLength, len(chatMessage.Message))
	}
	if len(chatMessage.Message) == 0 {
		return fmt.Errorf("no message sent")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := b.mux.SendSession(muxID, b.muxMessageType, data, func(ch *mux.Channel) bool {
		return c.ID() == ch.ID()
	})
	if err != nil {
		b.logger.Error("failed to session to other channels", "muxID", muxID, "msg", string(data))
	}

	err = b.sendToUserChat(ctx, muxID, chatMessage.Message, false)
	if err != nil {
		b.logger.Error("failed to forward user message", "err", err, "muxID", muxID)
	}

	return nil
}

func (b *ChatBot) sendToUserChat(ctx context.Context, muxID mux.ID, msg string, isSystem bool) error {
	assert.AssertNotEmpty(msg)

	if isSystem {
		msg = systemPrefix(muxID) + msg
	}

	userCh, err := b.userChannel(ctx, muxID)
	if err != nil {
		return err
	}

	_, err = b.sesh.ChannelMessageSend(userCh.ID, msg, discordgo.WithContext(ctx))
	if err != nil {
		var apiErr *discordgo.RESTError
		if !errors.As(err, &apiErr) {
			return err
		}

		b.cache.Delete(userChannelCacheKey(muxID))
		_, err = b.userChannel(ctx, muxID)
		if err != nil {
			return err
		}

		_, err = b.sesh.ChannelMessageSend(userCh.ID, msg, discordgo.WithContext(ctx))
	}

	return err
}

func (b *ChatBot) HandleMuxChatSubscription(c *mux.Channel, typ mux.MessageType, didSub bool) {
	assert.Assert(typ == b.muxMessageType, "expected only 'chat' message type")
	assert.AssertNotNil(c)

	muxID := c.Session().ID()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if didSub {
		b.replayChat(ctx, c)
	} else {
		err := b.sendToUserChat(ctx, muxID, "User disconnected.", true)
		if err != nil {
			b.logger.Warn("failed to send user disconnect messsage", "err", err)
			return
		}
	}
}

func (b *ChatBot) replayChat(ctx context.Context, c *mux.Channel) {
	assert.AssertNotNil(c)

	muxID := c.Session().ID()
	userCh, err := b.userChannel(ctx, muxID)
	if err != nil {
		return
	}

	replayAmount := 25
	// 100 is limit
	channelMsgs, err := b.sesh.ChannelMessages(userCh.ID, 100, "", "", "", discordgo.WithContext(ctx))
	if err != nil {
		return
	}
	if len(channelMsgs) == 0 {
		return
	}

	chatMsgs := make([]chatMessage, 0, replayAmount)
	for _, m := range channelMsgs {
		if isSystemMessage(m, muxID) {
			continue
		}

		if m.Author.Bot {
			chatMsgs = append(chatMsgs, chatMessage{
				Actor:   "user",
				Message: m.Content,
			})
		} else {
			chatMsgs = append(chatMsgs, chatMessage{
				Actor:   "joshua",
				Message: m.Content,
			})
		}
	}

	// Discord returns messages from latest to oldest.
	// Reverse to match the order in which messages
	// were sent.
	slices.Reverse(chatMsgs)

	payload, _ := json.Marshal(chatMsgs)
	data, err := json.Marshal(message{
		Type:    "replay",
		Payload: payload,
	})
	if err != nil {
		b.logger.Error("marshal chat replay payload", "err", err, "muxID", c.Session().ID())
		return
	}

	err = b.mux.SendChannel(c.ID(), b.muxMessageType, data)
	if err != nil {
		b.logger.Error("chat replay send channel", "err", err)
	}
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

func (b *ChatBot) userChannel(ctx context.Context, muxID mux.ID) (*discordgo.Channel, error) {
	cacheKey := userChannelCacheKey(muxID)
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
		ch, err = b.createChannel(ctx, cacheKey, siteChatsCh.ID)
		if err != nil {
			return nil, err
		}

		assert.AssertNotNil(ch)
		err = b.sendToUserChat(ctx, muxID, "Channel opened.", true)
		if err != nil {
			b.logger.Warn("failed to notify user channel created", "muxID", muxID)
		}
	}

	b.cache.Set(cacheKey, ch, useDefaultCacheTime)

	assert.AssertNotNil(ch)
	assert.Assert(ch.Type == discordgo.ChannelTypeGuildText, "expected channel of type GuildText")
	return ch, nil
}

func (b *ChatBot) createChannel(ctx context.Context, name, parentID string) (*discordgo.Channel, error) {
	assert.AssertNotEmpty(name)

	ch, err := b.sesh.GuildChannelCreateComplex(b.guildID, discordgo.GuildChannelCreateData{
		Name:     name,
		ParentID: parentID,
	}, discordgo.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("create channel: %s", err)
	}

	assert.AssertNotNil(ch)
	b.logger.Info("created channel", "name", name, "parentID", parentID)
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

func userChannelCacheKey(id mux.ID) string {
	return encodeMuxIDToChannelName(id)
}

func encodeMuxIDToChannelName(input mux.ID) string {
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	encoded := encoder.EncodeToString(input[:])
	encoded = strings.ToLower(encoded)
	return encoded
}

func decodeChannelNameToMuxID(channelName string) (mux.ID, error) {
	var result mux.ID
	upper := strings.ToUpper(channelName) // base32 expects uppercase
	decoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	decoded, err := decoder.DecodeString(upper)
	if err != nil {
		return result, err
	}
	if len(decoded) != 16 {
		return result, fmt.Errorf("invalid length: got %d bytes", len(decoded))
	}
	copy(result[:], decoded)
	return result, nil
}

func isSystemMessage(msg *discordgo.Message, muxID mux.ID) bool {
	return strings.HasPrefix(msg.Content, systemPrefix(muxID))
}

func systemPrefix(muxID mux.ID) string {
	return fmt.Sprintf("`[%s]`\n", encodeMuxIDToChannelName(muxID))
}
