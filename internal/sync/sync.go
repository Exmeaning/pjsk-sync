package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pjsk-sync/internal/assets"
	"pjsk-sync/internal/config"
	"pjsk-sync/internal/sekai"
)

func msToSec(ms int64) int64 {
	if ms <= 0 {
		return 0
	}
	return ms / 1000
}

func classifyGacha(g sekai.Gacha) (category string, rarity4 *float32, birthday *float32) {
	lowerType := strings.ToLower(g.GachaType)
	if strings.Contains(lowerType, "birthday") {
		return "birthday", nil, nil
	}
	if strings.Contains(lowerType, "fes") || strings.Contains(lowerType, "festival") {
		return "fes", nil, nil
	}

	var r4 *float32
	var rb *float32
	for _, rr := range g.GachaCardRarityRates {
		if rr.CardRarityType == "rarity_4" {
			tmp := rr.Rate
			r4 = &tmp
		}
		if rr.CardRarityType == "rarity_birthday" {
			tmp := rr.Rate
			rb = &tmp
		}
	}

	if rb != nil && *rb > 0 {
		return "birthday", r4, rb
	}
	if r4 != nil {
		if *r4 >= 6.0 {
			return "fes", r4, rb
		}
		if *r4 > 0 {
			return "normal", r4, rb
		}
	}
	return "other", r4, rb
}

func Run(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) error {
	// 1) fetch master
	cards, err := sekai.FetchJSON[[]sekai.Card](ctx, cfg.CardsURL)
	if err != nil {
		return fmt.Errorf("fetch cards: %w", err)
	}
	gachas, err := sekai.FetchJSON[[]sekai.Gacha](ctx, cfg.GachasURL)
	if err != nil {
		return fmt.Errorf("fetch gachas: %w", err)
	}
	events, err := sekai.FetchJSON[[]sekai.Event](ctx, cfg.EventsURL)
	if err != nil {
		return fmt.Errorf("fetch events: %w", err)
	}

	// 2) upsert db
	cardToChar := make(map[int]int, len(cards))
	if err := upsertCards(ctx, pool, cards, cardToChar); err != nil {
		return err
	}
	if err := upsertGachasAndPickups(ctx, pool, gachas, cardToChar); err != nil {
		return err
	}
	if err := upsertEvents(ctx, pool, events); err != nil {
		return err
	}

	log.Printf("db synced: cards=%d gachas=%d events=%d", len(cards), len(gachas), len(events))

	// 3) assets to local image repo (incremental)
	if cfg.DownloadAssets {
		if err := syncAssetsToDir(ctx, cfg, cards, events, gachas); err != nil {
			return err
		}
	}

	return nil
}

