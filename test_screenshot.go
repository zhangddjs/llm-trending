package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

func testScreenshotOnly() error {
	fmt.Println("🧪 Testing screenshot capture only...")
	
	// Create Chrome context with options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1920, 1080),
	)
	
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Generate screenshot filename
	now := time.Now()
	filename := fmt.Sprintf("test_screenshot_%s.png", now.Format("2006_01_02_15_04_05"))
	screenshotPath := filepath.Join("data", filename)

	var buf []byte
	
	fmt.Println("🌐 Navigating to OpenRouter rankings page...")
	
	// Navigate and capture screenshot
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://openrouter.ai/rankings"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(5*time.Second), // Wait for content to load
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("📸 Taking screenshot...")
			return chromedp.FullScreenshot(&buf, 90).Do(ctx)
		}),
	)

	if err != nil {
		return fmt.Errorf("failed to capture screenshot: %w", err)
	}

	if len(buf) == 0 {
		return fmt.Errorf("screenshot buffer is empty")
	}

	// Save screenshot to file
	if err := os.WriteFile(screenshotPath, buf, 0644); err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}

	fmt.Printf("✅ Test screenshot saved to: %s (%d bytes)\n", screenshotPath, len(buf))
	return nil
}

func main() {
	if err := testScreenshotOnly(); err != nil {
		fmt.Printf("❌ Screenshot test failed: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("🎉 Screenshot test passed!")
}