package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/TopiSenpai/MusicQuizBot"
	"github.com/TopiSenpai/MusicQuizBot/commands"
	"github.com/disgoorg/log"
)

const version = "development"

func main() {
	cfg, err := quizbot.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	logger := log.New(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(cfg.LogLevel)
	logger.Infof("starting Music Quiz Bot %s...", version)

	bot, err := quizbot.New(*cfg, logger)
	if err != nil {
		logger.Panic("failed to create bot: ", err.Error())
	}
	bot.AddCommands(commands.StartQuizCommand)
	if err = bot.Start(); err != nil {
		logger.Panic("failed to start bot: ", err.Error())
	}

	bot.Logger.Info("Music Quiz Bot is running. Press CTRL-C to exit.")
	defer bot.Close(context.TODO())
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	<-s
}
