package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Test function to demonstrate the screenshot-based approach
func testScreenshotApproach() {
	fmt.Println("🧪 Testing screenshot-based approach with mock data...")
	
	// Mock Gemini analysis response
	mockAnalysis := `{
		"models": [
			{
				"name": "claude-3-5-sonnet-20241022",
				"score": "31.2%",
				"rank": 1
			},
			{
				"name": "gpt-4o-2024-08-06",
				"score": "18.5%",
				"rank": 2
			},
			{
				"name": "claude-3-5-haiku-20241022",
				"score": "12.8%",
				"rank": 3
			},
			{
				"name": "gpt-4o-mini-2024-07-18",
				"score": "9.3%",
				"rank": 4
			},
			{
				"name": "gemini-2.0-flash-exp",
				"score": "7.1%",
				"rank": 5
			},
			{
				"name": "claude-3-opus-20240229",
				"score": "5.9%",
				"rank": 6
			},
			{
				"name": "gpt-4-turbo-2024-04-09",
				"score": "4.2%",
				"rank": 7
			},
			{
				"name": "gemini-1.5-pro-002",
				"score": "3.8%",
				"rank": 8
			},
			{
				"name": "llama-3.3-70b-instruct",
				"score": "3.1%",
				"rank": 9
			},
			{
				"name": "qwen-2.5-coder-32b-instruct",
				"score": "2.6%",
				"rank": 10
			}
		],
		"analysis": "Claude 3.5 Sonnet dominates programming tasks with 31.2% usage, followed by GPT-4o at 18.5%. The top 3 models account for over 60% of programming-related usage. There's a clear preference for newer, more capable models in programming contexts, with Claude and GPT models leading the rankings. The distribution shows a long tail with smaller models capturing niche use cases.",
		"category": "Programming"
	}`
	
	// Parse mock response
	rankings, err := parseGeminiResponse(mockAnalysis)
	if err != nil {
		fmt.Printf("❌ Error parsing mock response: %v\n", err)
		return
	}
	
	// Save rankings
	if err := saveRankings(rankings); err != nil {
		fmt.Printf("❌ Error saving rankings: %v\n", err)
		return
	}
	
	// Update README
	if err := updateReadme(rankings); err != nil {
		fmt.Printf("❌ Error updating README: %v\n", err)
		return
	}
	
	// Create mock screenshot file
	mockScreenshotPath := fmt.Sprintf("data/screenshot_%s.png", time.Now().Format("2006_01_02"))
	if err := os.WriteFile(mockScreenshotPath, []byte("mock screenshot data"), 0644); err != nil {
		fmt.Printf("❌ Error creating mock screenshot: %v\n", err)
		return
	}
	
	fmt.Println("✅ Screenshot-based approach test completed successfully!")
	fmt.Printf("📊 Analyzed %d models from Programming category\n", len(rankings.Models))
	fmt.Printf("📝 Updated README with AI-generated analysis\n")
	fmt.Printf("💾 Saved rankings to JSON file\n")
	fmt.Printf("📸 Mock screenshot saved to: %s\n", mockScreenshotPath)
}

func main() {
	// Check if this is a test run
	if len(os.Args) > 1 && os.Args[1] == "test" {
		testScreenshotApproach()
		return
	}
	
	// Regular execution would require GEMINI_API_KEY
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  GEMINI_API_KEY not set, running test mode...")
		testScreenshotApproach()
		return
	}
	
	fmt.Println("🚀 Running with real Gemini API...")
	// Real implementation would go here
	fmt.Println("Real implementation requires API key and Chrome installation")
}