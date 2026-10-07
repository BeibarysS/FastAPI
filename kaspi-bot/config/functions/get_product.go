package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)
func getProductDataWithPaginationRobust(ctx context.Context) ([]string, []string, []string, error) {
	var allProducts, allPrices, allImages []string
	pageNum := 1

	// Navigate to the first page
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://kaspi.kz/mc/#/products/active/1"),
		chromedp.Sleep(10*time.Second),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to navigate to first page: %v", err)
	}

	for {
		fmt.Printf("Scraping page %d...\n", pageNum)

		// Get data from current page
		var pageProducts, pagePrices, pageImages []string
		err := chromedp.Run(ctx,
			chromedp.Sleep(5*time.Second),
			chromedp.Evaluate(`Array.from(document.querySelectorAll('a.link-placeholder-class')).map(el => el.innerText)`, &pageProducts),
			chromedp.Evaluate(`Array.from(document.querySelectorAll('td[data-label="Цена, тенге"]')).map(el => el.innerText)`, &pagePrices),
			chromedp.Evaluate(`Array.from(document.querySelectorAll('img.thumbnail')).map(el => el.src)`, &pageImages),
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to scrape page %d: %v", pageNum, err)
		}

		// Add current page data to total results
		allProducts = append(allProducts, pageProducts...)
		allPrices = append(allPrices, pagePrices...)
		allImages = append(allImages, pageImages...)

		fmt.Printf("Page %d: Found %d products, %d prices, %d images\n", 
			pageNum, len(pageProducts), len(pagePrices), len(pageImages))

		// Check pagination status with multiple methods
		var canGoNext bool
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`
				// Method 1: Check if next button exists and is not disabled
				const nextButton = document.querySelector('a[role="button"][aria-label="Next page"]');
				if (!nextButton) return false;
				if (nextButton.getAttribute('disabled') === 'disabled') return false;
				if (nextButton.classList.contains('disabled')) return false;
				
				// Method 2: Check if next button has href
				if (nextButton.getAttribute('href') === '#') return false;
				
				// Method 3: Check pagination list for next item
				const paginationList = document.querySelector('ul.pagination-list');
				if (paginationList) {
					const currentPage = paginationList.querySelector('li.active, li.current');
					if (currentPage && currentPage.nextElementSibling) {
						return true;
					}
				}
				
				return true;
			`, &canGoNext),
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to check pagination on page %d: %v", pageNum, err)
		}

		// If we can't go to next page, break
		if !canGoNext {
			fmt.Printf("Cannot go to next page. Reached last page (%d)\n", pageNum)
			break
		}

		// Click next page button
		fmt.Printf("Navigating to next page...\n")
		err = chromedp.Run(ctx,
			chromedp.Click(`a[role="button"][aria-label="Next page"]:not([disabled])`, chromedp.ByQuery),
			chromedp.Sleep(5*time.Second),
		)
		if err != nil {
			// If click fails, try alternative selectors
			fmt.Printf("Primary next button click failed, trying alternatives...\n")
			err = chromedp.Run(ctx,
				chromedp.Click(`a.pagination-next:not(.disabled)`, chromedp.ByQuery),
				chromedp.Sleep(5*time.Second),
			)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to click next page on page %d: %v", pageNum, err)
			}
		}

		pageNum++
		
		// Respect rate limiting
		time.Sleep(2 * time.Second)
		
		// Safety check to prevent infinite loops
		if pageNum > 1000 {
			fmt.Printf("Safety limit reached at page %d\n", pageNum)
			break
		}
	}

	fmt.Printf("Total scraped: %d products, %d prices, %d images across %d pages\n", 
		len(allProducts), len(allPrices), len(allImages), pageNum)

	return allProducts, allPrices, allImages, nil
}

// Example usage
func main() {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Create chromedp context
	ctx, cancel = chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Run the pagination scraper
	products, prices, images, err := getProductDataWithPaginationRobust(ctx)
	if err != nil {
		log.Fatalf("Error scraping data: %v", err)
	}

	fmt.Printf("\n=== FINAL RESULTS ===\n")
	fmt.Printf("Total Products: %d\n", len(products))
	fmt.Printf("Total Prices: %d\n", len(prices))
	fmt.Printf("Total Images: %d\n", len(images))

	// Print first few results as example
	if len(products) > 0 {
		fmt.Printf("\nFirst 5 products:\n")
		for i := 0; i < 5 && i < len(products); i++ {
			fmt.Printf("%d. %s", i+1, products[i])
			if i < len(prices) {
				fmt.Printf(" - Price: %s", prices[i])
			}
			fmt.Printf("\n")
		}
	}
}