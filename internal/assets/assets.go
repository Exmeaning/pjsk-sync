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

func CardNormalURLJP(assetbundle string) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-jp-assets/thumbnail/chara/%s_normal.webp", assetbundle)
}
func CardAfterTrainingURLJP(assetbundle string) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-jp-assets/thumbnail/chara/%s_after_training.webp", assetbundle)
}

func EventLogoURLCN(eventAssetbundle string) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-cn-assets/event/%s/logo/logo.webp", eventAssetbundle)
}
func EventBgURLCN(eventAssetbundle string) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-cn-assets/event/%s/screen/bg.webp", eventAssetbundle)
}
func EventLogoURLJP(eventAssetbundle string) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-jp-assets/event/%s/logo/logo.webp", eventAssetbundle)
}
func EventBgURLJP(eventAssetbundle string) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-jp-assets/event/%s/screen/bg.webp", eventAssetbundle)
}

func GachaBannerURLCN(gachaID int) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-cn-assets/home/banner/banner_gacha%d/banner_gacha%d.webp", gachaID, gachaID)
}
func GachaBannerURLJP(gachaID int) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-jp-assets/home/banner/banner_gacha%d/banner_gacha%d.webp", gachaID, gachaID)
}

// Gacha logo as fallback when banner is not available
func GachaLogoURLCN(gachaID int) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-cn-assets/gacha/ab_gacha_%d/logo/logo.webp", gachaID)
}
func GachaLogoURLJP(gachaID int) string {
	return fmt.Sprintf("https://storage.sekai.best/sekai-jp-assets/gacha/ab_gacha_%d/logo/logo.webp", gachaID)
}