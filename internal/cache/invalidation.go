package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"videostreamgo/internal/config"
)

// InvalidationChannel is the Redis pub/sub channel for cache invalidation events
const InvalidationChannel = "cache:invalidate"

// CacheVersion holds the current cache schema version
var CacheVersion = struct {
	Major int
	Minor int
}{
	Major: 1,
	Minor: 0,
}

// GetCacheVersionString returns the cache version as a string
func GetCacheVersionString() string {
	return fmt.Sprintf("v%d.%d", CacheVersion.Major, CacheVersion.Minor)
}

// InvalidationEvent represents a cache invalidation event
type InvalidationEvent struct {
	EventType  string            `json:"event_type"`  // "invalidate", "invalidate_pattern", "flush"
	EntityType string            `json:"entity_type"` // "user", "video", "category", etc.
	EntityID   string            `json:"entity_id"`   // Specific entity ID or empty for pattern
	Pattern    string            `json:"pattern"`     // Redis key pattern for bulk invalidation
	Version    string            `json:"version"`     // Cache version at time of invalidation
	Timestamp  time.Time         `json:"timestamp"`
	ServiceID  string            `json:"service_id"` // ID of the service that triggered the event
	Metadata   map[string]string `json:"metadata"`   // Additional metadata
}

// InvalidationSubscriber handles subscribing to cache invalidation events
type InvalidationSubscriber struct {
	client        *RedisClient
	subscriptions map[string]*redis.PubSub
	handlers      map[string][]InvalidationHandler
	mu            sync.RWMutex
	stopChan      chan struct{}
	wg            sync.WaitGroup
	isConnected   bool
}

// InvalidationHandler is a function that handles invalidation events
type InvalidationHandler func(event InvalidationEvent) error

// NewInvalidationSubscriber creates a new invalidation subscriber
func NewInvalidationSubscriber(client *RedisClient) *InvalidationSubscriber {
	return &InvalidationSubscriber{
		client:        client,
		subscriptions: make(map[string]*redis.PubSub),
		handlers:      make(map[string][]InvalidationHandler),
		stopChan:      make(chan struct{}),
		isConnected:   false,
	}
}

// Subscribe starts subscribing to invalidation events
func (s *InvalidationSubscriber) Subscribe(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isConnected {
		return nil
	}

	// Subscribe to the main invalidation channel
	pubsub := s.client.GetClient().Subscribe(ctx, InvalidationChannel)

	// Wait for subscription confirmation
	if _, err := pubsub.Receive(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to invalidation channel: %w", err)
	}

	s.subscriptions[InvalidationChannel] = pubsub
	s.isConnected = true

	// Start listening for messages
	s.wg.Add(1)
	go s.handleMessages(ctx, pubsub)

	log.Printf("Cache invalidation subscriber connected to channel: %s", InvalidationChannel)
	return nil
}

// handleMessages processes incoming invalidation messages
func (s *InvalidationSubscriber) handleMessages(ctx context.Context, pubsub *redis.PubSub) {
	defer s.wg.Done()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			var event InvalidationEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("Failed to unmarshal invalidation event: %v", err)
				continue
			}

			// Skip events from this service (prevent echo)
			if event.ServiceID == s.client.GetClient().Options().Addr {
				continue
			}

			// Process the event
			if err := s.processEvent(event); err != nil {
				log.Printf("Failed to process invalidation event: %v", err)
			}
		}
	}
}

// processEvent processes a single invalidation event
func (s *InvalidationSubscriber) processEvent(event InvalidationEvent) error {
	s.mu.RLock()
	handlers, exists := s.handlers[event.EntityType]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	for _, handler := range handlers {
		if err := handler(event); err != nil {
			return err
		}
	}

	return nil
}

// RegisterHandler registers a handler for a specific entity type
func (s *InvalidationSubscriber) RegisterHandler(entityType string, handler InvalidationHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[entityType] = append(s.handlers[entityType], handler)
}

// Unsubscribe stops all subscriptions
func (s *InvalidationSubscriber) Unsubscribe() {
	close(s.stopChan)
	s.wg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, pubsub := range s.subscriptions {
		pubsub.Close()
	}
	s.subscriptions = make(map[string]*redis.PubSub)
	s.isConnected = false
	log.Println("Cache invalidation subscriber disconnected")
}

