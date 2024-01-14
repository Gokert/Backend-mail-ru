package film

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	"github.com/go-redis/redis/v8"
)

var mutex sync.RWMutex

type FilmRedisRepo struct {
	filmRedisClient    *redis.Client
	Connection         bool
}

func (redisRepo *FilmRedisRepo) CheckRedisNearFilmConnection(NearFilmCfg configs.DbRedisCfg) {
	ctx := context.Background()
	for {
		_, err := redisRepo.filmRedisClient.Ping(ctx).Result()
		mutex.Lock()
		mutex.RLock()
		redisRepo.Connection = err == nil
		mutex.Unlock()
		mutex.RUnlock()
		time.Sleep(time.Duration(NearFilmCfg.Timer) * time.Second)
	}
}

func GetFilmRedisRepo(NearFilmCfg configs.DbRedisCfg, lg *slog.Logger) (*FilmRedisRepo, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     NearFilmCfg.Host,
		Password: NearFilmCfg.Password,
		DB:       NearFilmCfg.DbNumber,
	})

	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	FilmRedisRepo := FilmRedisRepo{
		filmRedisClient:    redisClient,
		Connection:         true,
	}

	go FilmRedisRepo.CheckRedisNearFilmConnection(NearFilmCfg)

	return &FilmRedisRepo, nil
}

func (redisRepo *FilmRedisRepo) AddNearFilm(ctx context.Context, active models.NearFilm, lg *slog.Logger) (bool, error) {
	if !redisRepo.Connection {
		lg.Error("Redis NearFilm connection lost")
		return false, nil
	}

	_, err := redisRepo.filmRedisClient.HSet(ctx, "nearfilms:"+strconv.FormatUint(active.IdUser, 10), strconv.FormatUint(active.IdFilm, 10), "1").Result()
	if err != nil {
		return false, err
	}

	NearFilmAdded, err := redisRepo.CheckActiveNearFilm(ctx, strconv.FormatUint(active.IdUser, 10), strconv.FormatUint(active.IdFilm, 10), lg)
	if err != nil {
		return false, err
	}

	return NearFilmAdded, nil
}

func (redisRepo *FilmRedisRepo) CheckActiveNearFilm(ctx context.Context, uid string, fid string, lg *slog.Logger) (bool, error) {
	if !redisRepo.Connection {
		lg.Error("Redis NearFilm connection lost")
		return false, nil
	}

	exists, err := redisRepo.filmRedisClient.HExists(ctx, "nearfilms:"+uid, fid).Result()
	if err != nil {
		lg.Error("HExists request could not be completed ", err)
		return false, err
	}

	return exists, nil
}

func (redisRepo *FilmRedisRepo) GetNearFilms(ctx context.Context, uid string, lg *slog.Logger) ([]models.NearFilm, error) {
	if !redisRepo.Connection {
		lg.Error("Redis NearFilm connection lost")
		return nil, nil
	}

	result, err := redisRepo.filmRedisClient.HGetAll(ctx, "nearfilms:"+uid).Result()
	if err != nil {
		lg.Error("HGetAll request could not be completed:", err)
		return nil, err
	}

	var nearFilms []models.NearFilm
	for idFilmStr := range result {
		idFilm, err := strconv.ParseUint(idFilmStr, 10, 64)
		if err != nil {
			lg.Error("Error parsing IdFilm:", err)
			continue
		}

		idUser, err := strconv.ParseUint(uid, 10, 64)
		if err != nil {
			lg.Error("Error parsing IdUser:", err)
			continue
		}

		nearFilm := models.NearFilm{
			IdUser: idUser,
			IdFilm: idFilm,
		}
		nearFilms = append(nearFilms, nearFilm)
	}

	return nearFilms, nil
}


func (redisRepo *FilmRedisRepo) DeleteNearFilm(ctx context.Context, uid string, fid string, lg *slog.Logger) (bool, error) {
	deletedCount, err := redisRepo.filmRedisClient.HDel(ctx, "nearfilms:"+uid, fid).Result()
	if err != nil {
		lg.Error("HDEL request could not be completed:", err)
		return false, err
	}

	if deletedCount == 0 {
		lg.Info("Field " + fid + " does not exist in hash " + uid)
	}

	return true, nil
}
