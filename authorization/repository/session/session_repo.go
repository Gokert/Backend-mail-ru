package session

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-redis/redis/v8"
)

var mutex sync.RWMutex

type SessionRepo struct {
	sessionRedisClient *redis.Client
	Connection         bool
}

func (redisRepo *SessionRepo) CheckRedisSessionConnection(sessionCfg configs.DbRedisCfg) {
	ctx := context.Background()
	for {
		_, err := redisRepo.sessionRedisClient.Ping(ctx).Result()
		mutex.Lock()
		mutex.RLock()
		redisRepo.Connection = err == nil
		mutex.Unlock()
		mutex.RUnlock()
		time.Sleep(time.Duration(sessionCfg.Timer) * time.Second)
	}
}

func GetSessionRepo(sessionCfg configs.DbRedisCfg, lg *slog.Logger) (*SessionRepo, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     sessionCfg.Host,
		Password: sessionCfg.Password,
		DB:       sessionCfg.DbNumber,
	})

	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	sessionRepo := SessionRepo{
		sessionRedisClient: redisClient,
		Connection:         true,
	}

	go sessionRepo.CheckRedisSessionConnection(sessionCfg)

	return &sessionRepo, nil
}

func (redisRepo *SessionRepo) AddSession(ctx context.Context, active Session, lg *slog.Logger) (bool, error) {
	if !redisRepo.Connection {
		lg.Error("Redis session connection lost")
		return false, nil
	}

	redisRepo.sessionRedisClient.Set(ctx, active.SID, active.Login, 24*time.Hour)

	sessionAdded, err_check := redisRepo.CheckActiveSession(ctx, active.SID, lg)

	if err_check != nil {
		return false, err_check
	}

	return sessionAdded, nil
}

func (redisRepo *SessionRepo) GetUserLogin(ctx context.Context, sid string, lg *slog.Logger) (string, error) {
	if !redisRepo.Connection {
		lg.Error("Redis session connection lost")
		return "", nil
	}

	value, err := redisRepo.sessionRedisClient.Get(ctx, sid).Result()
	if err != nil {
		lg.Error("Error, cannot find session " + sid)
		return "", err
	}

	return value, nil
}

func (redisRepo *SessionRepo) CheckActiveSession(ctx context.Context, sid string, lg *slog.Logger) (bool, error) {
	if !redisRepo.Connection {
		lg.Error("Redis session connection lost")
		return false, nil
	}

	_, err := redisRepo.sessionRedisClient.Get(ctx, sid).Result()
	if err == redis.Nil {
		lg.Error("Key " + sid + " not found")
		return false, nil
	}

	if err != nil {
		lg.Error("Get request could not be completed ", err)
		return false, err
	}

	return true, nil
}

func (redisRepo *SessionRepo) DeleteSession(ctx context.Context, sid string, lg *slog.Logger) (bool, error) {
	_, err := redisRepo.sessionRedisClient.Del(ctx, sid).Result()
	if err != nil {
		lg.Error("Delete request could not be completed:", err)
		return false, err
	}

	return true, nil
}
