package cache

import (
	"context"
	"database/sql"
	"time"

	"github.com/suisrc/zgg/z/ze/sqlx"
)

// NewCacheMongo 创建基于 sqlx 存储实例, 未完成实现

type CacheSqlxData struct {
	ID        string         `db:"id"`
	Value     sql.NullString `db:"label"`
	CreatedAt time.Time      `db:"created_at"`
	ExpiredAt time.Time      `db:"expired_at"`
}

func (CacheSqlxData) TableName() string {
	return "xxx"
}

var _ Cache = new(SqlxCache)

// SqlxCache
type SqlxCache struct {
	dsc *sqlx.DB
}

// Get ...
func (s *SqlxCache) Get(ctx context.Context, key string) (string, bool, error) {
	res := CacheSqlxData{}

	// if err := s.dsc.Find(res, key); err != nil {
	// 	return "", false, err
	// }

	return res.Value.String, true, nil
}

// Set ... Save 包含create和update
func (s *SqlxCache) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	// res := CacheSqlxData{}
	// s.dsc.Find(res, key)
	// if res.ID != "" {
	// 	res = CacheSqlxData{
	// 		ID:        res.ID,
	// 		Value:     sql.NullString{String: value, Valid: true},
	// 		CreatedAt: res.CreatedAt,
	// 		ExpiredAt: time.Now().Add(expiration),
	// 	}
	// 	if err := s.dsc.Update(res); err != nil {
	// 		return err
	// 	}
	// } else {
	// 	res = CacheSqlxData{
	// 		ID:        key,
	// 		Value:     sql.NullString{String: value, Valid: true},
	// 		CreatedAt: time.Now(),
	// 		ExpiredAt: time.Now().Add(expiration),
	// 	}
	// 	// verr, err := c.ValidateAndSave(res)
	// 	if err := s.dsc.Create(res); err != nil {
	// 		return err
	// 	}
	// }
	return nil
}

// Set1 ...
func (s *SqlxCache) Set1(ctx context.Context, key string, expiration time.Duration) error {
	return s.Set(ctx, key, "1", expiration)
}

func (s *SqlxCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	// res := []CacheSqlxData{}
	// err := s.dsc.Select("id").Where("id like ?", strings.ReplaceAll(pattern, "*", "%")).All(res)
	// if err != nil {
	// 	return nil, err
	// }
	// keys := []string{}
	// for _, r := range res {
	// 	keys = append(keys, r.ID)
	// }
	// return keys, nil
	return []string{}, nil
}

// Expire ...
func (s *SqlxCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	// res := CacheSqlxData{}
	// if err := s.dsc.Find(res, key); err != nil {
	// 	return err
	// }
	// if res.ID != "" {
	// 	return errors.New("not exist")
	// }
	// res.ExpiredAt = time.Now().Add(expiration)
	// if err := s.dsc.Update(res, "value", "created_at"); err != nil {
	// 	return err
	// }
	return nil
}

// Exists ...
func (s *SqlxCache) Exists(ctx context.Context, key string) (bool, error) {
	_, exist, err := s.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return exist, nil
}

// Delete ...
func (s *SqlxCache) Delete(ctx context.Context, key string) error {
	// res := CacheSqlxData{ID: key}
	// if err := s.dsc.Destroy(res); err != nil {
	// 	return err
	// }
	return nil
}

func (s *SqlxCache) DelAll(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		s.Delete(ctx, key)
	}
	return nil
}

// Start ...
func (s *SqlxCache) Start(ctx context.Context) error {
	// cli, err := sqlx.ConnectDatabase("")
	// if err != nil {
	// 	return err
	// }
	// s.cli = cli
	return nil
}

// Close ...
func (s *SqlxCache) Close(ctx context.Context) error {
	s.dsc.Close()
	return nil
}
