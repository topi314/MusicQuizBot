package commands

import (
	"context"
	"math/rand"
	"time"

	"github.com/TopiSenpai/MusicQuizBot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/json"
	"github.com/disgoorg/snowflake/v2"
)

func init() {
	rand.Seed(time.Now().Unix())
}

var StartQuizCommand = quizbot.Command{
	Create: discord.SlashCommandCreate{
		CommandName: "music-quiz",
		Description: "Starts a music quiz",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "start",
				Description: "Starts a new music quiz",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "playlist",
						Description: "The playlist to use",
					},
				},
			},
		},
		DMPermission: false,
	},
	Handler: musicQuizHandler,
}

func musicQuizHandler(b *quizbot.Bot, e *events.ApplicationCommandInteractionCreate) error {
	if data, ok := e.Data.(discord.SlashCommandInteractionData); ok {
		switch *data.SubCommandName {
		case "start":
			return startQuizHandler(b, e)
		}
	}
	return nil
}

func startQuizHandler(b *quizbot.Bot, e *events.ApplicationCommandInteractionCreate) error {
	if b.Players.Get(*e.GuildID()) != nil {
		return e.CreateMessage(discord.MessageCreate{
			Content: "A quiz is already running in this server.",
			Flags:   discord.MessageFlagEphemeral,
		})
	}

	voiceState, ok := b.Client.Caches().VoiceStates().Get(*e.GuildID(), e.User().ID)
	if !ok {
		return e.CreateMessage(discord.MessageCreate{
			Content: "You must be in a voice channel to start a quiz.",
			Flags:   discord.MessageFlagEphemeral,
		})
	}

	match := quizbot.SpotifyRegex.FindStringSubmatch(e.SlashCommandInteractionData().String("playlist"))
	if match == nil {
		return e.CreateMessage(discord.MessageCreate{
			Content: "Unable to parse your spotify link.",
			Flags:   discord.MessageFlagEphemeral,
		})
	}

	if err := e.DeferCreateMessage(false); err != nil {
		return err
	}

	var (
		quizTracks []quizbot.QuizTrack
		err        error
	)

	identifier := match[quizbot.SpotifyRegex.SubexpIndex("identifier")]
	switch match[quizbot.SpotifyRegex.SubexpIndex("type")] {
	case "playlist":
		var tracks []quizbot.PlaylistTrack
		tracks, err = b.Spotify.GetPlaylist(identifier)
		for _, track := range tracks {
			if track.Track.PreviewURL == nil {
				continue
			}
			quizTracks = append(quizTracks, quizbot.QuizTrack{
				ID:     track.Track.ID,
				Name:   track.Track.Name,
				Artist: track.Track.Artists[0].Name,
				Image:  track.Track.Album.Images[0].URL,
				URL:    *track.Track.PreviewURL,
			})
		}

	default:
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), e.Token(), discord.MessageUpdate{
			Content: json.NewPtr("Unsupported link type."),
		})
		return err
	}

	if err != nil || len(quizTracks) == 0 {
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), e.Token(), discord.MessageUpdate{
			Content: json.NewPtr("Error getting playlist tracks."),
		})
		return err
	}

	go startQuiz(b, *voiceState.ChannelID, e, quizTracks)
	return nil
}

func startQuiz(b *quizbot.Bot, channelID snowflake.ID, e *events.ApplicationCommandInteractionCreate, quizTracks []quizbot.QuizTrack) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := e.Client().ConnectVoice(ctx, *e.GuildID(), channelID, false, false)
	if err != nil {
		_, _ = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), e.Token(), discord.MessageUpdate{
			Content: json.NewPtr("Error connecting to voice channel."),
		})
		return
	}

	if err = conn.WaitUntilConnected(ctx); err != nil {
		_, _ = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), e.Token(), discord.MessageUpdate{
			Content: json.NewPtr("Error connecting to voice channel."),
		})
		return
	}

	rand.Shuffle(len(quizTracks), func(i, j int) {
		quizTracks[i], quizTracks[j] = quizTracks[j], quizTracks[i]
	})

	player := b.Players.New(*e.GuildID(), e.Client(), conn)

	player.Queue.Push(quizTracks...)

	errorHandler := func(err error) {
		b.Logger.Errorf("Error in quiz: %s", err)
	}
	player.Play(errorHandler)
	// edit interaction response with button for a modal
}
