package assets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Downloader struct {
	http *http.Client
}

func NewDownloader() *Downloader {
	return &Downloader{http: &http.Client{Timeout: 60 * time.Second}}
}

func (d *Downloader) Get(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "pjsk-sync-action")

	resp, err := d.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}

const BaseURL = "https://assets.unipjsk.com"

func CardNormalURLJP(assetbundle string) string {
	return fmt.Sprintf("%s/startapp/thumbnail/chara/%s_normal.png", BaseURL, assetbundle)
}
func CardAfterTrainingURLJP(assetbundle string) string {
	return fmt.Sprintf("%s/startapp/thumbnail/chara/%s_after_training.png", BaseURL, assetbundle)
}

func EventLogoURLCN(eventAssetbundle string) string {
	return fmt.Sprintf("%s/ondemand/event/%s/logo/logo.png", BaseURL, eventAssetbundle)
}
func EventBgURLCN(eventAssetbundle string) string {
	return fmt.Sprintf("%s/ondemand/event/%s/screen/bg.png", BaseURL, eventAssetbundle)
}
func EventLogoURLJP(eventAssetbundle string) string {
	return fmt.Sprintf("%s/ondemand/event/%s/logo/logo.png", BaseURL, eventAssetbundle)
}
func EventBgURLJP(eventAssetbundle string) string {
	return fmt.Sprintf("%s/ondemand/event/%s/screen/bg.png", BaseURL, eventAssetbundle)
}

func GachaBannerURLCN(gachaID int) string {
	return fmt.Sprintf("%s/startapp/home/banner/banner_gacha%d/banner_gacha%d.png", BaseURL, gachaID, gachaID)
}
func GachaBannerURLJP(gachaID int) string {
	return fmt.Sprintf("%s/startapp/home/banner/banner_gacha%d/banner_gacha%d.png", BaseURL, gachaID, gachaID)
}

// Gacha logo as fallback when banner is not available
func GachaLogoURLCN(gachaID int) string {
	return fmt.Sprintf("%s/ondemand/gacha/ab_gacha_%d/logo/logo.png", BaseURL, gachaID)
}
func GachaLogoURLJP(gachaID int) string {
	return fmt.Sprintf("%s/ondemand/gacha/ab_gacha_%d/logo/logo.png", BaseURL, gachaID)
}
