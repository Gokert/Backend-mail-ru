package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	auth "github.com/go-park-mail-ru/2023_2_Vkladyshi/authorization/proto"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/calendar"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/crew"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/film"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/genre"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/profession"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrFoundFavorite = errors.New("found favorite")
)

//go:generate mockgen -source=core.go -destination=../mocks/core_mock.go -package=mocks

type ICore interface {
	GetFilmsAndGenreTitle(genreId uint64, start uint64, end uint64) ([]models.FilmItem, string, error)
	GetFilmInfo(filmId uint64) (*requests.FilmResponse, error)
	GetActorInfo(actorId uint64) (*requests.ActorResponse, error)
	GetActorsCareer(actorId uint64) ([]models.ProfessionItem, error)
	GetGenre(genreId uint64) (string, error)
	FindFilm(title string, dateFrom string, dateTo string, ratingFrom float32, ratingTo float32,
		mpaa string, genres []uint32, actors []string, first uint64, limit uint64,
	) ([]models.FilmItem, error)
	FavoriteFilms(userId uint64, start uint64, end uint64) ([]models.FilmItem, error)
	FavoriteFilmsAdd(userId uint64, filmId uint64) error
	FavoriteFilmsRemove(userId uint64, filmId uint64) error
	GetCalendar() (*requests.CalendarResponse, error)
	GetUserId(ctx context.Context, sid string) (uint64, error)
	FindActor(name string, birthDate string, films []string, career []string, country string, first, limit uint64) ([]models.Character, error)
	AddRating(filmId uint64, userId uint64, rating uint16) (bool, error)
	AddFilm(film models.FilmItem, genres []uint64, actors []uint64) error
	FavoriteActors(userId uint64, start uint64, end uint64) ([]models.Character, error)
	FavoriteActorsAdd(userId uint64, filmId uint64) error
	FavoriteActorsRemove(userId uint64, filmId uint64) error
	DeleteRating(idUser uint64, idFilm uint64) error
	GetNearFilms(ctx context.Context, userId uint64, lg *slog.Logger) ([]models.NearFilm, error)
	AddNearFilm(ctx context.Context, active models.NearFilm, lg *slog.Logger) (bool, error)
	UsersStatistics(idUser uint64) ([]requests.UsersStatisticsResponse, error)
	Trends() ([]models.FilmItem, error)
	GetLastSeen([]models.NearFilm) ([]models.FilmItem, error)
}

type Core struct {
	lg         *slog.Logger
	films      film.IFilmsRepo
	genres     genre.IGenreRepo
	crew       crew.ICrewRepo
	profession profession.IProfessionRepo
	calendar   calendar.ICalendarRepo
	client     auth.AuthorizationClient
	nearFilms  *film.FilmRedisRepo
}

func GetClient(port string) (auth.AuthorizationClient, error) {
	conn, err := grpc.Dial(port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc connect err: %w", err)
	}
	client := auth.NewAuthorizationClient(conn)

	return client, nil
}

func GetCore(cfg_sql *configs.DbDsnCfg, lg *slog.Logger,
	films film.IFilmsRepo, genres genre.IGenreRepo, actors crew.ICrewRepo, professions profession.IProfessionRepo, calendar calendar.ICalendarRepo,
	nearFilms *film.FilmRedisRepo) *Core {
	client, err := GetClient(cfg_sql.GrpcPort)
	if err != nil {
		lg.Error("get client error", "err", err.Error())
		return nil
	}

	core := Core{
		lg:         lg.With("module", "core"),
		films:      films,
		genres:     genres,
		crew:       actors,
		profession: professions,
		calendar:   calendar,
		client:     client,
		nearFilms:  nearFilms,
	}
	return &core
}

