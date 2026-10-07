package kaspi

import (
    "context"
    "fmt"
    "log"
    "strconv"
    "time"
    "strings"

    "github.com/chromedp/chromedp"
)

func UpdateProductPriceOnKaspi(login, password, name string, price float64) error {
    
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
    priceStr := strconv.FormatFloat(price, 'f', 0, 64)

    // add overall timeout (increase if you need more time)
    ctx, timeoutCancel := context.WithTimeout(ctx, 2*time.Minute)
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
        return fmt.Errorf("login failed: %w", loginErr)
    }

    // 2. Баға өзгерту логикасы (changePrice үлгісімен)
    encodedName := strings.ReplaceAll(name, " ", "%20")
    encodedName = strings.ReplaceAll(encodedName, "\"", "%22")
    encodedName = strings.ReplaceAll(encodedName, "#", "%23")
    encodedName = strings.ReplaceAll(encodedName, "&", "%26")
    encodedName = strings.ReplaceAll(encodedName, "=", "%3D")
    encodedName = strings.ReplaceAll(encodedName, "/", "%2F")

    url := fmt.Sprintf("https://kaspi.kz/mc/#/products/active/1?text=%s", encodedName)

    log.Printf("Changing price for: %s to %s", name, priceStr)

    err := chromedp.Run(ctx,
        chromedp.Navigate("https://kaspi.kz/mc/#/products/active/1"),
        chromedp.Sleep(8*time.Second),
        chromedp.Navigate(url),
        chromedp.Sleep(5*time.Second),

        chromedp.Click(`img.dropdown__icon`, chromedp.ByQuery),
        chromedp.Sleep(3*time.Second),
        chromedp.Click(`a[data-v-12b625cc]:first-child`, chromedp.ByQuery),
        chromedp.Sleep(2*time.Second),

        chromedp.Evaluate(`
            (function() {
                const input = document.querySelector('input.input.mb-1');
                if (input) {
                    input.value = '`+priceStr+`';
                    input.dispatchEvent(new Event('input', { bubbles: true }));
                    input.dispatchEvent(new Event('change', { bubbles: true }));
                    return true;
                }
                return false;
            })()
        `, nil),

        chromedp.Sleep(1*time.Second),

        chromedp.Evaluate(`
            (function() {
                const elements = document.querySelectorAll('span');
                for (let el of elements) {
                    if (el.textContent.trim() === "Сохранить изменения") {
                        el.click();
                        return true;
                    }
                }
                return false;
            })()
        `, nil),

        chromedp.Sleep(3*time.Second),
    )
    if err != nil {
        return fmt.Errorf("failed to change price: %v", err)
    }

    log.Printf("✅ Price changed successfully for: %s", name)
    return nil
}