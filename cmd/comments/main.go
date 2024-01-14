package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/comments/delivery"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/comments/repository/comment"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/comments/usecase"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
)

func main() {
	var path string
	flag.StringVar(&path, "comments_log_path", "comment_log.log", "Путь к логу комментов")
	logFile, _ := os.Create(path)
	lg := slog.New(slog.NewJSONHandler(logFile, nil))

	config, err := configs.ReadCommentConfig()
	if err != nil {
		lg.Error("read config error", "err", err.Error())
		return
	}

	var comments comment.ICommentRepo
	switch config.CommentsDb {
	case "postgres":
		comments, err = comment.GetCommentRepo(config, lg)
	}
	if err != nil {
		lg.Error("cant create repo")
		return
	}

	core := usecase.GetCore(config, lg, comments)
	api := delivery.GetApi(core, lg, config)

	api.ListenAndServe()
}
