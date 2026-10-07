package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"strings"
	"github.com/chromedp/chromedp"
)

func main() {
	allocatorCtx, cancel := chromedp.NewExecAllocator(context.Background(),
		chromedp.Flag("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)
	defer cancel()
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 60*time.Second)
	defer cancelTimeout()

	// Test Login
	for i := 0; i < 3; i++ {
		err := login(ctx, "beibarys4ever@gmail.com", "AqTmwGJZXt")
		if err == nil {
			break
		}
		log.Printf("Login attempt %d failed: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	// Test Ger all Product
	products, prices, images, err := getProductData(ctx)
	if err != nil {
		log.Fatalf("Failed to get product data: %v", err)
	}
	log.Printf("Products: %v\nPrices: %v\nImages: %v", products, prices, images)


// 	// Test Change Price
// 	err := changePrice(ctx, "25000", "Королевские малые Shantou 1021 21x22 см")
// if err != nil {
//     log.Printf("Error: %v", err)
// }





	// // test check min proce for product
	// productName := "Шахматы Королевские малые Shantou 1021 21x22"
	// price, err := getProductPrice(productName)
	// if err != nil {
	// 	log.Fatalf("Failed to get price: %v", err)
	// }
	// fmt.Printf("Price: %s\n", price)
}

// login функциясы
func login(ctx context.Context, email, password string) error {
	var success bool
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://idmc.shop.kaspi.kz"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery), // сайттың толық жүктелуін күту
		chromedp.WaitVisible(`#user_email_field`, chromedp.ByID),
		chromedp.SendKeys(`#user_email_field`, email, chromedp.ByID),
		chromedp.WaitVisible(`.button.is-primary`, chromedp.ByQuery),
		chromedp.Click(`.button.is-primary`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
		chromedp.WaitVisible(`#password_field`, chromedp.ByID),
		chromedp.SendKeys(`#password_field`, password, chromedp.ByID),
		chromedp.WaitVisible(`.button.is-primary`, chromedp.ByQuery),
		chromedp.Click(`.button.is-primary`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`document.querySelector('body') !== null`, &success),
		chromedp.WaitVisible(`body`, chromedp.ByQuery), // сайттың толық жүктелуін күту
		chromedp.Navigate("https://kaspi.kz/mc/#/products/active/1"),
		
	)
	if err != nil || !success {
		return fmt.Errorf("login failed: %v", err)
	}
	return nil
}

func getProductData(ctx context.Context) ([]string, []string, []string, error) {
	var products, prices, images []string
	err := chromedp.Run(ctx,
		chromedp.Sleep(5*time.Second),
		chromedp.Navigate("https://kaspi.kz/mc/#/products/active/1"),
		
		chromedp.Sleep(10*time.Second), // сайттың толық жүктелуін күту
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a.link-placeholder-class')).map(el => el.innerText)`, &products),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('td[data-label="Цена, тенге"]')).map(el => el.innerText)`, &prices),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('img.thumbnail')).map(el => el.src)`, &images),
		
		chromedp.Sleep(5*time.Second),
	)
	return products, prices, images, err
}


func changePrice(ctx context.Context, price string, name string) error {

	encodedName := strings.ReplaceAll(name, " ", "%20")
	encodedName = strings.ReplaceAll(encodedName, "\"", "%22")
	encodedName = strings.ReplaceAll(encodedName, "#", "%23")
	encodedName = strings.ReplaceAll(encodedName, "&", "%26")
	encodedName = strings.ReplaceAll(encodedName, "=", "%3D")
	encodedName = strings.ReplaceAll(encodedName, "/", "%2F")
	
	url := fmt.Sprintf("https://kaspi.kz/mc/#/products/active/1?text=%s", encodedName)
	
	log.Printf("Changing price for: %s to %s", name, price)
	
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://kaspi.kz/mc/#/products/active/1"),
		chromedp.Sleep(10*time.Second),
		chromedp.Navigate(url),
		chromedp.Sleep(5*time.Second),
		
		chromedp.Click(`img.dropdown__icon`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
		chromedp.Click(`a[data-v-12b625cc]:first-child`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		
		chromedp.Evaluate(`
    	(function() {
        	const input = document.querySelector('input.input.mb-1');
        	if (input) {
            	input.value = '`+price+`';
				input.dispatchEvent(new Event('input', { bubbles: true }));
				input.dispatchEvent(new Event('change', { bubbles: true }));
				return true;
        		}
        	return false;
   	 })()
`, nil),

		chromedp.SendKeys(`input.input.text-block__input`, "9", chromedp.ByQuery),
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
	
	log.Printf("Price changed successfully for: %s", name)
	return nil
}
