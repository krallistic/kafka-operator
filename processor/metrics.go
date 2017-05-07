package processor

import "github.com/prometheus/client_golang/prometheus"

var (
	clustersTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "kafka_operator",
		Subsystem: "processor",
		Name:      "clusters",
		Help:      "Total number of clusters managed by the processor",
	})

	clustersCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "kafka_operator",
		Subsystem: "processor",
		Name:      "clusters_created",
		Help:      "Total number of clusters created",
	})

	clustersDeleted = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "kafka_operator",
		Subsystem: "processor",
		Name:      "clusters_deleted",
		Help:      "Total number of clusters deleted",
	})

	clustersModified = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "kafka_operator",
		Subsystem: "processor",
		Name:      "clusters_modified",
		Help:      "Total number of clusters modified",
	})
)

func init() {
	prometheus.MustRegister(clustersTotal)
	prometheus.MustRegister(clustersCreated)
	prometheus.MustRegister(clustersDeleted)
	prometheus.MustRegister(clustersModified)
}
