package misc

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

const (
	cacheExpirationTime = 12 * time.Hour
	cachePurgeInterval  = 5 * time.Minute
	cacheDir            = "./screenshot_cache"
	targetURL           = "https://agi.creatorstation.com/influencers/%s?tab=3"
	targetURLTab4       = "https://agi.creatorstation.com/influencers/%s?tab=4"
)

func init() {
	if err := initCache(); err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	if err := installPlaywrightBrowsers(); err != nil {
		log.Fatalf("Failed to install Playwright browsers: %v", err)
	}

}

func getCachedScreenshot(cacheKey string) ([]byte, bool) {
	screenshotCache.mu.RLock()
	cacheItem, exists := screenshotCache.cache[cacheKey]
	screenshotCache.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Since(cacheItem.timestamp) < 12*time.Hour {
		imgBytes, err := os.ReadFile(cacheItem.filePath)
		if err == nil {
			return imgBytes, true
		}
		log.Printf("Error reading cached screenshot: %v", err)
	}
	return nil, false
}

func takeScreenshot(username, elementID string) ([]byte, error) {
	pw, browser, err := createPlaywrightBrowser()
	if err != nil {
		return nil, err
	}
	defer pw.Stop()
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		DeviceScaleFactor: playwright.Float(2.0),
	})
	if err != nil {
		return nil, fmt.Errorf("could not create context: %w", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %w", err)
	}

	err = page.SetExtraHTTPHeaders(map[string]string{
		"Authorization": os.Getenv("AGI_TOKEN"),
	})
	if err != nil {
		return nil, fmt.Errorf("could not set headers: %w", err)
	}

	url := fmt.Sprintf(targetURL, username)
	if _, err = page.Goto(url); err != nil {
		return nil, fmt.Errorf("could not navigate to page: %w", err)
	}

	if err := waitForCategoryResponse(page); err != nil {
		return nil, err
	}

	selector := fmt.Sprintf("#%s", elementID)
	elementHandle, err := page.WaitForSelector(selector)
	if err != nil {
		return nil, fmt.Errorf("could not find element: %w", err)
	}

	_, err = elementHandle.EvalOnSelectorAll("h5", "h5Elements => h5Elements.forEach(h5 => h5.remove())")
	if err != nil {
		return nil, fmt.Errorf("could not remove h5 elements: %w", err)
	}

	screenshot, err := elementHandle.Screenshot(playwright.ElementHandleScreenshotOptions{
		Scale: playwright.ScreenshotScaleDevice,
	})
	if err != nil {
		return nil, fmt.Errorf("could not take screenshot: %w", err)
	}

	return screenshot, nil
}