func upsertCards(ctx context.Context, pool *pgxpool.Pool, cards []sekai.Card, cardToChar map[int]int) error {
	batch := &pgx.Batch{}
	for _, c := range cards {
		cardToChar[c.ID] = c.CharacterID
		batch.Queue(`
			INSERT INTO pjsk_cards (id, character_id, attr, prefix, rarity, assetbundle_name, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6, now())
			ON CONFLICT (id) DO UPDATE SET
			  character_id=EXCLUDED.character_id,
			  attr=EXCLUDED.attr,
			  prefix=EXCLUDED.prefix,
			  rarity=EXCLUDED.rarity,
			  assetbundle_name=EXCLUDED.assetbundle_name,
			  updated_at=now()
		`, c.ID, c.CharacterID, c.Attr, c.Prefix, c.CardRarityType, c.AssetbundleName)
	}
	br := pool.SendBatch(ctx, batch)
	defer br.Close()
	for range cards {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func upsertGachasAndPickups(ctx context.Context, pool *pgxpool.Pool, gachas []sekai.Gacha, cardToChar map[int]int) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, g := range gachas {
		category, r4, rb := classifyGacha(g)

		_, err := tx.Exec(ctx, `
			INSERT INTO pjsk_gachas
			  (id, gacha_type, name, seq, assetbundle_name, start_at, end_at, pool_category, rarity4_rate, birthday_rate, updated_at)
			VALUES
			  ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now())
			ON CONFLICT (id) DO UPDATE SET
			  gacha_type=EXCLUDED.gacha_type,
			  name=EXCLUDED.name,
			  seq=EXCLUDED.seq,
			  assetbundle_name=EXCLUDED.assetbundle_name,
			  start_at=EXCLUDED.start_at,
			  end_at=EXCLUDED.end_at,
			  pool_category=EXCLUDED.pool_category,
			  rarity4_rate=EXCLUDED.rarity4_rate,
			  birthday_rate=EXCLUDED.birthday_rate,
			  updated_at=now()
		`, g.ID, g.GachaType, g.Name, g.Seq, g.AssetbundleName, msToSec(g.StartAt), msToSec(g.EndAt), category, r4, rb)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `DELETE FROM pjsk_gacha_pickups WHERE gacha_id=$1`, g.ID); err != nil {
			return err
		}

		for _, p := range g.GachaPickups {
			ch := cardToChar[p.CardID]
			var chAny any
			if ch != 0 {
				chAny = ch
			} else {
				chAny = nil
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO pjsk_gacha_pickups (gacha_id, card_id, character_id)
				VALUES ($1,$2,$3)
				ON CONFLICT (gacha_id, card_id) DO UPDATE SET character_id=EXCLUDED.character_id
			`, g.ID, p.CardID, chAny); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func upsertEvents(ctx context.Context, pool *pgxpool.Pool, events []sekai.Event) error {
	batch := &pgx.Batch{}
	for _, e := range events {
		batch.Queue(`
			INSERT INTO pjsk_events
			  (id, event_type, name, assetbundle_name, bgm_assetbundle_name,
			   event_only_component_display_start_at, start_at, aggregate_at, ranking_announce_at,
			   distribution_start_at, event_only_component_display_end_at, closed_at, updated_at)
			VALUES
			  ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, now())
			ON CONFLICT (id) DO UPDATE SET
			  event_type=EXCLUDED.event_type,
			  name=EXCLUDED.name,
			  assetbundle_name=EXCLUDED.assetbundle_name,
			  bgm_assetbundle_name=EXCLUDED.bgm_assetbundle_name,
			  event_only_component_display_start_at=EXCLUDED.event_only_component_display_start_at,
			  start_at=EXCLUDED.start_at,
			  aggregate_at=EXCLUDED.aggregate_at,
			  ranking_announce_at=EXCLUDED.ranking_announce_at,
			  distribution_start_at=EXCLUDED.distribution_start_at,
			  event_only_component_display_end_at=EXCLUDED.event_only_component_display_end_at,
			  closed_at=EXCLUDED.closed_at,
			  updated_at=now()
		`,
			e.ID, e.EventType, e.Name, e.AssetbundleName, e.BgmAssetbundleName,
			msToSec(e.EventOnlyComponentDisplayStart),
			msToSec(e.StartAt),
			msToSec(e.AggregateAt),
			msToSec(e.RankingAnnounceAt),
			msToSec(e.DistributionStartAt),
			msToSec(e.EventOnlyComponentDisplayEnd),
			msToSec(e.ClosedAt),
		)
	}
	br := pool.SendBatch(ctx, batch)
	defer br.Close()
	for range events {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

type assetJob struct {
	destRel string   // 相对 IMAGE_REPO_DIR 的路径
	urls    []string // fallback
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func syncAssetsToDir(ctx context.Context, cfg config.Config, cards []sekai.Card, events []sekai.Event, gachas []sekai.Gacha) error {
	root := cfg.ImageRepoDir
	if root == "" {
		return fmt.Errorf("IMAGE_REPO_DIR is empty")
	}

	dl := assets.NewDownloader()

	var jobs []assetJob

	// cards
	for _, c := range cards {
    	jobs = append(jobs, assetJob{
    	    destRel: fmt.Sprintf("card_thumbnails/%d_normal.webp", c.ID),
    	    urls:    []string{assets.CardNormalURLJP(c.AssetbundleName)},
    	})
    	if c.CardRarityType == "rarity_3" || c.CardRarityType == "rarity_4" {
    	    jobs = append(jobs, assetJob{
    	        destRel: fmt.Sprintf("card_thumbnails/%d_after_training.webp", c.ID),
    	        urls:    []string{assets.CardAfterTrainingURLJP(c.AssetbundleName)},
    	    })
    	}
	}

	// events
	for _, e := range events {
    	jobs = append(jobs, assetJob{
    	    destRel: fmt.Sprintf("sekai-events/event_%d/logo.webp", e.ID),
    	    urls:    []string{assets.EventLogoURLCN(e.AssetbundleName), assets.EventLogoURLJP(e.AssetbundleName)},
    	})
    	jobs = append(jobs, assetJob{
    	    destRel: fmt.Sprintf("sekai-events/event_%d/bg.webp", e.ID),
    	    urls:    []string{assets.EventBgURLCN(e.AssetbundleName), assets.EventBgURLJP(e.AssetbundleName)},
    	})
	}

	// gachas
	// gachas - banner first, fallback to logo if banner returns 404
	for _, g := range gachas {
	    jobs = append(jobs, assetJob{
	        destRel: fmt.Sprintf("sekai-gachas/gacha_%d/banner.webp", g.ID),
    	    urls: []string{
    	        assets.GachaBannerURLCN(g.ID),
    	        assets.GachaBannerURLJP(g.ID),
    	        assets.GachaLogoURLCN(g.ID),  // fallback: logo CN
    	        assets.GachaLogoURLJP(g.ID),  // fallback: logo JP
    	    },
    	})
	}

	sem := make(chan struct{}, cfg.MaxConcurrency)
	var wg sync.WaitGroup

	var mu sync.Mutex
	var downloaded, skipped int

	for _, j := range jobs {
		j := j
		destAbs := filepath.Join(root, j.destRel)

		if fileExists(destAbs) {
			skipped++
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// 逐 URL 尝试
			var content []byte
			var lastStatus int
			for _, u := range j.urls {
				b, status, err := dl.Get(ctx, u)
				lastStatus = status
				if err == nil && status >= 200 && status < 300 && len(b) > 0 {
					content = b
					break
				}
			}

			if content == nil {
				log.Printf("asset miss (download failed): %s last_status=%d", j.destRel, lastStatus)
				return
			}

			if err := writeFileAtomic(destAbs, content); err != nil {
				log.Printf("asset write failed: %s err=%v", j.destRel, err)
				return
			}

			mu.Lock()
			downloaded++
			mu.Unlock()
			log.Printf("asset saved: %s", j.destRel)
		}()
	}

	wg.Wait()
	log.Printf("assets: saved=%d skipped(existing)=%d total=%d", downloaded, skipped, len(jobs))
	return nil
}