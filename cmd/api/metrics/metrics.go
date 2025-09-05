package metrics

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	TicketsCreatedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tickets_created_total",
		Help: "Number of tickets created.",
	})
	TicketsUpdatedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tickets_updated_total",
		Help: "Number of tickets updated.",
	})
	AuthFailuresTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "auth_failures_total",
		Help: "Number of authentication failures.",
	})
	AttachmentsUploadedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "attachments_uploaded_total",
		Help: "Number of attachments uploads or finalizations.",
	})
	RateLimitRejectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_rejections_total",
			Help: "Number of requests rejected by rate limiting.",
		},
		[]string{"route"},
	)
	registerOnce sync.Once
)

func RegisterCounters() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			TicketsCreatedTotal,
			TicketsUpdatedTotal,
			AuthFailuresTotal,
			AttachmentsUploadedTotal,
			RateLimitRejectionsTotal,
		)
	})
}

func SLA(a *app.App) gin.HandlerFunc { return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) } }
func Resolution(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) }
}
func TicketVolume(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) }
}

// Agent returns per-agent quick metrics snapshot
func Agent(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) }
}

// Manager returns queue/manager analytics snapshot
func Manager(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) }
}
