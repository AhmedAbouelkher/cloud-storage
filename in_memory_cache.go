package main

import (
	"strings"
	"time"

	"golang.org/x/sync/syncmap"
)

var cache syncmap.Map
var ttlCache map[string]time.Time

type CValue any
type CKey []string

type CacheEntry struct {
	Key   CKey
	Value CacheValue
}

type CacheValue struct {
	Value CValue
	TTL   time.Duration
}

func OpenInMemoryCache() {
	cache = syncmap.Map{}
	ttlCache = map[string]time.Time{}
	go func() {
		for now := range time.Tick(time.Second) {
			if len(ttlCache) == 0 {
				continue
			}
			for key, t := range ttlCache {
				if t.Before(now) {
					cache.Delete(key)
				}
			}
		}
	}()
}

func MCacheSet(ce *CacheEntry) {
	key := constructCacheKey(ce.Key...)
	ttlCache[key] = time.Now().Add(ce.Value.TTL * time.Second)
	cache.Store(key, ce.Value)
}

func MCacheDel(k CKey) {
	key := constructCacheKey(k...)
	cache.Delete(key)
	delete(ttlCache, key)
}

func MCacheGet(k CKey) (*CacheValue, bool) {
	key := constructCacheKey(k...)
	if v, ok := cache.Load(key); ok {
		val := v.(CacheValue)
		return &val, true
	}
	return nil, false
}

func constructCacheKey(ks ...string) string {
	return "cache:" + strings.Join(ks, ":")
}

func PreviewAllMCache() []map[string]any {
	entries := []map[string]any{}
	cache.Range(func(key, value any) bool {
		entries = append(entries, map[string]any{
			"Value": value,
			"Key":   key,
		})
		return true
	})
	return entries
}
