package util

import (
	"crypto/rand"
	"encoding/hex"
	"hash/fnv"
	"sync"
	"time"
)

var (
	uidPool = sync.Pool{
		New: func() any {
			return make([]byte, 16)
		},
	}
)

func GenerateUID() string {
	buf := uidPool.Get().([]byte)
	defer uidPool.Put(buf)

	rand.Read(buf)
	return hex.EncodeToString(buf)
}

func Hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func StringSliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func MergeLabels(base, override map[string]string) map[string]string {
	result := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

func MatchLabels(selector, labels map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

type RateLimiter struct {
	tokens chan struct{}
	stop   chan struct{}
}

func NewRateLimiter(rps int) *RateLimiter {
	rl := &RateLimiter{
		tokens: make(chan struct{}, rps),
		stop:   make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(rps))
		defer ticker.Stop()

		for {
			select {
			case <-rl.stop:
				return
			case <-ticker.C:
				select {
				case rl.tokens <- struct{}{}:
				default:
				}
			}
		}
	}()

	return rl
}

func (rl *RateLimiter) Wait() {
	<-rl.tokens
}

func (rl *RateLimiter) TryAcquire() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

type RetryConfig struct {
	MaxRetries int
	Delay      time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

func Retry(cfg RetryConfig, fn func() error) error {
	var err error
	delay := cfg.Delay

	for i := 0; i <= cfg.MaxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if i < cfg.MaxRetries {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * cfg.Multiplier)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}

	return err
}

type WorkerPool struct {
	workers int
	tasks   chan func()
	wg      sync.WaitGroup
	stop    chan struct{}
}

func NewWorkerPool(workers int) *WorkerPool {
	wp := &WorkerPool{
		workers: workers,
		tasks:   make(chan func(), workers*10),
		stop:    make(chan struct{}),
	}

	wp.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for {
		select {
		case <-wp.stop:
			return
		case task := <-wp.tasks:
			task()
		}
	}
}

func (wp *WorkerPool) Submit(task func()) {
	select {
	case wp.tasks <- task:
	case <-wp.stop:
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.stop)
	wp.wg.Wait()
}

type Cache[K comparable, V any] struct {
	data map[K]cacheEntry[V]
	mu   sync.RWMutex
	ttl  time.Duration
}

type cacheEntry[V any] struct {
	value   V
	expires time.Time
}

func NewCache[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		data: make(map[K]cacheEntry[V]),
		ttl:  ttl,
	}

	go c.cleanup()
	return c
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok || time.Now().After(entry.expires) {
		var zero V
		return zero, false
	}
	return entry.value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry[V]{
		value:   value,
		expires: time.Now().Add(c.ttl),
	}
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *Cache[K, V]) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.data {
			if now.After(v.expires) {
				delete(c.data, k)
			}
		}
		c.mu.Unlock()
	}
}
