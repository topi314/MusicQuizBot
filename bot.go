package quizbot

import (
	"context"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/sharding"
	"github.com/disgoorg/log"
)

func New(cfg Config, logger log.Logger) (*Bot, error) {
	b := &Bot{
		Logger:   logger,
		Config:   cfg,
		Commands: make(map[string]Command),
		Players:  NewQuizPlayers(),
		Spotify:  NewSpotify(cfg.Spotify, logger),
	}

	client, err := disgo.New(cfg.Token,
		bot.WithLogger(logger),
		bot.WithShardManagerConfigOpts(
			sharding.WithGatewayConfigOpts(
				gateway.WithGatewayIntents(discord.GatewayIntentGuilds|discord.GatewayIntentGuildVoiceStates),
			),
		),
		bot.WithCacheConfigOpts(
			cache.WithCacheFlags(cache.FlagGuilds|cache.FlagVoiceStates),
		),
		bot.WithEventListenerFunc(b.OnApplicationCommandInteractionCreate),
	)
	if err != nil {
		return nil, err
	}
	b.Client = client

	return b, nil
}

type Bot struct {
	Logger   log.Logger
	Config   Config
	Client   bot.Client
	Commands map[string]Command
	Players  *QuizPlayers
	Spotify  *Spotify
}

func (b *Bot) Start() error {
	return b.Client.ConnectShardManager(context.TODO())
}

func (b *Bot) Close(ctx context.Context) {
	b.Client.Close(ctx)
}
