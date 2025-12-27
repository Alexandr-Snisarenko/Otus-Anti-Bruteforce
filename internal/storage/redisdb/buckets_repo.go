package redisdb

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

type BucketsRepo struct {
	client *redis.Client
}

func NewBucketsRepo(rdc *redis.Client) *BucketsRepo {
	return &BucketsRepo{client: rdc}
}

// Allow проверяет, можно ли выполнить действие, ограниченное лимитом.
// key — уникальный ключ для ограничения (например, "login:username").
// limit — максимальное количество действий в окне window.
// window — временное окно для подсчета лимита.
// Возвращает true, если действие разрешено, и false, если превышен лимит.
// Используем паттерн "скользящее окно" на основе  Redis Sorted Set (ZSET)
// с Member = текущее время в миллисекундах + случайное число.
// В такой комбинации повторение Member маловероятно,
// Считаем такую уникальность приемлимой, чтобы избежать коллизий при одновременных запросах.
// TTL ключа устанавливаем в два раза больше окна, чтобы данные не накапливались бесконечно.
//
// Таким образом, по каждому ключу (например конкретному логину) в Redis создаётся отдельный ZSET,
// в котором хранятся события обращения за проверкой — временные метки запросов.
// Для проверки лимита - считаем количество элементов в ZSET, соответствующих текущему окну времени.
// Если количество элементов меньше или равно лимиту - разрешаем действие.
// Алгоритм:
// 1. Добавляем текущий запрос с текущей временной меткой.
// 2. Удаляем из множества все запросы старше текущего времени минус окно.
// 3. Считаем количество оставшихся запросов в множестве.
// 4. Если количество меньше или равно лимиту — разрешаем действие.

func (c *BucketsRepo) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UnixMilli()
	pipe := c.client.TxPipeline()

	// 1. Добавляем текущий запрос
	// генерируем случайное число для уникальности Member
	sfx, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return false, err
	}
	// Member = "<текущее время в миллисекундах>:<случайное число>"
	member := fmt.Sprintf("%d:%d", now, sfx)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: member})

	// 2. Удаляем устаревшие
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprint(now-window.Milliseconds()))

	// 3. Считаем, сколько осталось
	count := pipe.ZCard(ctx, key)

	// 4. Обновляем TTL
	pipe.Expire(ctx, key, window*2)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	n, _ := count.Result()
	return n <= int64(limit), nil
}

func (c *BucketsRepo) Reset(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
