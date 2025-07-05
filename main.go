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
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"github.com/tifye/shigure/activity"
	"github.com/tifye/shigure/api"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/personalsite"
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

	deps, err := initDependencies(logger, config)
	if err != nil {
		return fmt.Errorf("init deps: %s", err)
	}

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

func initDependencies(logger *log.Logger, config *viper.Viper) (*api.ServerDependencies, error) {
	youtubeApiKey := config.GetString("YOUTUBE_DATA_API_KEY")
	assert.AssertNotEmpty(youtubeApiKey)

	rh := personalsite.NewRoomHub(logger.WithPrefix("room"))
	go rh.Run()

	return &api.ServerDependencies{
		ActivityClient:       activity.NewClient(logger.WithPrefix("youtube"), youtubeApiKey),
		VSCodeActivityClient: activity.NewVSCodeActivityClient(logger.WithPrefix("vscode")),
		RoomHub:              rh,
	}, nil
}
