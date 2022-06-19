package quizbot

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/TopiSenpai/MusicQuizBot/db"
	"github.com/disgoorg/log"
	"github.com/disgoorg/snowflake/v2"
)

func LoadConfig() (*Config, error) {
	file, err := os.Open("config.json")
	if os.IsNotExist(err) {
		if file, err = os.Create("config.json"); err != nil {
			return nil, err
		}
		var data []byte
		if data, err = json.Marshal(Config{}); err != nil {
			return nil, err
		}
		if _, err = file.Write(data); err != nil {
			return nil, err
		}
		return nil, errors.New("config.json not found, created new one")
	} else if err != nil {
		return nil, err
	}

	var cfg Config
	if err = json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(config Config) error {
	file, err := os.OpenFile("config.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Sync()
		_ = file.Close()
	}()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}

type (
	Config struct {
		DevMode      bool          `json:"dev_mode"`
		GuildID      snowflake.ID  `json:"guild_id"`
		LogLevel     log.Level     `json:"log_level"`
		Token        string        `json:"token"`
		SyncCommands bool          `json:"sync_commands"`
		Database     db.Config     `json:"database"`
		Spotify      SpotifyConfig `json:"spotify"`
	}
	SpotifyConfig struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
)
