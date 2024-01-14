package requests

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/metrics"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	easyjson "github.com/mailru/easyjson"
)

//easyjson:json
type (
	Response struct {
		Status int `json:"status"`
		Body   any `json:"body"`
	}

	FilmsResponse struct {
		Page           uint64            `json:"current_page"`
		PageSize       uint64            `json:"page_size"`
		CollectionName string            `json:"collection_name"`
		Total          uint64            `json:"total"`
		Films          []models.FilmItem `json:"films"`
	}

	FilmResponse struct {
		Film       models.FilmItem    `json:"film"`
		Genres     []models.GenreItem `json:"genre"`
		Rating     float64            `json:"rating"`
		Number     uint64             `json:"number"`
		Directors  []models.CrewItem  `json:"directors"`
		Scenarists []models.CrewItem  `json:"scenarists"`
		Characters []models.Character `json:"actors"`
	}

	ActorResponse struct {
		Name      string                  `json:"name"`
		Photo     string                  `json:"poster_href"`
		Career    []models.ProfessionItem `json:"career"`
		BirthDate string                  `json:"birthday"`
		Country   string                  `json:"country"`
		Info      string                  `json:"info_text"`
	}

	ActorsResponse struct {
		Actors []models.Character `json:"actors"`
		Total  uint64             `json:"total"`
	}

	CommentResponse struct {
		Comments []models.CommentItem `json:"comment"`
	}

	ProfileResponse struct {
		Email     string `json:"email"`
		Name      string `json:"name"`
		Login     string `json:"login"`
		Photo     string `json:"photo"`
		BirthDate string `json:"birthday"`
	}

	AuthCheckResponse struct {
		Login string `json:"login"`
		Role  string `json:"role"`
	}

	CalendarResponse struct {
		MonthName  string           `json:"monthName"`
		MonthText  string           `json:"monthText"`
		CurrentDay uint8            `json:"currentDay"`
		Days       []models.DayItem `json:"days"`
	}

	SubcribeResponse struct {
		IsSubcribed bool `json:"subscribe"`
	}

	UsersResponse struct {
		Users []models.UserItem `json:"users"`
	}

	UsersStatisticsResponse struct {
		GenreId uint64  `json:"genre_id"`
		Count   uint64  `json:"count"`
		Avg     float64 `json:"avg"`
	}
)

type Collector struct {
	mt *metrics.Metrics
}

func GetCollector() *Collector {
	return &Collector{
		mt: metrics.GetMetrics(),
	}
}

func sendMetrics(mt *metrics.Metrics, path string, status int, start time.Time) {
	end := time.Since(start)
	mt.Time.WithLabelValues(strconv.Itoa(status), path).Observe(end.Seconds())
	mt.Hits.WithLabelValues(strconv.Itoa(status), path).Inc()
}

func (c *Collector) SendResponse(w http.ResponseWriter, r *http.Request, response Response, lg *slog.Logger, start time.Time) {
	jsonResponse, err := easyjson.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		sendMetrics(c.mt, r.URL.Path, http.StatusInternalServerError, start)
		lg.Error("failed to pack json", "err", err.Error())
		return
	}
	sendMetrics(c.mt, r.URL.Path, response.Status, start)

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonResponse)
	if err != nil {
		lg.Error("failed to send response", "err", err.Error())
		return
	}
}
