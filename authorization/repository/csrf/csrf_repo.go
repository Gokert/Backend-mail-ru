package csrf

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	"github.com/go-redis/redis/v8"
)

var mutex sync.RWMutex

type CsrfRepo struct {
	csrfRedisClient *redis.Client
	Connection      bool
}

func (redisRepo *CsrfRepo) CheckRedisCsrfConnection(csrfCfg configs.DbRedisCfg) {
	ctx := context.Background()
	for {
		_, err := redisRepo.csrfRedisClient.Ping(ctx).Result()
		mutex.RLock()
		mutex.Lock()
		redisRepo.Connection = err == nil
		mutex.Unlock()
		mutex.RUnlock()

		time.Sleep(time.Duration(csrfCfg.Timer) * time.Second)
	}
}

func GetCsrfRepo(csrfConfigs configs.DbRedisCfg, lg *slog.Logger) (*CsrfRepo, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     csrfConfigs.Host,
		Password: csrfConfigs.Password,
		DB:       csrfConfigs.DbNumber,
	})

	ctx := context.Background()

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	csrfRepo := CsrfRepo{
		csrfRedisClient: redisClient,
		Connection:      true,
	}

	go csrfRepo.CheckRedisCsrfConnection(csrfConfigs)

	return &csrfRepo, nil
}

func (redisRepo *CsrfRepo) AddCsrf(ctx context.Context, active models.Csrf, lg *slog.Logger) (bool, error) {
	if !redisRepo.Connection {
		lg.Error("Redis csrf connection lost")
		return false, nil
	}

	redisRepo.csrfRedisClient.Set(ctx, active.SID, active.SID, 3*time.Hour)

	csrfAdded, err_check := redisRepo.CheckActiveCsrf(ctx, active.SID, lg)

	if err_check != nil {
		lg.Error("Error, cannot create csrf token " + err_check.Error())
		return false, err_check
	}

	return csrfAdded, nil
}

func (redisRepo *CsrfRepo) CheckActiveCsrf(ctx context.Context, sid string, lg *slog.Logger) (bool, error) {
	if !redisRepo.Connection {
		lg.Error("Redis csrf connection lost")
		return false, nil
	}

	_, err := redisRepo.csrfRedisClient.Get(ctx, sid).Result()
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

func (redisRepo *CsrfRepo) DeleteSession(ctx context.Context, sid string, lg *slog.Logger) (bool, error) {
	_, err := redisRepo.csrfRedisClient.Del(ctx, sid).Result()
	if err != nil {
		lg.Error("Delete request could not be completed:", err)
		return false, err
	}

	return true, nil
}
