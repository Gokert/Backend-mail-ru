package main

import (
	"log/slog"
	"os"

	delivery_auth "github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/delivery/http"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"

	delivery_auth_grpc "github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/delivery/grpc"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/usecase"
)

func main() {
	logFile, _ := os.Create("auth_log.log")
	lg := slog.New(slog.NewJSONHandler(logFile, nil))

	config, err := configs.ReadConfig()
	if err != nil {
		lg.Error("read config error", "err", err.Error())
		return
	}

	configCsrf, err := configs.ReadCsrfRedisConfig()
	if err != nil {
		lg.Error("read config error", "err", err.Error())
		return
	}

	configSession, err := configs.ReadSessionRedisConfig()
	if err != nil {
		lg.Error("read config error", "err", err.Error())
		return
	}

	core, err := usecase.GetCore(config, *configCsrf, *configSession, lg)
	if err != nil {
		lg.Error("cant create core")
		return
	}

	api := delivery_auth.GetApi(core, lg)

	errs := make(chan error, 2)

	grpcServ, err := delivery_auth_grpc.NewServer(lg)
	if err != nil {
		lg.Error("cant create server")
		return
	}

	go func() {
		errs <- api.ListenAndServe()
	}()
	go func() {
		errs <- grpcServ.ListenAndServeGrpc()
	}()

	err = <-errs
	if err != nil {
		lg.Error("listen and serve error", "err", err.Error())
	}
}
