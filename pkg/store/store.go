package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	ErrNotFound     = errors.New("key not found")
	ErrKeyExists    = errors.New("key already exists")
	ErrVersionConflict = errors.New("version conflict")
)

type Store interface {
	Get(ctx context.Context, key string) ([]byte, int64, error)
	List(ctx context.Context, prefix string) ([][]byte, error)
	Put(ctx context.Context, key string, value []byte) error
	Create(ctx context.Context, key string, value []byte) error
	Update(ctx context.Context, key string, value []byte, version int64) error
	Delete(ctx context.Context, key string) error
	Watch(ctx context.Context, prefix string) <-chan WatchEvent
	Close() error
}

type WatchEvent struct {
	Type  EventType
	Key   string
	Value []byte
}

type EventType int

const (
	EventPut EventType = iota
	EventDelete
)

type EtcdStore struct {
	client     *clientv3.Client
	keyPrefix  string
	bufferPool sync.Pool
}

type EtcdConfig struct {
	Endpoints   []string
	DialTimeout time.Duration
	KeyPrefix   string
}

func NewEtcdStore(cfg EtcdConfig) (*EtcdStore, error) {
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5 * time.Second
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		return nil, err
	}

	return &EtcdStore{
		client:    client,
		keyPrefix: cfg.KeyPrefix,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 4096)
			},
		},
	}, nil
}

func (s *EtcdStore) fullKey(key string) string {
	return s.keyPrefix + key
}

func (s *EtcdStore) Get(ctx context.Context, key string) ([]byte, int64, error) {
	resp, err := s.client.Get(ctx, s.fullKey(key))
	if err != nil {
		return nil, 0, err
	}
	if len(resp.Kvs) == 0 {
		return nil, 0, ErrNotFound
	}
	return resp.Kvs[0].Value, resp.Kvs[0].Version, nil
}

func (s *EtcdStore) List(ctx context.Context, prefix string) ([][]byte, error) {
	resp, err := s.client.Get(ctx, s.fullKey(prefix), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	result := make([][]byte, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		result[i] = kv.Value
	}
	return result, nil
}

func (s *EtcdStore) Put(ctx context.Context, key string, value []byte) error {
	_, err := s.client.Put(ctx, s.fullKey(key), string(value))
	return err
}

func (s *EtcdStore) Create(ctx context.Context, key string, value []byte) error {
	fullKey := s.fullKey(key)
	txn := s.client.Txn(ctx)
	txn = txn.If(clientv3.Compare(clientv3.Version(fullKey), "=", 0))
	txn = txn.Then(clientv3.OpPut(fullKey, string(value)))

	resp, err := txn.Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrKeyExists
	}
	return nil
}

func (s *EtcdStore) Update(ctx context.Context, key string, value []byte, version int64) error {
	fullKey := s.fullKey(key)
	txn := s.client.Txn(ctx)
	txn = txn.If(clientv3.Compare(clientv3.Version(fullKey), "=", version))
	txn = txn.Then(clientv3.OpPut(fullKey, string(value)))

	resp, err := txn.Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrVersionConflict
	}
	return nil
}

func (s *EtcdStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.Delete(ctx, s.fullKey(key))
	return err
}

