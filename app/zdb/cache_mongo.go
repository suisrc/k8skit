package zdb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	//readpref "go.mongodb.org/mongo-driver/mongo/readpref"
)

// NewCacheMongo 创建基于 mongo 存储实例, 未完成实现
func NewCacheMongo(mdb *mongo.Database) *MongoCache {
	cll := mdb.Collection("xxx")
	return &MongoCache{
		cll: cll,
	}
}

var _ Cache = new(MongoCache)

// MongoCache 存储
type MongoCache struct {
	cll *mongo.Collection
}

// Get ...
func (s *MongoCache) Get(ctx context.Context, key string) (string, bool, error) {
	filter := bson.M{"id": key}
	res := s.cll.FindOne(ctx, filter)
	if res != nil {
		return "", false, nil
	}
	if err := res.Err(); err != nil {
		return "", false, err
	}
	data := bson.M{}
	if err := res.Decode(&data); err != nil {
		return "", false, err
	}
	return data["value"].(string), true, nil
}

// Set ...
func (s *MongoCache) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	data := bson.M{"id": key, "value": value, "created_time": time.Now(), "expired_time": time.Now().Add(expiration)}
	_, err := s.cll.InsertOne(ctx, data)
	if err != nil {
		return err
	}
	return nil
}

// Set1 ...
func (s *MongoCache) Set1(ctx context.Context, key string, expiration time.Duration) error {
	data := bson.M{"id": key, "value": "1", "created_time": time.Now(), "expired_time": time.Now().Add(expiration)}
	_, err := s.cll.InsertOne(ctx, data)
	if err != nil {
		return err
	}
	return nil
}

// Expire ...
func (s *MongoCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	value, exist, err := s.Get(ctx, key)
	if err != nil {
		return err
	} else if !exist {
		return errors.New("not exist")
	}
	return s.Set(ctx, key, value, expiration)
}

// Exists ...
func (s *MongoCache) Exists(ctx context.Context, key string) (bool, error) {
	_, exist, err := s.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return exist, nil
}

// Delete ...
func (s *MongoCache) Delete(ctx context.Context, key string) error {
	filter := bson.M{"id": key}
	res, err := s.cll.DeleteOne(ctx, filter)
	if err != nil {
		return err
	} else if res.DeletedCount == 0 {
		return errors.New("not exist")
	}
	return nil
}

func (s *MongoCache) DelAll(ctx context.Context, keys ...string) error {
	return errors.New("not implement")

}

func (s *MongoCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return nil, errors.New("not implement")
}

// ==================================================================================

// Start ...
func (s *MongoCache) Start(ctx context.Context) error {
	// idx := mongo.IndexModel{
	// 	Keys:    bson.M{"expired_time": 1},                // 设置TTL索引列"expired_time"
	// 	Options: options.Index().SetExpireAfterSeconds(1), // 设置过期时间1s
	// }
	// _, err = cll.Indexes().CreateOne(ctx, idx) // 创建TTL
	// if err != nil {
	// 	return err
	// }
	return nil
}

// Close ...
func (s *MongoCache) Close(ctx context.Context) error {
	return nil
}