func (core *Core) GetFilmsAndGenreTitle(genreId uint64, start uint64, end uint64) ([]models.FilmItem, string, error) {
	var films []models.FilmItem
	var err error

	if genreId == 0 {
		films, err = core.films.GetFilms(start, end)
	} else {
		films, err = core.films.GetFilmsByGenre(genreId, start, end)
	}
	if err != nil {
		core.lg.Error("failed to get films from db", "err", err.Error())
		return nil, "", fmt.Errorf("GetFilms err: %w", err)
	}

	genre, err := core.genres.GetGenreById(genreId)
	if err != nil {
		core.lg.Error("failed to get genre by id", "err", err.Error())
		return nil, "", fmt.Errorf("GetFilms err: %w", err)
	}

	return films, genre, nil
}

func (core *Core) GetFilmInfo(filmId uint64) (*requests.FilmResponse, error) {
	film, err := core.films.GetFilm(filmId)
	if err != nil {
		core.lg.Error("get film error", "err", err.Error())
		return nil, fmt.Errorf("get film err: %w", err)
	}
	if film.Title == "" {
		return nil, ErrNotFound
	}

	genres, err := core.genres.GetFilmGenres(filmId)
	if err != nil {
		core.lg.Error("get film genres error", "err", err.Error())
		return nil, fmt.Errorf("get film genres err: %w", err)
	}

	rating, number, err := core.films.GetFilmRating(filmId)
	if err != nil {
		core.lg.Error("get film rating error", "err", err.Error())
		return nil, fmt.Errorf("get film rating err: %w", err)
	}

	directors, err := core.crew.GetFilmDirectors(filmId)
	if err != nil {
		core.lg.Error("get film directors error", "err", err.Error())
		return nil, fmt.Errorf("get film directors err: %w", err)
	}

	scenarists, err := core.crew.GetFilmScenarists(filmId)
	if err != nil {
		core.lg.Error("get film scenarists error", "err", err.Error())
		return nil, fmt.Errorf("get film scenarists err: %w", err)
	}

	characters, err := core.crew.GetFilmCharacters(filmId)
	if err != nil {
		core.lg.Error("get film characters error", "err", err.Error())
		return nil, fmt.Errorf("get film scenarists err: %w", err)
	}

	result := requests.FilmResponse{
		Film:       *film,
		Genres:     genres,
		Rating:     rating,
		Number:     number,
		Directors:  directors,
		Scenarists: scenarists,
		Characters: characters,
	}

	return &result, nil
}

func (core *Core) GetActorInfo(actorId uint64) (*requests.ActorResponse, error) {
	actor, err := core.crew.GetActor(actorId)
	if err != nil {
		core.lg.Error("get actor error", "err", err.Error())
		return nil, fmt.Errorf("get actor err: %w", err)
	}
	if actor.Name == "" {
		return nil, ErrNotFound
	}

	career, err := core.profession.GetActorsProfessions(actorId)
	if err != nil {
		core.lg.Error("get actor profession error", "err", err.Error())
		return nil, fmt.Errorf("get actor profession err: %w", err)
	}

	result := requests.ActorResponse{
		Name:      actor.Name,
		Photo:     actor.Photo,
		BirthDate: actor.Birthdate,
		Country:   actor.Country,
		Info:      actor.Info,
		Career:    career,
	}
	return &result, nil
}

func (core *Core) GetActorsCareer(actorId uint64) ([]models.ProfessionItem, error) {
	career, err := core.profession.GetActorsProfessions(actorId)
	if err != nil {
		core.lg.Error("Get Actors Career error", "err", err.Error())
		return nil, fmt.Errorf("GetActorsCareer err: %w", err)
	}

	return career, nil
}

func (core *Core) GetGenre(genreId uint64) (string, error) {
	genre, err := core.genres.GetGenreById(genreId)
	if err != nil {
		core.lg.Error("GetGenre error", "err", err.Error())
		return "", fmt.Errorf("GetGenre err: %w", err)
	}

	return genre, nil
}

