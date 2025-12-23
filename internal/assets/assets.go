package assets

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg" // Added for JPEG decoder registration
	"io"
	"net/http"
	"time"

	"github.com/HugoSmits86/nativewebp"
	_ "golang.org/x/image/webp" // Added for WebP decoder registration
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

// ConvertToWebP converts image data (expected to be PNG, but supports others) to WebP format using pure Go encoder.
func ConvertToWebP(data []byte) ([]byte, error) {
	// Register formats to ensure we can decode whatever we got (some "png" urls might actually redirect to other formats, or be mislabeled)
	// image/png is imported, so registered.
	// Let's also ensure we import others if needed, but for now assuming input might be missing headers or weird.
	// Actually, the error "image: unknown format" usually means the magic bytes weren't recognized.
	// Importing _ "image/jpeg" and _ "golang.org/x/image/webp" in a loop or init is better practice but here we can just ensure imports.

	img, _, err := image.Decode(bytes.NewReader(data))

	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	var buf bytes.Buffer
	// nativewebp.Encode requires options. Passing nil usually defaults, but let's be explicit if needed.
	// Based on error: want (io.Writer, image.Image, *nativewebp.Options)
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return buf.Bytes(), nil
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