// InvalidationPublisher handles publishing cache invalidation events
type InvalidationPublisher struct {
	client    *RedisClient
	serviceID string
	mu        sync.Mutex
}

// NewInvalidationPublisher creates a new invalidation publisher
func NewInvalidationPublisher(client *RedisClient, serviceID string) *InvalidationPublisher {
	return &InvalidationPublisher{
		client:    client,
		serviceID: serviceID,
	}
}

// PublishInvalidate publishes an invalidation event for a specific entity
func (p *InvalidationPublisher) PublishInvalidate(ctx context.Context, entityType, entityID string) error {
	return p.publishEvent(ctx, InvalidationEvent{
		EventType:  "invalidate",
		EntityType: entityType,
		EntityID:   entityID,
		Version:    GetCacheVersionString(),
		Timestamp:  time.Now(),
		ServiceID:  p.serviceID,
	})
}

// PublishInvalidatePattern publishes an invalidation event for entities matching a pattern
func (p *InvalidationPublisher) PublishInvalidatePattern(ctx context.Context, entityType, pattern string) error {
	return p.publishEvent(ctx, InvalidationEvent{
		EventType:  "invalidate_pattern",
		EntityType: entityType,
		Pattern:    pattern,
		Version:    GetCacheVersionString(),
		Timestamp:  time.Now(),
		ServiceID:  p.serviceID,
	})
}

// PublishFlush publishes a flush event for an entire entity type
func (p *InvalidationPublisher) PublishFlush(ctx context.Context, entityType string) error {
	return p.publishEvent(ctx, InvalidationEvent{
		EventType:  "flush",
		EntityType: entityType,
		Version:    GetCacheVersionString(),
		Timestamp:  time.Now(),
		ServiceID:  p.serviceID,
	})
}

// publishEvent publishes an invalidation event to Redis pub/sub
func (p *InvalidationPublisher) publishEvent(ctx context.Context, event InvalidationEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	event.Version = GetCacheVersionString()
	event.Timestamp = time.Now()
	event.ServiceID = p.serviceID

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal invalidation event: %w", err)
	}

	return p.client.GetClient().Publish(ctx, InvalidationChannel, string(data)).Err()
}

// CacheWarmer handles warming cache on service startup
type CacheWarmer struct {
	client   *RedisClient
	fetchers map[string]CacheFetcher
	mu       sync.RWMutex
}

// CacheFetcher is a function that fetches data to warm the cache
type CacheFetcher func(ctx context.Context) error

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(client *RedisClient) *CacheWarmer {
	return &CacheWarmer{
		client:   client,
		fetchers: make(map[string]CacheFetcher),
	}
}

// RegisterFetcher registers a cache fetcher for a specific entity type
func (w *CacheWarmer) RegisterFetcher(entityType string, fetcher CacheFetcher) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.fetchers[entityType] = fetcher
}

// WarmUp performs cache warming for all registered fetchers
func (w *CacheWarmer) WarmUp(ctx context.Context) error {
	w.mu.RLock()
	fetchers := w.fetchers
	w.mu.RUnlock()

	log.Printf("Starting cache warming for %d entity types", len(fetchers))

	var failed []string
	for entityType, fetcher := range fetchers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := fetcher(ctx); err != nil {
				log.Printf("Failed to warm cache for %s: %v", entityType, err)
				failed = append(failed, entityType)
			} else {
				log.Printf("Cache warmed for %s", entityType)
			}
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to warm cache for: %v", failed)
	}

	log.Println("Cache warming completed")
	return nil
}

// InvalidationManager coordinates cache invalidation across the service
type InvalidationManager struct {
	publisher  *InvalidationPublisher
	subscriber *InvalidationSubscriber
	warmer     *CacheWarmer
	client     *RedisClient
	serviceID  string
}

