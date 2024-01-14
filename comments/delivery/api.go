package delivery

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/comments/usecase"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/middleware"
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

func GetApi(c *usecase.Core, l *slog.Logger, cfg *configs.CommentCfg) *API {

	api := &API{
		core:   c,
		lg:     l.With("module", "api"),
		mx:     http.NewServeMux(),
		ct:     requests.GetCollector(),
		adress: cfg.ServerAdress,
	}

	api.mx.Handle("/metrics", promhttp.Handler())
	api.mx.HandleFunc("/api/v1/comment", api.Comment)
	api.mx.Handle("/api/v1/comment/add", middleware.AuthCheck(http.HandlerFunc(api.AddComment), c, l))
	api.mx.Handle("/api/v1/comment/delete", middleware.AuthCheck(http.HandlerFunc(api.DeleteComment), c, l))

	return api
}

func (a *API) ListenAndServe() {
	err := http.ListenAndServe(a.adress, a.mx)
	if err != nil {
		a.lg.Error("listen and serve error", "err", err.Error())
	}
}

func (a *API) Comment(w http.ResponseWriter, r *http.Request) {
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
	page, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(r.URL.Query().Get("per_page"), 10, 64)
	if err != nil {
		pageSize = 10
	}

	comments, err := a.core.GetFilmComments(filmId, (page-1)*pageSize, pageSize)
	if err != nil {
		a.lg.Error("Comment", "err", err.Error())
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	commentsResponse := requests.CommentResponse{Comments: comments}

	response.Body = commentsResponse

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) AddComment(w http.ResponseWriter, r *http.Request) {
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

	found, err := a.core.AddComment(commentRequest.FilmId, userId, commentRequest.Rating, commentRequest.Text)
	if err != nil {
		a.lg.Error("Add Comment error", "err", err.Error())
		response.Status = http.StatusInternalServerError
	}
	if found {
		response.Status = http.StatusNotAcceptable
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}

	a.ct.SendResponse(w, r, response, a.lg, start)
}

func (a *API) DeleteComment(w http.ResponseWriter, r *http.Request) {
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

	err = a.core.DeleteComment(request.IdUser, request.IdFilm)
	if err != nil {
		response.Status = http.StatusInternalServerError
		a.ct.SendResponse(w, r, response, a.lg, start)
		return
	}
	a.ct.SendResponse(w, r, response, a.lg, start)
}
