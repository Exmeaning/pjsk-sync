package config

import (
	"os"
	"strconv"
)

type Config struct {
	PostgresConnString string
	PGSSLMode          string // require / verify-full 等（可选：自动补进 URL DSN）

	GachasURL string
	CardsURL  string
	EventsURL string

	DownloadAssets bool
	ImageRepoDir   string // 图床仓库被 checkout 到哪个目录
	MaxConcurrency int
}

func Load() Config {
	return Config{
		PostgresConnString: os.Getenv("POSTGRES_CONNECTION_STRING"),
		PGSSLMode:          getenv("PG_SSLMODE", "require"),

		// 修改了默认源至 kotori8823/sekai-sc-master-db
		// 注意：此处使用了 raw.githubusercontent.com 以获取纯文本 JSON
		GachasURL: getenv("GACHAS_URL", "https://raw.githubusercontent.com/kotori8823/sekai-sc-master-db/master/gachas.json"),
		CardsURL:  getenv("CARDS_URL", "https://raw.githubusercontent.com/kotori8823/sekai-sc-master-db/master/cards.json"),
		EventsURL: getenv("EVENTS_URL", "https://raw.githubusercontent.com/kotori8823/sekai-sc-master-db/master/events.json"),

		DownloadAssets: getenvBool("DOWNLOAD_ASSETS", true),
		ImageRepoDir:   getenv("IMAGE_REPO_DIR", "image-hosting"),
		MaxConcurrency: getenvInt("MAX_CONCURRENCY", 6),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvBool(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}