func (s *EtcdStore) Watch(ctx context.Context, prefix string) <-chan WatchEvent {
	ch := make(chan WatchEvent, 256)

	go func() {
		defer close(ch)
		watchCh := s.client.Watch(ctx, s.fullKey(prefix), clientv3.WithPrefix())

		for resp := range watchCh {
			for _, ev := range resp.Events {
				var eventType EventType
				if ev.Type == clientv3.EventTypePut {
					eventType = EventPut
				} else {
					eventType = EventDelete
				}

				select {
				case ch <- WatchEvent{
					Type:  eventType,
					Key:   string(ev.Kv.Key),
					Value: ev.Kv.Value,
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}

func (s *EtcdStore) Close() error {
	return s.client.Close()
}

type InMemoryStore struct {
	mu      sync.RWMutex
	data    map[string]entry
	watches map[string][]chan WatchEvent
	watchMu sync.RWMutex
}

type entry struct {
	value   []byte
	version int64
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data:    make(map[string]entry),
		watches: make(map[string][]chan WatchEvent),
	}
}

func (s *InMemoryStore) Get(ctx context.Context, key string) ([]byte, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok {
		return nil, 0, ErrNotFound
	}
	return e.value, e.version, nil
}

func (s *InMemoryStore) List(ctx context.Context, prefix string) ([][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result [][]byte
	for k, v := range s.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, v.value)
		}
	}
	return result, nil
}

func (s *InMemoryStore) Put(ctx context.Context, key string, value []byte) error {
	s.mu.Lock()
	e := s.data[key]
	e.value = value
	e.version++
	s.data[key] = e
	s.mu.Unlock()

	s.notify(key, value, EventPut)
	return nil
}

func (s *InMemoryStore) Create(ctx context.Context, key string, value []byte) error {
	s.mu.Lock()
	if _, ok := s.data[key]; ok {
		s.mu.Unlock()
		return ErrKeyExists
	}
	s.data[key] = entry{value: value, version: 1}
	s.mu.Unlock()

	s.notify(key, value, EventPut)
	return nil
}

func (s *InMemoryStore) Update(ctx context.Context, key string, value []byte, version int64) error {
	s.mu.Lock()
	e, ok := s.data[key]
	if !ok {
		s.mu.Unlock()
		return ErrNotFound
	}
	if e.version != version {
		s.mu.Unlock()
		return ErrVersionConflict
	}
	e.value = value
	e.version++
	s.data[key] = e
	s.mu.Unlock()

	s.notify(key, value, EventPut)
	return nil
}

func (s *InMemoryStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()

	s.notify(key, nil, EventDelete)
	return nil
}

func (s *InMemoryStore) Watch(ctx context.Context, prefix string) <-chan WatchEvent {
	ch := make(chan WatchEvent, 256)

	s.watchMu.Lock()
	s.watches[prefix] = append(s.watches[prefix], ch)
	s.watchMu.Unlock()

	go func() {
		<-ctx.Done()
		s.watchMu.Lock()
		chs := s.watches[prefix]
		for i, c := range chs {
			if c == ch {
				s.watches[prefix] = append(chs[:i], chs[i+1:]...)
				break
			}
		}
		s.watchMu.Unlock()
		close(ch)
	}()

	return ch
}

func (s *InMemoryStore) notify(key string, value []byte, eventType EventType) {
	s.watchMu.RLock()
	defer s.watchMu.RUnlock()

	event := WatchEvent{Type: eventType, Key: key, Value: value}

	for prefix, chs := range s.watches {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			for _, ch := range chs {
				select {
				case ch <- event:
				default:
				}
			}
		}
	}
}

func (s *InMemoryStore) Close() error {
	return nil
}

type CachedStore struct {
	backend Store
	cache   sync.Map
	ttl     time.Duration
}

type cacheEntry struct {
	value   []byte
	version int64
	expires time.Time
}

func NewCachedStore(backend Store, ttl time.Duration) *CachedStore {
	return &CachedStore{
		backend: backend,
		ttl:     ttl,
	}
}

func (s *CachedStore) Get(ctx context.Context, key string) ([]byte, int64, error) {
	if v, ok := s.cache.Load(key); ok {
		e := v.(cacheEntry)
		if time.Now().Before(e.expires) {
			return e.value, e.version, nil
		}
		s.cache.Delete(key)
	}

	value, version, err := s.backend.Get(ctx, key)
	if err != nil {
		return nil, 0, err
	}

	s.cache.Store(key, cacheEntry{
		value:   value,
		version: version,
		expires: time.Now().Add(s.ttl),
	})

	return value, version, nil
}

func (s *CachedStore) List(ctx context.Context, prefix string) ([][]byte, error) {
	return s.backend.List(ctx, prefix)
}

func (s *CachedStore) Put(ctx context.Context, key string, value []byte) error {
	s.cache.Delete(key)
	return s.backend.Put(ctx, key, value)
}

func (s *CachedStore) Create(ctx context.Context, key string, value []byte) error {
	return s.backend.Create(ctx, key, value)
}

func (s *CachedStore) Update(ctx context.Context, key string, value []byte, version int64) error {
	s.cache.Delete(key)
	return s.backend.Update(ctx, key, value, version)
}

func (s *CachedStore) Delete(ctx context.Context, key string) error {
	s.cache.Delete(key)
	return s.backend.Delete(ctx, key)
}

func (s *CachedStore) Watch(ctx context.Context, prefix string) <-chan WatchEvent {
	return s.backend.Watch(ctx, prefix)
}

func (s *CachedStore) Close() error {
	return s.backend.Close()
}

func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
