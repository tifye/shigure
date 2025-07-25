package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"github.com/tifye/shigure/activity"
	"github.com/tifye/shigure/api"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/discord"
	"github.com/tifye/shigure/personalsite"
	"github.com/tifye/shigure/stream"
)

func main() {
	config := viper.New()
	config.AutomaticEnv()

	err := godotenv.Load()
	if err != nil {
		log.Warn("could not load .env file: %s", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger := log.NewWithOptions(os.Stdout, log.Options{
		Level: log.DebugLevel,
	})

	err = run(ctx, logger, config)
	if err != nil {
		logger.Error(err)
	}
}

func run(ctx context.Context, logger *log.Logger, config *viper.Viper) error {
	config.SetDefault("PORT", 6565)
	port := config.GetInt("PORT")

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("net listen: %s", err)
	}

	deps, cfs, err := initDependencies(logger, config)
	if err != nil {
		return fmt.Errorf("init deps: %s", err)
	}
	defer func() {
		if err := cfs.Cleanup(); err != nil {
			logger.Error("cleanup funcs", "err", err)
		}
	}()

	s := api.NewServer(logger, config, deps)
	go func() {
		logger.Printf("serving on %s", ln.Addr())
		err := s.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = s.Shutdown(closeCtx)
	if err != nil {
		return fmt.Errorf("server shutdown: %s", err)
	}

	return nil
}

func initDependencies(logger *log.Logger, config *viper.Viper) (deps *api.ServerDependencies, cfs CleanupFuncs, err error) {
	defer func() {
		if err == nil {
			return
		}

		if ferr := cfs.Cleanup(); ferr != nil {
			err = errors.Join(err, ferr)
		}
	}()

	youtubeApiKey := config.GetString("YOUTUBE_DATA_API_KEY")
	assert.AssertNotEmpty(youtubeApiKey)

	mux := stream.NewMux()

	room := personalsite.NewRoomHubV2(logger.WithPrefix("room-v2"), mux, config.GetString("DISCORD_WEBHOOK_URL"))
	mux.RegisterHandler(room.MessageType(), room.HandleMessage)
	mux.RegisterDisconnectHook(room.HandleDisconnect)

	vsc := activity.NewVSCodeActivityClient(logger.WithPrefix("vscode"), mux)

	discordBot, err := discord.NewChatBot(
		logger.WithPrefix("chatbot"),
		config.GetString("DISCORD_BOT_TOKEN"),
		config.GetString("DISCORD_GUILD_ID"),
		config.GetString("DISCORD_CHAT_CATEGORY_ID"),
		mux,
	)
	if err != nil {
		// todo: be able to start without certain services marking them as "unavailable"
		return nil, cfs, fmt.Errorf("new discord bot: %s", err)
	}
	if err := discordBot.Start(); err != nil {
		return nil, cfs, fmt.Errorf("init discord bot: %s", err)
	}
	cfs.Defer(func() error {
		if err := discordBot.Stop(); err != nil {
			return fmt.Errorf("close discord bot: %s", err)
		}
		return nil
	})
	mux.RegisterHandler(discordBot.MessageType(), discordBot.HandleMessage)

	sessionStore := sessions.NewFilesystemStore("", []byte(config.GetString("OTP_SECRET")))
	newSessionCookie := func(s *sessions.Session) (*http.Cookie, error) {
		val, err := securecookie.EncodeMulti(s.Name(), s.ID, sessionStore.Codecs...)
		if err != nil {
			return nil, err
		}
		return sessions.NewCookie(s.Name(), val, s.Options), nil
	}

	return &api.ServerDependencies{
		ActivityClient:       activity.NewClient(logger.WithPrefix("youtube"), youtubeApiKey),
		VSCodeActivityClient: vsc,
		WSMux:                mux,
		SessionStore:         sessionStore,
		NewSessionCookie:     newSessionCookie,
	}, cfs, nil
}

type CleanupFuncs []func() error

func (cf *CleanupFuncs) Defer(f func() error) {
	*cf = append(*cf, f)
}

func (cf *CleanupFuncs) Cleanup() error {
	errs := make([]error, 0)
	for i := len(*cf) - 1; i >= 0; i-- {
		if ferr := (*cf)[i](); ferr != nil {
			errs = append(errs, ferr)
		}
	}
	return errors.Join(errs...)
}
