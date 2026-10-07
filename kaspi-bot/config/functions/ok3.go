package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// KaspiPriceChecker handles price checking operations
type KaspiPriceChecker struct {
	baseURL   string
	searchURL string
}

// NewKaspiPriceChecker creates a new price checker instance
func NewKaspiPriceChecker() *KaspiPriceChecker {
	return &KaspiPriceChecker{
		baseURL:   "https://kaspi.kz",
		searchURL: "https://kaspi.kz/shop/search/",
	}
}

// GetProductPrice searches for a product and returns its price
func (k *KaspiPriceChecker) GetProductPrice(productName string) (float64, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create chromedp context with options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Encode product name for URL
	encodedName := url.QueryEscape(productName)
	searchURL := fmt.Sprintf("%s?text=%s", k.searchURL, encodedName)

	fmt.Printf("Searching for: %s\n", productName)
	fmt.Printf("URL: %s\n", searchURL)

	var priceText string
	var nodes []*cdp.Node

	// Run chromedp tasks
	err := chromedp.Run(ctx,
		// Navigate to search page
		chromedp.Navigate(searchURL),
		
		// Wait for page to load
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		
		// Wait a bit for dynamic content to load
		chromedp.Sleep(3*time.Second),
		
		// Find all product cards
		chromedp.Nodes(`div.item-card.ddl_product.ddl_product_link`, &nodes, chromedp.ByQueryAll),
	)

	if err != nil {
		return 0, fmt.Errorf("failed to load page: %v", err)
	}

	if len(nodes) == 0 {
		return 0, fmt.Errorf("no products found")
	}

	// Try to get price from the first product
	err = chromedp.Run(ctx,
		// Look for price in the first product card
		chromedp.Text(`div.item-card.ddl_product.ddl_product_link:first-child span.item-card__prices-price`, &priceText, chromedp.ByQuery),
	)

	if err != nil {
		// If first selector doesn't work, try alternative selectors
		err = chromedp.Run(ctx,
			chromedp.Text(`.item-card__prices-price`, &priceText, chromedp.ByQuery),
		)
		
		if err != nil {
			return 0, fmt.Errorf("price element not found: %v", err)
		}
	}

	fmt.Printf("Found price text: %s\n", priceText)

	// Parse price from text
	price, err := k.parsePrice(priceText)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price: %v", err)
	}

	fmt.Printf("Extracted price: %.2f KZT\n", price)
	return price, nil
}

// GetProductPriceByURL gets price from a specific product URL
func (k *KaspiPriceChecker) GetProductPriceByURL(productURL string) (float64, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create chromedp context with options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	fmt.Printf("Fetching product from URL: %s\n", productURL)

	var priceText string

	// Run chromedp tasks
	err := chromedp.Run(ctx,
		// Navigate to product page
		chromedp.Navigate(productURL),
		
		// Wait for page to load
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		
		// Wait a bit for dynamic content to load
		chromedp.Sleep(3*time.Second),
		
		// Get price text
		chromedp.Text(`span.item-card__prices-price`, &priceText, chromedp.ByQuery),
	)

	if err != nil {
		return 0, fmt.Errorf("failed to get price: %v", err)
	}

	fmt.Printf("Found price text: %s\n", priceText)

	// Parse price from text
	price, err := k.parsePrice(priceText)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price: %v", err)
	}

	fmt.Printf("Extracted price: %.2f KZT\n", price)
	return price, nil
}

// parsePrice extracts numeric price from text
func (k *KaspiPriceChecker) parsePrice(priceText string) (float64, error) {
	// Remove currency symbols and clean the text
	cleaned := strings.ReplaceAll(priceText, "₸", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "\u00a0", "") // Non-breaking space
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\t", "")
	cleaned = strings.TrimSpace(cleaned)

	// Use regex to extract numbers
	re := regexp.MustCompile(`[\d,]+`)
	matches := re.FindAllString(cleaned, -1)
	
	if len(matches) == 0 {
		return 0, fmt.Errorf("no numeric value found in: %s", priceText)
	}

	// Take the first (and usually only) numeric match
	numStr := strings.ReplaceAll(matches[0], ",", "")
	
	price, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert to float: %v", err)
	}

	return price, nil
}

// GetProductPriceWithRetry attempts to get price with retry logic
func (k *KaspiPriceChecker) GetProductPriceWithRetry(productName string, maxRetries int) (float64, error) {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			fmt.Printf("Retry attempt %d/%d\n", i+1, maxRetries)
			time.Sleep(time.Duration(i*2) * time.Second) // Exponential backoff
		}
		
		price, err := k.GetProductPrice(productName)
		if err == nil {
			return price, nil
		}
		
		lastErr = err
		fmt.Printf("Attempt %d failed: %v\n", i+1, err)
	}
	
	return 0, fmt.Errorf("all retry attempts failed, last error: %v", lastErr)
}

func main() {
	checker := NewKaspiPriceChecker()

	// Test with the chess set example
	productName := "Шахматы Королевские малые Shantou 1021 21x22"
	
	price, err := checker.GetProductPriceWithRetry(productName, 3)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Product: %s\n", productName)
	fmt.Printf("💰 Price: %.2f KZT\n", price)

	// Example with direct URL (uncomment to test)
	/*
	url := "https://kaspi.kz/shop/p/korolevskie-malye-shantou-1021-21x22-sm-119133533/"
	price, err := checker.GetProductPriceByURL(url)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("💰 Price from URL: %.2f KZT\n", price)
	*/
}