func takeScreenshotTab4(username, elementID, selectedDate, labelFilters, withLinkStory string) ([]byte, error) {
	pw, browser, err := createPlaywrightBrowser()
	if err != nil {
		return nil, err
	}
	defer pw.Stop()
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		DeviceScaleFactor: playwright.Float(2.0),
	})
	if err != nil {
		return nil, fmt.Errorf("could not create context: %w", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %w", err)
	}

	err = page.SetExtraHTTPHeaders(map[string]string{
		"Authorization": os.Getenv("AGI_TOKEN"),
	})
	if err != nil {
		return nil, fmt.Errorf("could not set headers: %w", err)
	}

	url := fmt.Sprintf(targetURLTab4, username)
	if _, err = page.Goto(url); err != nil {
		return nil, fmt.Errorf("could not navigate to page: %w", err)
	}

	if err := waitForCategoryResponse(page); err != nil {
		return nil, err
	}

	selectSelector := "select"
	if _, err := page.WaitForSelector(selectSelector); err != nil {
		return nil, fmt.Errorf("could not find select element: %w", err)
	}

	time.Sleep(1 * time.Second)

	_, err = page.Evaluate(fmt.Sprintf(`() => {
		const select = document.querySelector('select');
		select.focus();
		select.value = '%s';
		select.dispatchEvent(new Event('change', { bubbles: true }));
	}`, selectedDate))
	if err != nil {
		return nil, fmt.Errorf("could not select option via JavaScript: %w", err)
	}

	// Handle withLinkStory checkbox - click element at index 6 if true
	if withLinkStory == "true" {
		_, err = page.Evaluate(`() => {
			const elements = document.querySelectorAll('.form-multi-select-option.form-multi-select-option-with-checkbox');
			if (elements.length > 6) {
				elements[6].click();
			}
		}`)
		if err != nil {
			return nil, fmt.Errorf("could not click withLinkStory checkbox at index 6: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Handle labelFilters checkboxes
	if labelFilters != "" {
		filters := strings.Split(labelFilters, ",")
		for _, filter := range filters {
			filter = strings.TrimSpace(filter)
			_, err = page.Evaluate(fmt.Sprintf(`() => {
				const elements = document.querySelectorAll('.form-multi-select-option.form-multi-select-option-with-checkbox');
				for (let element of elements) {
					const text = (element.innerText || element.textContent).trim();
					if (text === '%s') {
						element.click();
						break;
					}
				}
			}`, filter))
			if err != nil {
				return nil, fmt.Errorf("could not click label filter checkbox for '%s': %w", filter, err)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	time.Sleep(1 * time.Second)

	selector := fmt.Sprintf("#%s", elementID)
	elementHandle, err := page.WaitForSelector(selector)
	if err != nil {
		return nil, fmt.Errorf("could not find element: %w", err)
	}

	time.Sleep(1 * time.Second)

	_, err = elementHandle.EvalOnSelectorAll("h5", "h5Elements => h5Elements.forEach(h5 => h5.remove())")
	if err != nil {
		return nil, fmt.Errorf("could not remove h5 elements: %w", err)
	}

	screenshot, err := elementHandle.Screenshot(playwright.ElementHandleScreenshotOptions{
		Scale: playwright.ScreenshotScaleDevice,
	})
	if err != nil {
		return nil, fmt.Errorf("could not take screenshot: %w", err)
	}

	return screenshot, nil
}

func createPlaywrightBrowser() (*playwright.Playwright, playwright.Browser, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("could not start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless:        playwright.Bool(true),
		Args:            []string{"--disable-gpu", "--no-sandbox", "--no-zygote"},
		ChromiumSandbox: playwright.Bool(false),
	})
	if err != nil {
		pw.Stop()
		return nil, nil, fmt.Errorf("could not launch browser: %w", err)
	}

	return pw, browser, nil
}

func waitForCategoryResponse(page playwright.Page) error {
	responseChan := make(chan bool, 1)
	page.On("response", func(res playwright.Response) {
		pData, err := res.Request().PostData()
		if err != nil {
			fmt.Println("Error getting post data", err)
			return
		}
		if strings.Contains(pData, "\"operationName\":\"Influencer\"") && res.Status() == 200 {
			responseChan <- true
		}
	})

	select {
	case <-responseChan:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for /stats/category response")
	}
}

func saveToCache(cacheKey string, imgBytes []byte) {
	cachePath := filepath.Join(screenshotCache.cachePath, cacheKey+".png")
	if err := os.WriteFile(cachePath, imgBytes, 0644); err != nil {
		log.Printf("Failed to write screenshot to cache: %v", err)
		return
	}

	screenshotCache.mu.Lock()
	screenshotCache.cache[cacheKey] = CacheItem{
		filePath:  cachePath,
		timestamp: time.Now(),
	}
	screenshotCache.mu.Unlock()
}

func initCache() error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	screenshotCache = &ScreenshotCache{
		cache:     make(map[string]CacheItem),
		cachePath: cacheDir,
		mu:        sync.RWMutex{},
	}

	go cleanExpiredCache()
	return nil
}

func cleanExpiredCache() {
	for {
		time.Sleep(cachePurgeInterval)

		screenshotCache.mu.Lock()
		now := time.Now()
		for key, item := range screenshotCache.cache {
			if now.Sub(item.timestamp) > cacheExpirationTime {
				if err := os.Remove(item.filePath); err != nil {
					log.Printf("Failed to remove expired cache file: %v", err)
				}
				delete(screenshotCache.cache, key)
			}
		}
		screenshotCache.mu.Unlock()
	}
}

func installPlaywrightBrowsers() error {
	fmt.Println("Installing Playwright browsers...")
	err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
	})
	if err != nil {
		return fmt.Errorf("could not install browsers: %w", err)
	}
	fmt.Println("Playwright browsers installed successfully")
	return nil
}
