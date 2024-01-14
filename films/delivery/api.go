package delivery

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/usecase"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/middleware"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
	"github.com/mailru/easyjson"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	core   usecase.ICore
	lg     *slog.Logger
	mx     *http.ServeMux
	ct     *requests.Collector
	adress string
}

func GetApi(c *usecase.Core, l *slog.Logger, cfg *configs.DbDsnCfg) *API {
	api := &API{
		core:   c,
		lg:     l.With("module", "api"),
		mx:     http.NewServeMux(),
		ct:     requests.GetCollector(),
		adress: cfg.ServerAdress,
	}

	api.mx.Handle("/metrics", promhttp.Handler())
	api.mx.HandleFunc("/api/v1/films", api.Films)
	api.mx.Handle("/api/v1/film", middleware.AuthCheck(http.HandlerFunc(api.Film), c, l))
	api.mx.HandleFunc("/api/v1/actor", api.Actor)
	api.mx.Handle("/api/v1/favorite/films", middleware.AuthCheck(http.HandlerFunc(api.FavoriteFilms), c, l))
	api.mx.Handle("/api/v1/favorite/film/add", middleware.AuthCheck(http.HandlerFunc(api.FavoriteFilmsAdd), c, l))
	api.mx.Handle("/api/v1/favorite/film/remove", middleware.AuthCheck(http.HandlerFunc(api.FavoriteFilmsRemove), c, l))
	api.mx.Handle("/api/v1/favorite/actors", middleware.AuthCheck(http.HandlerFunc(api.FavoriteActors), c, l))
	api.mx.Handle("/api/v1/favorite/actor/add", middleware.AuthCheck(http.HandlerFunc(api.FavoriteActorsAdd), c, l))
	api.mx.Handle("/api/v1/favorite/actor/remove", middleware.AuthCheck(http.HandlerFunc(api.FavoriteActorsRemove), c, l))
	api.mx.HandleFunc("/api/v1/find", api.FindFilm)
	api.mx.HandleFunc("/api/v1/search/actor", api.FindActor)
	api.mx.HandleFunc("/api/v1/calendar", api.Calendar)
	api.mx.Handle("/api/v1/rating/add", middleware.AuthCheck(http.HandlerFunc(api.AddRating), c, l))
	api.mx.HandleFunc("/api/v1/add/film", api.AddFilm)
	api.mx.Handle("/api/v1/rating/delete", middleware.AuthCheck(http.HandlerFunc(api.DeleteRating), c, l))
	api.mx.Handle("/api/v1/statistics", middleware.AuthCheck(http.HandlerFunc(api.UsersStatistics), c, l))
	api.mx.HandleFunc("/api/v1/trends", api.Trends)
	api.mx.Handle("/api/v1/lasts", middleware.AuthCheck(http.HandlerFunc(api.LastSeen), c, l))

	return api
}

func (a *API) ListenAndServe() {
	err := http.ListenAndServe(a.adress, a.mx)
	if err != nil {
		a.lg.Error("listen and serve error", "err", err.Error())
	}
}