func (core *Core) FindFilm(title string, dateFrom string, dateTo string, ratingFrom float32, ratingTo float32,
	mpaa string, genres []uint32, actors []string, first uint64, limit uint64,
) ([]models.FilmItem, error) {

	films, err := core.films.FindFilm(title, dateFrom, dateTo, ratingFrom, ratingTo, mpaa, genres, actors, first, limit)
	if err != nil {
		core.lg.Error("find film error", "err", err.Error())
		return nil, fmt.Errorf("find film err: %w", err)
	}

	if len(films) == 0 {
		return nil, ErrNotFound
	}

	return films, nil
}

func (core *Core) FavoriteFilms(userId uint64, start uint64, end uint64) ([]models.FilmItem, error) {
	films, err := core.films.GetFavoriteFilms(userId, start, end)
	if err != nil {
		core.lg.Error("favorite films error", "err", err.Error())
		return nil, fmt.Errorf("favorite films err: %w", err)
	}

	return films, nil
}

func (core *Core) FavoriteFilmsAdd(userId uint64, filmId uint64) error {
	found, err := core.films.CheckFilm(userId, filmId)
	if err != nil {
		core.lg.Error("favorite film add error", "err", err.Error())
		return fmt.Errorf("favorite film add err: %w", err)
	}
	if found {
		return ErrFoundFavorite
	}

	err = core.films.AddFavoriteFilm(userId, filmId)
	if err != nil {
		core.lg.Error("favorite film add error", "err", err.Error())
		return fmt.Errorf("favorite film add err: %w", err)
	}

	return nil
}

func (core *Core) FavoriteFilmsRemove(userId uint64, filmId uint64) error {
	err := core.films.RemoveFavoriteFilm(userId, filmId)
	if err != nil {
		core.lg.Error("favorite film remove error", "err", err.Error())
		return fmt.Errorf("favorite film remove err: %w", err)
	}

	return nil
}

