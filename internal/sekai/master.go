package sekai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Gacha struct {
	ID              int    `json:"id"`
	GachaType       string `json:"gachaType"`
	Name            string `json:"name"`
	Seq             int    `json:"seq"`
	AssetbundleName string `json:"assetbundleName"`
	StartAt         int64  `json:"startAt"` // ms
	EndAt           int64  `json:"endAt"`   // ms

	GachaCardRarityRates []struct {
		CardRarityType string  `json:"cardRarityType"`
		LotteryType    string  `json:"lotteryType"`
		Rate           float32 `json:"rate"`
	} `json:"gachaCardRarityRates"`

	GachaPickups []struct {
		GachaID int `json:"gachaId"`
		CardID  int `json:"cardId"`
	} `json:"gachaPickups"`
}

type Card struct {
	ID              int    `json:"id"`
	CharacterID     int    `json:"characterId"`
	CardRarityType  string `json:"cardRarityType"`
	Attr            string `json:"attr"`
	Prefix          string `json:"prefix"`
	AssetbundleName string `json:"assetbundleName"`
}

type Event struct {
	ID                             int    `json:"id"`
	EventType                      string `json:"eventType"`
	Name                           string `json:"name"`
	AssetbundleName                string `json:"assetbundleName"`
	BgmAssetbundleName             string `json:"bgmAssetbundleName"`
	EventOnlyComponentDisplayStart int64  `json:"eventOnlyComponentDisplayStartAt"`
	StartAt                        int64  `json:"startAt"`
	AggregateAt                    int64  `json:"aggregateAt"`
	RankingAnnounceAt              int64  `json:"rankingAnnounceAt"`
	DistributionStartAt            int64  `json:"distributionStartAt"`
	EventOnlyComponentDisplayEnd   int64  `json:"eventOnlyComponentDisplayEndAt"`
	ClosedAt                       int64  `json:"closedAt"`
}

func FetchJSON[T any](ctx context.Context, url string) (T, error) {
	var zero T

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "pjsk-sync-action")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return zero, fmt.Errorf("fetch %s: status=%d body=%s", url, resp.StatusCode, string(b))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}

	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		return zero, err
	}
	return out, nil
}