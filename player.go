package quizbot

import (
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/disgoplayer"
	"github.com/disgoorg/disgoplayer/mp3"
	"github.com/disgoorg/snowflake/v2"
)

func NewQuizPlayers() *QuizPlayers {
	return &QuizPlayers{
		players: make(map[snowflake.ID]*QuizPlayer),
	}
}

type QuizPlayers struct {
	players map[snowflake.ID]*QuizPlayer
	mu      sync.Mutex
}

func (p *QuizPlayers) Get(guildID snowflake.ID) *QuizPlayer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if player, ok := p.players[guildID]; ok {
		return player
	}
	return nil
}

func (p *QuizPlayers) New(guildID snowflake.ID, client bot.Client, conn voice.Connection) *QuizPlayer {
	p.mu.Lock()
	defer p.mu.Unlock()

	player := &QuizPlayer{
		client: client,
		conn:   conn,
		Queue:  NewQuizQueue(),
	}

	p.players[guildID] = player

	return player
}

type QuizPlayer struct {
	client       bot.Client
	conn         voice.Connection
	writer       io.Writer
	Queue        *QuizQueue
	CurrentTrack *QuizTrack
	// TODO: hold quiz state
}

func (p *QuizPlayer) Play(errorHandler func(err error)) {
	mp3Decoder, err := mp3.CreateDecoder()
	if err != nil {
		errorHandler(err)
		return
	}

	pcmFrameProvider, writer, err := disgoplayer.NewMP3PCMFrameProvider(mp3Decoder)
	if err != nil {
		errorHandler(err)
		return
	}
	p.writer = writer

	opusFrameProvider, err := disgoplayer.NewPCMOpusProvider(nil, pcmFrameProvider)
	if err != nil {
		errorHandler(err)
		return
	}

	playerOpusProvider := disgoplayer.NewErrorHandlerOpusFrameProvider(opusFrameProvider, func(err error) {
		if err == disgoplayer.OpusProviderClosed {
			p.conn.Close()
			return
		}
		// io.EOF means the mp3 track is done, so we try to play the next one
		if err == io.EOF {
			// close old mp3 decoder
			//_ = mp3Decoder.Close() closing the old somehow produces a lot of errors - figure out why
			// create a new mp3 decoder to avoid frankenstein errors
			mp3Decoder, err = mp3.CreateDecoder()
			if err != nil {
				errorHandler(err)
				return
			}
			// try to play the next track
			p.PlayNext(errorHandler)
			return
		}
		// send error to error handler
		errorHandler(err)
		return
	})
	p.conn.SetOpusFrameProvider(playerOpusProvider)
}

func (p *QuizPlayer) PlayNext(errorHandler func(err error)) {
	track, ok := p.Queue.Pop()
	if !ok {
		// TODO: queue is empty stop quiz
		_ = p.client.DisconnectVoice(context.Background(), p.conn.GuildID())
		return
	}

	rs, err := http.Get(track.URL)
	if err != nil {
		// something went wrong, send error to error handler. play next?
		errorHandler(err)
		return
	}

	// copy mp3 data into mp3 decoder
	_, _ = io.Copy(p.writer, rs.Body)
	_ = rs.Body.Close()
	p.CurrentTrack = &track
}

func NewQuizQueue() *QuizQueue {
	return &QuizQueue{
		tracks: []QuizTrack{},
	}
}

type QuizQueue struct {
	tracks []QuizTrack
	mu     sync.Mutex
}

func (q *QuizQueue) Pop() (QuizTrack, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.tracks) == 0 {
		return QuizTrack{}, false
	}
	var track QuizTrack
	track, q.tracks = q.tracks[0], q.tracks[1:]
	return track, true
}

func (q *QuizQueue) Push(tracks ...QuizTrack) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tracks = append(q.tracks, tracks...)
}

type QuizTrack struct {
	ID     string
	Name   string
	Artist string
	Image  *string
	URL    string
}