func (core *Core) GetCalendar() (*requests.CalendarResponse, error) {
	result := &requests.CalendarResponse{}

	news, err := core.calendar.GetCalendar()
	if err != nil {
		core.lg.Error("get calendar error", "err", err.Error())
		return nil, fmt.Errorf("get calendar err: %w", err)
	}

	result.Days = news
	result.CurrentDay = uint8(time.Now().Day())
	result.MonthName = time.Now().Month().String()
	result.MonthText = "Новинки этого месяца"

	return result, nil
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

func (core *Core) FindActor(name string, birthDate string, films []string, career []string, country string, first, limit uint64) ([]models.Character, error) {
	actors, err := core.crew.FindActor(name, birthDate, films, career, country, first, limit)
	if err != nil {
		core.lg.Error("find actor error", "err", err.Error())
		return nil, fmt.Errorf("find actor err: %w", err)
	}
	if len(actors) == 0 {
		return nil, ErrNotFound
	}

	return actors, nil
}

func (core *Core) AddRating(filmId uint64, userId uint64, rating uint16) (bool, error) {
	found, err := core.films.HasUsersRating(userId, filmId)
	if err != nil {
		core.lg.Error("find users rating error", "err", err.Error())
		return false, fmt.Errorf("find users rating error: %w", err)
	}
	if found {
		return found, nil
	}

	err = core.films.AddRating(filmId, userId, rating)
	if err != nil {
		core.lg.Error("add rating error", "err", err.Error())
		return false, fmt.Errorf("add rating err: %w", err)
	}

	return false, nil
}

func (core *Core) AddFilm(film models.FilmItem, genres []uint64, actors []uint64) error {
	err := core.films.AddFilm(film)
	if err != nil {
		core.lg.Error("add film error", "err", err.Error())
		return fmt.Errorf("add film err: %w", err)
	}

	id, err := core.films.GetFilmId(film.Title)
	if err != nil {
		core.lg.Error("get film id", "err", err.Error())
		return fmt.Errorf("add film err: %w", err)
	}

	err = core.genres.AddFilm(genres, id)
	if err != nil {
		core.lg.Error("add films genres error", "err", err.Error())
		return fmt.Errorf("add film err: %w", err)
	}

	err = core.crew.AddFilm(actors, id)
	if err != nil {
		core.lg.Error("add films actors error", "err", err.Error())
		return fmt.Errorf("add film err: %w", err)
	}

	return nil
}

func (core *Core) FavoriteActors(userId uint64, start uint64, end uint64) ([]models.Character, error) {
	actors, err := core.crew.GetFavoriteActors(userId, start, end)
	if err != nil {
		core.lg.Error("favorite actors error", "err", err.Error())
		return nil, fmt.Errorf("favorite actors err: %w", err)
	}

	return actors, nil
}

func (core *Core) FavoriteActorsAdd(userId uint64, actorId uint64) error {
	found, err := core.crew.CheckActor(userId, actorId)
	if err != nil {
		core.lg.Error("favorite actors add error", "err", err.Error())
		return fmt.Errorf("favorite actors add err: %w", err)
	}
	if found {
		return ErrFoundFavorite
	}

	err = core.crew.AddFavoriteActor(userId, actorId)
	if err != nil {
		core.lg.Error("favorite actors add error", "err", err.Error())
		return fmt.Errorf("favorite actors add err: %w", err)
	}

	return nil
}

func (core *Core) FavoriteActorsRemove(userId uint64, actorId uint64) error {
	err := core.crew.RemoveFavoriteActor(userId, actorId)
	if err != nil {
		core.lg.Error("favorite actors remove error", "err", err.Error())
		return fmt.Errorf("favorite actors remove err: %w", err)
	}

	return nil
}

func (core *Core) DeleteRating(idUser uint64, idFilm uint64) error {
	err := core.films.DeleteRating(idUser, idFilm)
	if err != nil {
		core.lg.Error("delete rating error", "err", err.Error())
		return fmt.Errorf("delete rating err: %w", err)
	}

	return nil
}

func (c *Core) GetNearFilms(ctx context.Context, userId uint64, lg *slog.Logger) ([]models.NearFilm, error) {
	nearFilms, err := c.nearFilms.GetNearFilms(ctx, strconv.FormatUint(userId, 10), lg)
	if err != nil {
		lg.Error("Failed to get near films", "error", err.Error())
		return nil, err
	}
	return nearFilms, nil
}

func (c *Core) AddNearFilm(ctx context.Context, active models.NearFilm, lg *slog.Logger) (bool, error) {
	added, err := c.nearFilms.AddNearFilm(ctx, active, lg)
	if err != nil {
		lg.Error("Failed to add near film", "error", err.Error())
		return false, err
	}
	return added, nil
}

func (core *Core) UsersStatistics(idUser uint64) ([]requests.UsersStatisticsResponse, error) {
	stats, err := core.genres.UsersStatistics(idUser)
	if err != nil {
		core.lg.Error("users statistics error", "err", err.Error())
		return nil, fmt.Errorf("users statistics err: %w", err)
	}

	return stats, nil
}

func (core *Core) Trends() ([]models.FilmItem, error) {
	trends, err := core.films.Trends()
	if err != nil {
		core.lg.Error("trends error", "err", err.Error())
		return nil, fmt.Errorf("trends err: %w", err)
	}

	return trends, nil
}

func (core *Core) GetLastSeen(filmsIds []models.NearFilm) ([]models.FilmItem, error) {
	ids := []uint64{}
	for _, id := range filmsIds {
		ids = append(ids, id.IdFilm)
	}

	films, err := core.films.GetLasts(ids)
	if err != nil {
		core.lg.Error("trends error", "err", err.Error())
		return nil, fmt.Errorf("trends err: %w", err)
	}

	if len(films) == 0 {
		return nil, ErrNotFound
	}

	return films, nil
}
