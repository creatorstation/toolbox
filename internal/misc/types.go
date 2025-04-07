package misc

import (
	"sync"
	"time"
)

// ScreenshotCache stores and manages cached screenshots
type ScreenshotCache struct {
	cache     map[string]CacheItem
	cachePath string
	mu        sync.RWMutex
}

// CacheItem represents a single cached screenshot
type CacheItem struct {
	filePath  string
	timestamp time.Time
}
