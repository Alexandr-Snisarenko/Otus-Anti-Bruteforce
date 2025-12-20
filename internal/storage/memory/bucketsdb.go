package memory

import (
	"context"
	"sync"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/ports"
)

var _ ports.LimiterRepo = (*BucketsDB)(nil)

// BucketsDB — простая in-memory реализация скользящего окна.
// Ключ - идентификатор бакета (например, IP или логин).
// Значение - таймлайн попыток.
type BucketsDB struct {
	mu   sync.RWMutex
	data map[string]*timeline
}

type timeline struct {
	mu    sync.Mutex
	times []int64 // список попыток (уникальных записей - время попытки в миллисекундах)
}

// NewBucketsDB возвращает новую in-memory реализацию.
func NewBucketsDB() *BucketsDB {
	return &BucketsDB{
		data: make(map[string]*timeline),
	}
}

func (l *BucketsDB) getTimeline(key string) *timeline {
	l.mu.RLock()
	tl := l.data[key]
	l.mu.RUnlock()
	if tl != nil {
		return tl
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	// double-check
	if tl = l.data[key]; tl == nil {
		tl = &timeline{times: make([]int64, 0, 16)}
		l.data[key] = tl
	}
	return tl
}

// Allow — возвращает true, если попытка разрешена (в пределах лимита).
func (l *BucketsDB) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 {
		return false, nil
	}
	if window <= 0 {
		return true, nil
	}

	now := time.Now()
	tl := l.getTimeline(key)

	tl.mu.Lock()
	defer tl.mu.Unlock()

	cut := now.Add(-window).UnixMilli()

	// Удаляем старые попытки за пределами окна
	i := 0
	for i < len(tl.times) && tl.times[i] <= cut {
		i++
	}
	if i > 0 {
		tl.times = tl.times[i:]
	}

	// Проверяем, не превышен ли лимит
	if len(tl.times) >= limit {
		return false, nil
	}

	// Добавляем новую попытку
	tl.times = append(tl.times, now.UnixMilli())
	return true, nil
}

// Reset — сбрасывает все попытки по ключу.
func (l *BucketsDB) Reset(_ context.Context, key string) error {
	l.mu.RLock()
	tl := l.data[key]
	l.mu.RUnlock()
	if tl == nil {
		return nil
	}

	tl.mu.Lock()
	tl.times = tl.times[:0]
	tl.mu.Unlock()
	return nil
}
