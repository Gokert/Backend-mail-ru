package usecase

import (
	"context"
	"fmt"
	"log/slog"

	auth "github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/proto"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/comments/repository/comment"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//go:generate mockgen -source=core.go -destination=../mocks/core_mock.go -package=mocks

type ICore interface {
	GetFilmComments(filmId uint64, first uint64, limit uint64) ([]models.CommentItem, error)
	AddComment(filmId uint64, userId uint64, rating uint16, text string) (bool, error)
	GetUserId(ctx context.Context, sid string) (uint64, error)
	DeleteComment(idUser uint64, idFilm uint64) error
}

type Core struct {
	lg       *slog.Logger
	comments comment.ICommentRepo
	client   auth.AuthorizationClient
}

func GetClient(port string) (auth.AuthorizationClient, error) {
	conn, err := grpc.Dial(port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc connect err: %w", err)
	}
	client := auth.NewAuthorizationClient(conn)

	return client, nil
}

func GetCore(cfg_sql *configs.CommentCfg, lg *slog.Logger, comments comment.ICommentRepo) *Core {
	client, err := GetClient(cfg_sql.GrpcPort)
	if err != nil {
		lg.Error("get client error", "err", err.Error())
		return nil
	}
	core := Core{
		lg:       lg.With("module", "core"),
		comments: comments,
		client:   client,
	}
	return &core
}

func (core *Core) GetFilmComments(filmId uint64, first uint64, limit uint64) ([]models.CommentItem, error) {
	comments, err := core.comments.GetFilmComments(filmId, first, limit)
	if err != nil {
		core.lg.Error("Get Film Comments error", "err", err.Error())
		return nil, fmt.Errorf("GetFilmComments err: %w", err)
	}
	ids := make([]int32, len(comments))
	for i := 0; i < len(ids); i++ {
		ids[i] = int32(comments[i].IdUser)
	}

	namesAndPhotos, err := core.client.GetIdsAndPaths(context.Background(), &auth.NamesAndPathsListRequest{Ids: ids})
	if err != nil {
		core.lg.Error("get film comments grpc error", "err", err.Error())
		return nil, fmt.Errorf("get film comments grpc err: %w", err)
	}
	for i := 0; i < len(namesAndPhotos.Names); i++ {
		comments[i].Username = namesAndPhotos.Names[i]
		comments[i].Photo = namesAndPhotos.Paths[i]
	}
	return comments, nil
}

func (core *Core) AddComment(filmId uint64, userId uint64, rating uint16, text string) (bool, error) {
	found, err := core.comments.HasUsersComment(userId, filmId)
	if err != nil {
		core.lg.Error("find users comment error", "err", err.Error())
		return false, fmt.Errorf("find users comment error: %w", err)
	}
	if found {
		return found, nil
	}

	err = core.comments.AddComment(filmId, userId, rating, text)
	if err != nil {
		core.lg.Error("add Comment error", "err", err.Error())
		return false, fmt.Errorf("add comment err: %w", err)
	}

	return false, nil
}

func (core *Core) GetUserId(ctx context.Context, sid string) (uint64, error) {
	request := auth.FindIdRequest{Sid: sid}

	response, err := core.client.GetId(ctx, &request)
	if err != nil {
		core.lg.Error("get user id error", "err", err.Error())
		return 0, fmt.Errorf("get user id err: %w", err)
	}
	return uint64(response.Value), nil
}

func (core *Core) DeleteComment(idUser uint64, idFilm uint64) error {
	err := core.comments.DeleteComment(idUser, idFilm)
	if err != nil {
		core.lg.Error("delete comment error", "err", err.Error())
		return fmt.Errorf("delete comment err: %w", err)
	}

	return nil
}
