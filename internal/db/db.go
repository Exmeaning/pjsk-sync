package db

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, conn string, sslMode string) (*pgxpool.Pool, error) {
	if conn == "" {
		return nil, fmt.Errorf("POSTGRES_CONNECTION_STRING is empty")
	}
	conn = ensureSSLMode(conn, sslMode)

	cfg, err := pgxpool.ParseConfig(conn)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

// URL 形式 DSN 自动补 sslmode（postgres:// 或 postgresql://）
func ensureSSLMode(conn, sslMode string) string {
	if sslMode == "" {
		return conn
	}
	if !(strings.HasPrefix(conn, "postgres://") || strings.HasPrefix(conn, "postgresql://")) {
		return conn
	}
	u, err := url.Parse(conn)
	if err != nil {
		return conn
	}
	q := u.Query()
	if q.Get("sslmode") == "" {
		q.Set("sslmode", sslMode)
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// 扩展可能在部分托管库需要更高权限：失败就跳过
	if _, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_trgm;`); err != nil {
		log.Printf("warn: create extension pg_trgm failed (skip trigram indexes): %v", err)
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS pjsk_cards (
			id INT PRIMARY KEY,
			character_id INT NOT NULL,
			attr TEXT NOT NULL,
			prefix TEXT NOT NULL DEFAULT '',
			rarity TEXT NOT NULL,
			assetbundle_name TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_pjsk_cards_character_id ON pjsk_cards(character_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pjsk_cards_assetbundle_name ON pjsk_cards(assetbundle_name);`,

		`CREATE TABLE IF NOT EXISTS pjsk_gachas (
			id INT PRIMARY KEY,
			gacha_type TEXT NOT NULL,
			name TEXT NOT NULL,
			seq INT NOT NULL,
			assetbundle_name TEXT NOT NULL,
			start_at BIGINT NOT NULL,
			end_at BIGINT NOT NULL,
			pool_category TEXT NOT NULL,
			rarity4_rate REAL,
			birthday_rate REAL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_pjsk_gachas_start_end ON pjsk_gachas(start_at, end_at);`,

		`CREATE TABLE IF NOT EXISTS pjsk_gacha_pickups (
			gacha_id INT NOT NULL REFERENCES pjsk_gachas(id) ON DELETE CASCADE,
			card_id INT NOT NULL REFERENCES pjsk_cards(id),
			character_id INT,
			PRIMARY KEY (gacha_id, card_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_pjsk_gacha_pickups_gacha_id ON pjsk_gacha_pickups(gacha_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pjsk_gacha_pickups_character_id ON pjsk_gacha_pickups(character_id);`,

		`CREATE TABLE IF NOT EXISTS pjsk_events (
			id INT PRIMARY KEY,
			event_type TEXT NOT NULL,
			name TEXT NOT NULL,
			assetbundle_name TEXT NOT NULL,
			bgm_assetbundle_name TEXT NOT NULL DEFAULT '',
			event_only_component_display_start_at BIGINT,
			start_at BIGINT NOT NULL,
			aggregate_at BIGINT,
			ranking_announce_at BIGINT,
			distribution_start_at BIGINT,
			event_only_component_display_end_at BIGINT,
			closed_at BIGINT,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_pjsk_events_start_at ON pjsk_events(start_at);`,
	}

	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s); err != nil {
			return err
		}
	}

	// trigram 索引：失败不致命
	if _, err := pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_pjsk_gachas_name_trgm ON pjsk_gachas USING GIN (name gin_trgm_ops);`); err != nil {
		log.Printf("warn: create trigram index on gachas.name failed: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_pjsk_events_name_trgm ON pjsk_events USING GIN (name gin_trgm_ops);`); err != nil {
		log.Printf("warn: create trigram index on events.name failed: %v", err)
	}

	return nil
}