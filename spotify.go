package quizbot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/log"
)

const spotifyBase = "https://api.spotify.com/v1"

var SpotifyRegex = regexp.MustCompile(`(https?://)?(www\.)?open\.spotify\.com/(user/[\w-]+/)?(?P<type>track|album|playlist|artist)/(?P<identifier>[\w-]+)`)

func NewSpotify(cfg SpotifyConfig, logger log.Logger) *Spotify {
	return &Spotify{
		logger:       logger,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

type Spotify struct {
	logger       log.Logger
	httpClient   *http.Client
	clientID     string
	clientSecret string

	mu      sync.Mutex
	token   string
	expires time.Time
}

func (s *Spotify) formatClientIDSecret() string {
	return base64.StdEncoding.EncodeToString([]byte(s.clientID + ":" + s.clientSecret))
}

func (s *Spotify) requestToken() error {
	form := url.Values{}
	form.Add("grant_type", "client_credentials")
	req, err := http.NewRequest(http.MethodPost, "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Basic "+s.formatClientIDSecret())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		b, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to get token (response status %d)", response.StatusCode)
		}
		return fmt.Errorf("failed to get token (response status %d): %s", response.StatusCode, b)
	}

	var v struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err = json.NewDecoder(response.Body).Decode(&v); err != nil {
		return err
	}

	s.token = v.AccessToken
	s.expires = time.Now().Add(time.Duration(v.ExpiresIn-10) * time.Second)
	return nil
}

func (s *Spotify) GetToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token == "" || s.expires.After(time.Now()) {
		maxRequests := 5
		for err := s.requestToken(); err != nil; err = s.requestToken() {
			maxRequests--
			s.logger.Warn("failed to fetch spotify token: ", err)
			time.Sleep(time.Second)
			if maxRequests == 0 {
				return "", err
			}
		}
	}

	return s.token, nil
}

func (s *Spotify) Do(url string, rsBody any) error {
	rq, err := http.NewRequest(http.MethodGet, spotifyBase+url, nil)
	if err != nil {
		return err
	}
	token, err := s.GetToken()
	if err != nil {
		return err
	}
	rq.Header.Set("Authorization", "Bearer "+token)
	rs, err := s.httpClient.Do(rq)
	if err != nil {
		return err
	}
	if rs.StatusCode != http.StatusOK {
		return HTTPError(rs.StatusCode)
	}
	return json.NewDecoder(rs.Body).Decode(rsBody)
}

func (s *Spotify) GetPlaylist(id string) ([]PlaylistTrack, error) {
	var v Response[PlaylistTrack]
	if err := s.Do("/playlists/"+id+"/tracks?market=US&limit=5&fields=items(track.id,track.album.images,track.artists.name,track.name,track.preview_url)", &v); err != nil {
		return nil, err
	}
	return v.Items, nil
}

type HTTPError int

func (e HTTPError) Error() string {
	return fmt.Sprintf("http code: %d", e)
}

type Response[T any] struct {
	HREF     string  `json:"href"`
	Items    []T     `json:"items"`
	Limit    int     `json:"limit"`
	Next     *string `json:"next"`
	Offset   int     `json:"offset"`
	Previous *string `json:"previous"`
	Total    int     `json:"total"`
}

type PlaylistTrack struct {
	Track Track `json:"track"`
}

type Track struct {
	Album *struct {
		Images []struct {
			URL *string `json:"url"`
		} `json:"images"`
	}
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`

	ID         string  `json:"id"`
	Name       string  `json:"name"`
	PreviewURL *string `json:"preview_url"`
}
