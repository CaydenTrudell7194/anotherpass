package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

type GeoResult struct {
	CountryCode string `json:"country_code"`
}

type geoCacheEntry struct {
	Result    GeoResult
	ExpiresAt time.Time
}

var geoCache = struct {
	sync.RWMutex
	items map[string]geoCacheEntry
}{items: make(map[string]geoCacheEntry)}

func resolveGeo(ip string) GeoResult {
	if net.ParseIP(ip) == nil {
		return GeoResult{}
	}
	geoCache.RLock()
	entry, ok := geoCache.items[ip]
	geoCache.RUnlock()
	if ok && time.Now().Before(entry.ExpiresAt) {
		return entry.Result
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/" + ip + "?fields=countryCode")
	if err != nil {
		return GeoResult{}
	}
	defer resp.Body.Close()
	var result GeoResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return GeoResult{}
	}
	geoCache.Lock()
	geoCache.items[ip] = geoCacheEntry{Result: result, ExpiresAt: time.Now().Add(24 * time.Hour)}
	if len(geoCache.items) > 10000 {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range geoCache.items {
			if oldestKey == "" || v.ExpiresAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.ExpiresAt
			}
		}
		if oldestKey != "" {
			delete(geoCache.items, oldestKey)
		}
	}
	geoCache.Unlock()
	return result
}
