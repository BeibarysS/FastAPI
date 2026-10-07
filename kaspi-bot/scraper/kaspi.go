package scraper

import (
    "kaspi-bot/models"
    "context"
    "fmt"
    "log"
    "net/url"
    "strconv"
    "strings"
    "time"

    "github.com/chromedp/cdproto/cdp"
    "github.com/chromedp/chromedp"
)

func ScrapeKaspiProducts2(login, password string) ([]models.Product, error) {
	opts, cancel := chromedp.NewExecAllocator(context.Background(),
		chromedp.Flag("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
        chromedp.Flag("headless", true),
        chromedp.Flag("no-sandbox", true),
        chromedp.Flag("disable-gpu", true),
        chromedp.Flag("disable-dev-shm-usage", true),
        chromedp.Flag("disable-setuid-sandbox", true),
        chromedp.Flag("disable-extensions", true),
        chromedp.Flag("disable-background-networking", true),
        chromedp.Flag("disable-sync", true),
        chromedp.Flag("disable-translate", true),
        )
	defer cancel()
	allocCtx, cancel := chromedp.NewContext(opts)
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(allocCtx, 120*time.Second)
	defer cancelTimeout()


    log.Println("🔄 Starting Kaspi product scraping...")

    var products []models.Product
    var success bool
    // Login first
    err := chromedp.Run(ctx,
        chromedp.Navigate("https://idmc.shop.kaspi.kz"),
		chromedp.WaitVisible(`#user_email_field`, chromedp.ByID),
		chromedp.SendKeys(`#user_email_field`, login, chromedp.ByID),
		chromedp.Click(`.button.is-primary`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.WaitVisible(`#password_field`, chromedp.ByID),
		chromedp.SendKeys(`#password_field`, password, chromedp.ByID),
		chromedp.Click(`.button.is-primary`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`document.querySelector('body') !== null`, &success),
		chromedp.Navigate("https://kaspi.kz/mc/#/products/active/1"),
    )
    
    if err != nil || !success {
        return nil, fmt.Errorf("login failed: %w", err)
    }

    // Navigate to products page
    err = chromedp.Run(ctx,
        chromedp.Navigate("https://kaspi.kz/mc/#/products/active/1"),
        chromedp.Sleep(3*time.Second),
    )
    
    if err != nil {
        return nil, fmt.Errorf("failed to navigate to products: %w", err)
    }

    // Scrape products from multiple pages
    page := 1
    for page <= 5 { // Limit to 5 pages to avoid infinite loops
        log.Printf("📄 Scraping page %d", page)
        
        var names, prices, images []string
        
        err := chromedp.Run(ctx,
        	chromedp.Sleep(5*time.Second),
			chromedp.Evaluate(`Array.from(document.querySelectorAll('a.link-placeholder-class')).map(el => el.innerText)`, &names),
			chromedp.Evaluate(`Array.from(document.querySelectorAll('td[data-label="Цена, тенге"]')).map(el => el.innerText)`, &prices),
			chromedp.Evaluate(`Array.from(document.querySelectorAll('img.thumbnail')).map(el => el.src)`, &images),
        )
        
        if err != nil || len(names) == 0 {
            log.Printf("No more products found on page %d", page)
            break
        }

        // Process scraped data
        for i := 0; i < len(names) && i < len(prices); i++ {
            price := parsePrice(prices[i])
            imageURL := ""
            if i < len(images) {
                imageURL = images[i]
            }
            
            product := models.Product{
                Name:     strings.TrimSpace(names[i]),
                Price:    price,
                MinPrice: price, // 
                MaxPrice: price, // 
                Image:    imageURL,
            }
            
            products = append(products, product)
        }

        // Try to go to next page
        var hasNextPage bool
        err = chromedp.Run(ctx,
            chromedp.Evaluate(`
                document.querySelector('.pagination .next, .page-next, [aria-label="Next"]') !== null
            `, &hasNextPage),
        )
        
        if err != nil || !hasNextPage {
            break
        }
        
        err = chromedp.Run(ctx,
            chromedp.Click(`.pagination .next, .page-next, [aria-label="Next"]`, chromedp.ByQuery),
            chromedp.Sleep(3*time.Second),
        )
        
        if err != nil {
            break
        }
        
        page++
    }

    log.Printf("✅ Scraped %d products from Kaspi", len(products))
    return products, nil
}

func ScrapeKaspiProducts(login, password string) ([]models.Product, error) {
    // --- create allocator with user-agent and flags
    allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", true),
        chromedp.Flag("no-sandbox", true),
        chromedp.Flag("disable-gpu", true),
        chromedp.Flag("disable-dev-shm-usage", true),
        chromedp.Flag("disable-setuid-sandbox", true),
        chromedp.Flag("disable-extensions", true),
        chromedp.Flag("disable-background-networking", true),
        chromedp.Flag("disable-sync", true),
        chromedp.Flag("disable-translate", true),
        chromedp.Flag("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
    )
    allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
    defer allocCancel()

    // create browser context
    ctx, ctxCancel := chromedp.NewContext(allocCtx)
    defer ctxCancel()

    // add overall timeout (increase if you need more time)
    ctx, timeoutCancel := context.WithTimeout(ctx, 1*time.Minute)
    defer timeoutCancel()

    log.Println("🔄 Starting Kaspi product scraping...")

    // --- Try login with retries
    var loginErr error
    for i := 0; i < 3; i++ {
        loginErr = chromedp.Run(ctx,
            chromedp.Navigate("https://idmc.shop.kaspi.kz"),
            chromedp.WaitVisible(`body`, chromedp.ByQuery),
            chromedp.WaitVisible(`#user_email_field`, chromedp.ByID),
            chromedp.SendKeys(`#user_email_field`, login, chromedp.ByID),
            chromedp.WaitVisible(`.button.is-primary`, chromedp.ByQuery),
            chromedp.Click(`.button.is-primary`, chromedp.ByQuery),
            chromedp.Sleep(2*time.Second),
            chromedp.WaitVisible(`#password_field`, chromedp.ByID),
            chromedp.SendKeys(`#password_field`, password, chromedp.ByID),
            chromedp.WaitVisible(`.button.is-primary`, chromedp.ByQuery),
            chromedp.Click(`.button.is-primary`, chromedp.ByQuery),
            chromedp.Sleep(3*time.Second),
        )
        if loginErr == nil {
            break
        }
        log.Printf("Login attempt %d failed: %v", i+1, loginErr)
        time.Sleep(2 * time.Second)
    }
    if loginErr != nil {
        return nil, fmt.Errorf("login failed: %w", loginErr)
    }

    // --- navigate to products page and wait some element that indicates page is ready
    if err := chromedp.Run(ctx,
        chromedp.Sleep(2*time.Second),
        chromedp.Navigate("https://kaspi.kz/mc/#/products/active/1"),
        // Wait for either product link selector or a table container — choose the best existing selector for your UI
        chromedp.Sleep(2*time.Second),
        chromedp.Navigate("https://kaspi.kz/mc/#/products/active/1"),
        chromedp.WaitVisible(`a.link-placeholder-class, .products-table, .product-list`, chromedp.ByQuery),
        chromedp.Sleep(2*time.Second),
    ); err != nil {
        // don't fail hard here — log and continue; sometimes SPA navigation is slow
        log.Printf("warn: navigate to products page: %v", err)
    }

    var products []models.Product

    // --- scrape up to N pages
    for page := 1; page <= 10; page++ {
        log.Printf("📄 Scraping page %d", page)

        var names, prices, images []string
        evalErr := chromedp.Run(ctx,
            chromedp.Sleep(5*time.Second),
            
             chromedp.Evaluate(`Array.from(document.querySelectorAll('a.link-placeholder-class')).map(el => (el.innerText||el.textContent).trim())`, &names),
            chromedp.Evaluate(`Array.from(document.querySelectorAll('td[data-label="Цена, тенге"]')).map(el => (el.innerText||el.textContent).trim())`, &prices),
            chromedp.Evaluate(`Array.from(document.querySelectorAll('img.thumbnail')).map(el => el.src)`, &images),
        )
        if evalErr != nil {
            log.Printf("❌ Evaluate error on page %d: %v", page, evalErr)
            // if evaluate failed completely, break — could be selector change
            break
        }

        log.Printf("found names=%d prices=%d images=%d", len(names), len(prices), len(images))
        if len(names) == 0 {
            log.Printf("No products found on page %d — stopping", page)
            break
        }

        // build products slice
        for i := 0; i < len(names) && i < len(prices); i++ {
            p := models.Product{
                Name:     strings.TrimSpace(names[i+1]),
                Price:    parsePrice(prices[i]),
                MinPrice: parsePrice(prices[i]),
                MaxPrice: parsePrice(prices[i]),
                Image: images[i],
            }
            products = append(products, p)
        }

        // check next page existence
        var hasNext bool
        if err := chromedp.Run(ctx,
            chromedp.Evaluate(`document.querySelector('.pagination .next, .page-next, [aria-label="Next"]') !== null`, &hasNext),
        ); err != nil {
            log.Printf("Next page check error: %v", err)
            break
        }
        if !hasNext {
            log.Printf("No next page, finished")
            break
        }

        // click next
        if err := chromedp.Run(ctx,
            chromedp.Click(`.pagination .next, .page-next, [aria-label="Next"]`, chromedp.ByQuery),
            chromedp.Sleep(2*time.Second),
        ); err != nil {
            log.Printf("Failed to click next on page %d: %v", page, err)
            break
        }
    }

    log.Printf("✅ Scraped %d products from Kaspi", len(products))
    return products, nil
}



func GetLowestCompetitorPrice(productName string) (float64, error){
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
	searchURL := "https://kaspi.kz/shop/search/?text=" + url.QueryEscape(productName)
    

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
	price := parsePrice(priceText)
    return price, nil
}

func parsePrice(s string) float64 {
    // Clean price string
    s = strings.ReplaceAll(s, "\u00a0", "") // non-breaking space
    s = strings.ReplaceAll(s, " ", "")
    s = strings.ReplaceAll(s, "₸", "")
    s = strings.ReplaceAll(s, "тг", "")
    s = strings.ReplaceAll(s, ",", "")
    s = strings.TrimSpace(s)
    
    price, err := strconv.ParseFloat(s, 64)
    if err != nil {
        return 0
    }
    
    return price
}
