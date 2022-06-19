package db

type Config struct {
	Address  string `json:"address"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Insecure bool   `json:"insecure"`
	Verbose  bool   `json:"verbose"`
}