func (a *API) Films(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()

	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	page, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(r.URL.Query().Get("page_size"), 10, 64)
	if err != nil {
		pageSize = 8
	}

	genreId, err := strconv.ParseUint(r.URL.Query().Get("collection_id"), 10, 64)
	if err != nil {
		genreId = 0
	}

	var films []models.FilmItem

	films, genre, err := a.core.GetFilmsAndGenreTitle(genreId, uint64((page-1)*pageSize), pageSize)
	if err != nil {
		a.lg.Error("get films error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	filmsResponse := requests.FilmsResponse{
		Page:           page,
		PageSize:       pageSize,
		Total:          uint64(len(films)),
		CollectionName: genre,
		Films:          films,
	}
	response.Body = filmsResponse

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) Film(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	filmId, err := strconv.ParseUint(r.URL.Query().Get("film_id"), 10, 64)
	if err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	film, err := a.core.GetFilmInfo(filmId)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}
		a.lg.Error("film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	response.Body = film

	a.ct.SendResponse(w, r, response, a.lg, start)

	userId, isAuth := r.Context().Value(middleware.UserIDKey).(uint64)

	if !isAuth {
		a.lg.Error("User StatusUnauthorized", "err", !isAuth)
		return
	}

	nearFilm := models.NearFilm{
		IdFilm: filmId,
		IdUser: userId,
	}

	addedNearFilm, err := a.core.AddNearFilm(r.Context(), nearFilm, a.lg)
	if err != nil {
		a.lg.Error("Failed to add near film", "error", err.Error())
		return
	}

	if !addedNearFilm {
		a.lg.Error("Failed to add near film", "error", err.Error())
		return
	}
}

func (a *API) Actor(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	actorId, err := strconv.ParseUint(r.URL.Query().Get("actor_id"), 10, 64)
	if err != nil {
		a.lg.Error("actor error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	actor, err := a.core.GetActorInfo(actorId)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}
		a.lg.Error("actor error", "err", err.Error())
		response.Status = http.StatusInternalServerError

		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	response.Body = actor

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) FindFilm(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	var request requests.FindFilmRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.lg.Error("find film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	if err = easyjson.Unmarshal(body, &request); err != nil {
		a.lg.Error("find film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	films, err := a.core.FindFilm(request.Title, request.DateFrom, request.DateTo, request.RatingFrom, request.RatingTo,
		request.Mpaa, request.Genres, request.Actors, (request.Page-1)*request.PerPage, request.PerPage)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}

		a.lg.Error("find film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	filmsResponse := requests.FilmsResponse{
		Films: films,
		Total: uint64(len((films))),
	}
	response.Body = filmsResponse

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) FavoriteFilmsAdd(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	filmId, err := strconv.ParseUint(r.URL.Query().Get("film_id"), 10, 64)
	if err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err = a.core.FavoriteFilmsAdd(userId, filmId)
	if err != nil {
		if errors.Is(err, usecase.ErrFoundFavorite) {
			response.Status = http.StatusNotAcceptable
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}

		a.lg.Error("favorite films error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) FavoriteFilmsRemove(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	filmId, err := strconv.ParseUint(r.URL.Query().Get("film_id"), 10, 64)
	if err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err = a.core.FavoriteFilmsRemove(userId, filmId)
	if err != nil {
		a.lg.Error("favorite films error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) FavoriteFilms(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	page, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}

	pageSize, err := strconv.ParseUint(r.URL.Query().Get("per_page"), 10, 64)
	if err != nil {
		pageSize = 8
	}

	films, err := a.core.FavoriteFilms(userId, uint64((page-1)*pageSize), pageSize)
	if err != nil {
		a.lg.Error("favorite films error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	response.Body = films

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) Calendar(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	calendar, err := a.core.GetCalendar()
	if err != nil {
		a.lg.Error("calendar error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	response.Body = calendar

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) FindActor(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	var request requests.FindActorRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.lg.Error("find actor error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	if err = easyjson.Unmarshal(body, &request); err != nil {
		a.lg.Error("find actor error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	actors, err := a.core.FindActor(request.Name, request.BirthDate, request.Films, request.Career, request.Country, (request.Page-1)*request.PerPage, request.PerPage)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}

		a.lg.Error("find actor error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	actorsResponse := requests.ActorsResponse{
		Actors: actors,
		Total:  uint64(len(actors)),
	}
	response.Body = actorsResponse

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) AddRating(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	var commentRequest requests.CommentRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	if err = easyjson.Unmarshal(body, &commentRequest); err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	found, err := a.core.AddRating(commentRequest.FilmId, userId, commentRequest.Rating)
	if err != nil {
		a.lg.Error("add rating error", "err", err.Error())
		response.Status = http.StatusInternalServerError
	}
	if found {
		response.Status = http.StatusNotAcceptable
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) AddFilm(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	title := r.FormValue("title")
	info := r.FormValue("info")
	date := r.FormValue("date")
	country := r.FormValue("country")

	genresString := r.FormValue("genre")
	var genres []uint64
	prev := 0
	for i := 0; i < len(genresString); i++ {
		if genresString[i] == ',' {
			genreUint, err := strconv.ParseUint(genresString[prev:i], 10, 64)
			if err != nil {
				a.lg.Error("add film error", "err", err.Error())
				response.Status = http.StatusBadRequest
				a.ct.SendResponse(w, r, response, a.lg, start)
				return
			}
			genres = append(genres, genreUint)
			prev = i + 1
		}
	}
	genreUint, err := strconv.ParseUint(genresString[prev:], 10, 64)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	genres = append(genres, genreUint)
	prev = 0

	actorsString := r.FormValue("actors")
	var actors []uint64
	for i := 0; i < len(actorsString); i++ {
		if actorsString[i] == ',' {
			actorUint, err := strconv.ParseUint(actorsString[prev:i], 10, 64)
			if err != nil {
				a.lg.Error("add film error", "err", err.Error())
				response.Status = http.StatusBadRequest
				a.ct.SendResponse(w, r, response, a.lg, start)
				return
			}
			actors = append(actors, actorUint)
			prev = i + 1
		}
	}
	actorUint, err := strconv.ParseUint(actorsString[prev:], 10, 64)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	actors = append(actors, actorUint)

	fmt.Println(actors, genres)
	poster, handler, err := r.FormFile("photo")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	filename := "/icons/" + handler.Filename
	if err != nil && handler != nil && poster != nil {
		a.lg.Error("Post profile error", "err", err.Error())
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	filePhoto, err := os.OpenFile("/home/ubuntu/frontend-project"+filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	defer filePhoto.Close()

	_, err = io.Copy(filePhoto, poster)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	film := models.FilmItem{
		Title:       title,
		Info:        info,
		Poster:      filename,
		ReleaseDate: date,
		Country:     country,
	}

	err = a.core.AddFilm(film, genres, actors)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) FavoriteActorsAdd(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	actorId, err := strconv.ParseUint(r.URL.Query().Get("actor_id"), 10, 64)
	if err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err = a.core.FavoriteActorsAdd(userId, actorId)
	if err != nil {
		if errors.Is(err, usecase.ErrFoundFavorite) {
			response.Status = http.StatusNotAcceptable
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}
		a.lg.Error("favorite actors error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) FavoriteActorsRemove(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	actorId, err := strconv.ParseUint(r.URL.Query().Get("actor_id"), 10, 64)
	if err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err = a.core.FavoriteActorsRemove(userId, actorId)
	if err != nil {
		a.lg.Error("favorite actors error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) FavoriteActors(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	page, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(r.URL.Query().Get("per_page"), 10, 64)
	if err != nil {
		pageSize = 8
	}

	actors, err := a.core.FavoriteActors(userId, uint64((page-1)*pageSize), pageSize)
	if err != nil {
		a.lg.Error("favorite actors error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	actorsResponse := requests.ActorsResponse{
		Actors: actors,
	}

	response.Body = actorsResponse

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) DeleteRating(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()

	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	var request requests.DeleteCommentRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	if err = easyjson.Unmarshal(body, &request); err != nil {
		response.Status = http.StatusBadRequest
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	err = a.core.DeleteRating(request.IdUser, request.IdFilm)
	if err != nil {
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) UsersStatistics(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()

	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	stats, err := a.core.UsersStatistics(userId)
	if err != nil {
		a.lg.Error("users statistics error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	response.Body = stats
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) Trends(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()

	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	trends, err := a.core.Trends()
	if err != nil {
		a.lg.Error("trends error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	trendsResponse := requests.FilmsResponse{
		Films: trends,
		Total: uint64(len(trends)),
	}

	response.Body = trendsResponse
	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) LastSeen(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	start := time.Now()

	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	filmsIds, err := a.core.GetNearFilms(r.Context(), userId, a.lg)
	if err != nil {
		a.lg.Error("last seen error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	films, err := a.core.GetLastSeen(filmsIds)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			a.ct.SendResponse(w, r, response, a.lg, start)
			return
		}
		a.lg.Error("last seen error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	filmsResponse := requests.FilmsResponse{
		Films: films,
		Total: uint64(len(films)),
	}

	response.Body = filmsResponse
	a.ct.SendResponse(w, r, response, a.lg, start)
}
