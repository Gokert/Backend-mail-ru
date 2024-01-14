package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	Time *prometheus.HistogramVec
	Hits *prometheus.CounterVec
}

func GetMetrics() *Metrics {
	description := []string{"status", "path"}

	metrics := &Metrics{
		Time: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "Time_Req",
			Help:    "Request work time.",
			Buckets: prometheus.LinearBuckets(0, 100, 6),
		}, description),
 
		Hits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "Hits_Req",
			Help: "Step",
		}, description),
	}

	prometheus.MustRegister(metrics.Time, metrics.Hits)

	return metrics
}
