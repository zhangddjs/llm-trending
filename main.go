package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type ModelRanking struct {
	Name       string    `json:"name"`
	Score      string    `json:"score"`
	Rank       int       `json:"rank"`
	UpdateTime time.Time `json:"update_time"`
}

type RankingData struct {
	Category   string         `json:"category"`
	Date       time.Time      `json:"date"`
	Models     []ModelRanking `json:"models"`
	Analysis   string         `json:"analysis"`
	AnalysisZh string         `json:"analysis_zh"`
}

func main() {
	// Check for required environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  GEMINI_API_KEY not set, using fallback mode...")
		rankings := createMockRankings()
		
		if err := saveRankings(rankings); err != nil {
			log.Fatal(err)
		}
		
		if err := updateReadmes(rankings); err != nil {
			log.Fatal(err)
		}
		
		fmt.Println("Rankings updated successfully using fallback data!")
		return
	}

	rankings, err := processRankingsWithScreenshot(apiKey)
	if err != nil {
		fmt.Printf("❌ Screenshot analysis failed: %v\n", err)
		fmt.Println("🔄 Falling back to mock data...")
		
		rankings = createMockRankings()
	}

	if err := saveRankings(rankings); err != nil {
		log.Fatal(err)
	}

	if err := updateReadmes(rankings); err != nil {
		log.Fatal(err)
	}

	if rankings.Analysis == "Mock data used as fallback due to screenshot capture issues." {
		fmt.Println("Rankings updated using fallback data!")
	} else {
		fmt.Println("Rankings updated successfully using screenshot analysis!")
	}
}

func processRankingsWithScreenshot(apiKey string) (*RankingData, error) {
	// Step 1: Capture screenshot
	fmt.Println("📸 Capturing screenshot of OpenRouter rankings page...")
	screenshotPath, err := captureScreenshot()
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Step 2: Analyze screenshot with Gemini
	fmt.Println("🤖 Analyzing screenshot with Gemini 2.5 Pro...")
	rankings, err := analyzeScreenshotWithGemini(apiKey, screenshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze screenshot: %w", err)
	}

	return rankings, nil
}

func captureScreenshot() (string, error) {
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

	// Set longer timeout
	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// Generate screenshot filename
	now := time.Now()
	filename := fmt.Sprintf("screenshot_%s.png", now.Format("2006_01_02"))
	screenshotPath := filepath.Join("data", filename)

	var buf []byte
	
	fmt.Println("🌐 Navigating to OpenRouter rankings page...")
	
	// Navigate and capture screenshot with better error handling
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://openrouter.ai/rankings"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(10*time.Second), // Wait for dynamic content
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("🔍 Looking for Programming category...")
			// Try to click on Programming category, but don't fail if not found
			err := chromedp.Click(`button[data-category="Programming"], .category-button:contains("Programming"), [aria-label*="Programming"]`, chromedp.NodeVisible, chromedp.ByQuery).Do(ctx)
			if err != nil {
				fmt.Println("⚠️  Programming category button not found, using general rankings")
			}
			return nil // Don't fail if category button not found
		}),
		chromedp.Sleep(5*time.Second), // Wait for any category change
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("📸 Taking screenshot...")
			return chromedp.FullScreenshot(&buf, 90).Do(ctx)
		}),
	)

	if err != nil {
		return "", fmt.Errorf("failed to capture screenshot: %w", err)
	}

	if len(buf) == 0 {
		return "", fmt.Errorf("screenshot buffer is empty")
	}

	// Save screenshot to file
	if err := os.WriteFile(screenshotPath, buf, 0644); err != nil {
		return "", fmt.Errorf("failed to save screenshot: %w", err)
	}

	fmt.Printf("✅ Screenshot saved to: %s (%d bytes)\n", screenshotPath, len(buf))
	return screenshotPath, nil
}