// NewInvalidationManager creates a new invalidation manager
func NewInvalidationManager(client *RedisClient, cfg *config.Config) *InvalidationManager {
	serviceID := fmt.Sprintf("%s-%d", cfg.App.ServiceIdentifier, cfg.App.Port)

	return &InvalidationManager{
		publisher:  NewInvalidationPublisher(client, serviceID),
		subscriber: NewInvalidationSubscriber(client),
		warmer:     NewCacheWarmer(client),
		client:     client,
		serviceID:  serviceID,
	}
}

// Initialize starts the invalidation manager
func (m *InvalidationManager) Initialize(ctx context.Context) error {
	// Subscribe to invalidation events
	if err := m.subscriber.Subscribe(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to invalidation events: %w", err)
	}

	// Start cache warming
	if err := m.warmer.WarmUp(ctx); err != nil {
		log.Printf("Warning: Cache warming failed: %v", err)
	}

	return nil
}

// Shutdown gracefully shuts down the invalidation manager
func (m *InvalidationManager) Shutdown() {
	m.subscriber.Unsubscribe()
}

// GetPublisher returns the invalidation publisher
func (m *InvalidationManager) GetPublisher() *InvalidationPublisher {
	return m.publisher
}

// GetWarmer returns the cache warmer
func (m *InvalidationManager) GetWarmer() *CacheWarmer {
	return m.warmer
}

// GetSubscriber returns the invalidation subscriber
func (m *InvalidationManager) GetSubscriber() *InvalidationSubscriber {
	return m.subscriber
}

// InvalidateUser invalidates user-related cache entries
func (m *InvalidationManager) InvalidateUser(ctx context.Context, userID string) error {
	// Invalidate specific user cache
	if err := m.client.Delete(ctx, fmt.Sprintf("user:%s", userID)); err != nil {
		log.Printf("Failed to delete user cache: %v", err)
	}

	// Publish invalidation event to other instances
	return m.publisher.PublishInvalidate(ctx, "user", userID)
}

// InvalidateVideo invalidates video-related cache entries
func (m *InvalidationManager) InvalidateVideo(ctx context.Context, videoID string) error {
	// Delete specific video cache
	if err := m.client.Delete(ctx, fmt.Sprintf("video:%s", videoID)); err != nil {
		log.Printf("Failed to delete video cache: %v", err)
	}

	// Invalidate related lists (e.g., user videos, category videos)
	pattern := fmt.Sprintf("*video*%s*", videoID)
	keys, _ := m.client.Keys(ctx, pattern)
	if len(keys) > 0 {
		m.client.Delete(ctx, keys...)
	}

	// Publish invalidation event
	return m.publisher.PublishInvalidate(ctx, "video", videoID)
}

// InvalidateCategory invalidates category-related cache entries
func (m *InvalidationManager) InvalidateCategory(ctx context.Context, categoryID string) error {
	// Delete category cache
	if err := m.client.Delete(ctx, fmt.Sprintf("category:%s", categoryID)); err != nil {
		log.Printf("Failed to delete category cache: %v", err)
	}

	// Invalidate video lists for this category
	pattern := fmt.Sprintf("*category:%s:videos*", categoryID)
	keys, _ := m.client.Keys(ctx, pattern)
	if len(keys) > 0 {
		m.client.Delete(ctx, keys...)
	}

	// Publish invalidation event
	return m.publisher.PublishInvalidate(ctx, "category", categoryID)
}

// InvalidateInstance invalidates instance-related cache entries
func (m *InvalidationManager) InvalidateInstance(ctx context.Context, instanceID string) error {
	// Delete instance cache
	if err := m.client.Delete(ctx, fmt.Sprintf("instance:%s", instanceID)); err != nil {
		log.Printf("Failed to delete instance cache: %v", err)
	}

	// Publish invalidation event
	return m.publisher.PublishInvalidate(ctx, "instance", instanceID)
}

// InvalidateTenant invalidates tenant configuration cache
func (m *InvalidationManager) InvalidateTenant(ctx context.Context, tenantID string) error {
	// Delete tenant config cache
	if err := m.client.Delete(ctx, fmt.Sprintf("tenant:%s:config", tenantID)); err != nil {
		log.Printf("Failed to delete tenant config cache: %v", err)
	}

	// Publish invalidation event
	return m.publisher.PublishInvalidate(ctx, "tenant", tenantID)
}