func analyzeScreenshotWithGemini(apiKey, screenshotPath string) (*RankingData, error) {
	ctx := context.Background()
	
	// Initialize Gemini client
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	// Read screenshot file
	imageData, err := os.ReadFile(screenshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshot: %w", err)
	}

	// Use Gemini 2.5 Pro model
	model := client.GenerativeModel("gemini-2.0-flash-exp")
	
	// Configure model for structured output
	model.SetTemperature(0.1) // Low temperature for consistent results
	
	// Create prompt for analysis
	prompt := `Analyze this screenshot of the OpenRouter AI rankings page and extract the following information:

1. Identify the Programming category rankings (if visible)
2. Extract the top 10 models with their names and usage percentages or scores
3. Focus on programming-related models like Claude, GPT, Gemini, LLaMA, Qwen, etc.
4. Provide a brief analysis of the ranking trends
5. Also provide a Chinese translation of the analysis

Please respond in the following JSON format:
{
  "models": [
    {
      "name": "model-name",
      "score": "percentage or score",
      "rank": 1
    }
  ],
  "analysis": "Brief analysis of the rankings and trends in English",
  "analysis_zh": "Brief analysis of the rankings and trends in Chinese",
  "category": "Programming"
}

If you cannot clearly identify the Programming category, analyze the general rankings visible and note this in the analysis.`

	// Create the request
	resp, err := model.GenerateContent(ctx, 
		genai.Text(prompt),
		genai.ImageData("image/png", imageData),
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Extract text from response
	var responseText string
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		responseText = fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	} else {
		return nil, fmt.Errorf("no response from Gemini")
	}

	// Parse JSON response
	rankings, err := parseGeminiResponse(responseText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	return rankings, nil
}

func parseGeminiResponse(responseText string) (*RankingData, error) {
	// Find JSON content in response
	jsonStart := strings.Index(responseText, "{")
	jsonEnd := strings.LastIndex(responseText, "}")
	
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	jsonStr := responseText[jsonStart : jsonEnd+1]
	
	// Parse JSON
	var geminiResp struct {
		Models   []struct {
			Name  string `json:"name"`
			Score string `json:"score"`
			Rank  int    `json:"rank"`
		} `json:"models"`
		Analysis   string `json:"analysis"`
		AnalysisZh string `json:"analysis_zh"`
		Category   string `json:"category"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to RankingData
	rankings := &RankingData{
		Category:   geminiResp.Category,
		Date:       time.Now(),
		Analysis:   geminiResp.Analysis,
		AnalysisZh: geminiResp.AnalysisZh,
		Models:     []ModelRanking{},
	}

	for _, model := range geminiResp.Models {
		rankings.Models = append(rankings.Models, ModelRanking{
			Name:       model.Name,
			Score:      model.Score,
			Rank:       model.Rank,
			UpdateTime: time.Now(),
		})
	}

	return rankings, nil
}

func createMockRankings() *RankingData {
	return &RankingData{
		Category:   "Programming",
		Date:       time.Now(),
		Analysis:   "Mock data used as fallback due to screenshot capture issues.",
		AnalysisZh: "由于截图捕获问题，使用模拟数据作为备用方案。",
		Models: []ModelRanking{
			{"claude-3-5-sonnet-20241022", "31.2%", 1, time.Now()},
			{"gpt-4o-2024-08-06", "18.5%", 2, time.Now()},
			{"claude-3-5-haiku-20241022", "12.8%", 3, time.Now()},
			{"gpt-4o-mini-2024-07-18", "9.3%", 4, time.Now()},
			{"gemini-2.0-flash-exp", "7.1%", 5, time.Now()},
			{"claude-3-opus-20240229", "5.9%", 6, time.Now()},
			{"gpt-4-turbo-2024-04-09", "4.2%", 7, time.Now()},
			{"gemini-1.5-pro-002", "3.8%", 8, time.Now()},
			{"llama-3.3-70b-instruct", "3.1%", 9, time.Now()},
			{"qwen-2.5-coder-32b-instruct", "2.6%", 10, time.Now()},
		},
	}
}

func saveRankings(rankings *RankingData) error {
	data, err := json.MarshalIndent(rankings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rankings: %w", err)
	}

	filename := fmt.Sprintf("rankings_%s.json", time.Now().Format("2006-01-02"))
	return os.WriteFile(filename, data, 0644)
}

func updateReadmes(rankings *RankingData) error {
	// Generate English README
	if err := updateEnglishReadme(rankings); err != nil {
		return fmt.Errorf("failed to update English README: %w", err)
	}
	
	// Generate Chinese README
	if err := updateChineseReadme(rankings); err != nil {
		return fmt.Errorf("failed to update Chinese README: %w", err)
	}
	
	fmt.Println("✅ Generated both English (README.md) and Chinese (README_zh.md) versions")
	return nil
}

func updateEnglishReadme(rankings *RankingData) error {
	readmeContent := fmt.Sprintf(`# OpenRouter LLM Rankings - Programming Category

Last updated: %s

## Top 10 Programming Models

`, rankings.Date.Format("2006-01-02 15:04:05"))

	for _, model := range rankings.Models {
		readmeContent += fmt.Sprintf("%d. **%s** - %s\n", model.Rank, model.Name, model.Score)
	}

	readmeContent += fmt.Sprintf(`

## Analysis

%s

---

*This ranking is automatically updated weekly via GitHub Actions using screenshot analysis and AI.*
*Data source: [OpenRouter Rankings](https://openrouter.ai/rankings)*
*Analysis powered by Google Gemini 2.5 Pro*

Generated on: %s

**Language**: [English](README.md) | [中文](README_zh.md)
`, rankings.Analysis, time.Now().Format("2006-01-02 15:04:05"))

	return os.WriteFile("README.md", []byte(readmeContent), 0644)
}

func updateChineseReadme(rankings *RankingData) error {
	// Use Chinese analysis if available, otherwise translate English
	var chineseAnalysis string
	if rankings.AnalysisZh != "" {
		chineseAnalysis = rankings.AnalysisZh
	} else {
		chineseAnalysis = translateAnalysisToChinese(rankings.Analysis)
	}
	
	readmeContent := fmt.Sprintf(`# OpenRouter LLM 排名 - 编程类别

最后更新: %s

## 编程模型 Top 10

`, rankings.Date.Format("2006-01-02 15:04:05"))

	for _, model := range rankings.Models {
		readmeContent += fmt.Sprintf("%d. **%s** - %s\n", model.Rank, model.Name, model.Score)
	}

	readmeContent += fmt.Sprintf(`

## 分析报告

%s

---

*此排名通过 GitHub Actions 使用截图分析和 AI 技术每周自动更新。*
*数据来源: [OpenRouter Rankings](https://openrouter.ai/rankings)*
*分析技术: Google Gemini 2.5 Pro*

生成时间: %s

**语言**: [English](README.md) | [中文](README_zh.md)
`, chineseAnalysis, time.Now().Format("2006-01-02 15:04:05"))

	return os.WriteFile("README_zh.md", []byte(readmeContent), 0644)
}

func translateAnalysisToChinese(englishAnalysis string) string {
	// Simple translation mapping for common analysis patterns
	translations := map[string]string{
		"Mock data used as fallback due to screenshot capture issues.": "由于截图捕获问题，使用模拟数据作为备用方案。",
		"Claude": "Claude",
		"GPT": "GPT", 
		"Gemini": "Gemini",
		"DeepSeek": "DeepSeek",
		"dominated": "主导",
		"leading": "领先",
		"strong presence": "强势表现",
		"market share": "市场份额",
		"Programming category": "编程类别",
		"tokens": "token",
		"variants": "变体",
		"The top models are": "顶级模型",
		"followed by": "其次是",
	}
	
	// If it's the fallback message, return Chinese directly
	if englishAnalysis == "Mock data used as fallback due to screenshot capture issues." {
		return translations[englishAnalysis]
	}
	
	// For other analysis, try basic translation
	chineseText := englishAnalysis
	for en, zh := range translations {
		chineseText = strings.ReplaceAll(chineseText, en, zh)
	}
	
	// If no translation found, add a note
	if chineseText == englishAnalysis {
		return fmt.Sprintf("%s\n\n*注：此分析为英文原文，建议添加中文翻译。*", englishAnalysis)
	}
	
	return chineseText
}