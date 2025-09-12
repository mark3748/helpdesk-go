package metrics

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

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

func SLA(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if a.DB == nil {
			c.JSON(http.StatusOK, gin.H{"total": 0, "met": 0, "sla_attainment": 0.0})
			return
		}
		var met, total int
		err := a.DB.QueryRow(ctx, `
               select
                       count(*) filter (where tsc.resolution_elapsed_ms <= sp.resolution_target_mins * 60000) as met,
                       count(*) as total
               from ticket_sla_clocks tsc
               join tickets t on t.id = tsc.ticket_id
               join sla_policies sp on sp.id = tsc.policy_id
               where t.status = 'Resolved'
       `).Scan(&met, &total)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sla query"})
			return
		}
		attainment := 0.0
		if total > 0 {
			attainment = float64(met) / float64(total)
		}
		c.JSON(http.StatusOK, gin.H{"total": total, "met": met, "sla_attainment": attainment})
	}
}

func Resolution(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if a.DB == nil {
			c.JSON(http.StatusOK, gin.H{"avg_resolution_ms": 0})
			return
		}
		var avg sql.NullFloat64
		err := a.DB.QueryRow(ctx, `
               select avg(tsc.resolution_elapsed_ms)
               from ticket_sla_clocks tsc
               join tickets t on t.id = tsc.ticket_id
               where t.status = 'Resolved' and tsc.resolution_elapsed_ms > 0
       `).Scan(&avg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "resolution query"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"avg_resolution_ms": avg.Float64})
	}
}

func TicketVolume(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if a.DB == nil {
			c.JSON(http.StatusOK, gin.H{"daily": []any{}})
			return
		}
		rows, err := a.DB.Query(ctx, `
               select date_trunc('day', created_at)::date as day, count(*)
               from tickets
               group by day
               order by day desc
               limit 30
       `)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "volume query"})
			return
		}
		defer rows.Close()
		type dayCount struct {
			Day   time.Time `json:"day"`
			Count int       `json:"count"`
		}
		out := []dayCount{}
		for rows.Next() {
			var dc dayCount
			if err := rows.Scan(&dc.Day, &dc.Count); err == nil {
				out = append(out, dc)
			}
		}
		c.JSON(http.StatusOK, gin.H{"daily": out})
	}
}

// Dashboard aggregates key ticket metrics for quick dashboard display.
func Dashboard(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if a.DB == nil {
			c.JSON(http.StatusOK, gin.H{
				"sla":               gin.H{"total": 0, "met": 0, "sla_attainment": 0.0},
				"avg_resolution_ms": 0,
				"volume":            []any{},
			})
			return
		}
		var met, total int
		err := a.DB.QueryRow(ctx, `
               select
                       count(*) filter (where tsc.resolution_elapsed_ms <= sp.resolution_target_mins * 60000) as met,
                       count(*) as total
               from ticket_sla_clocks tsc
               join tickets t on t.id = tsc.ticket_id
               join sla_policies sp on sp.id = tsc.policy_id
               where t.status = 'Resolved'
       `).Scan(&met, &total)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sla query"})
			return
		}
		attainment := 0.0
		if total > 0 {
			attainment = float64(met) / float64(total)
		}

		var avg sql.NullFloat64
		err = a.DB.QueryRow(ctx, `
               select avg(tsc.resolution_elapsed_ms)
               from ticket_sla_clocks tsc
               join tickets t on t.id = tsc.ticket_id
               where t.status = 'Resolved' and tsc.resolution_elapsed_ms > 0
       `).Scan(&avg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "resolution query"})
			return
		}

		rows, err := a.DB.Query(ctx, `
               select date_trunc('day', created_at)::date as day, count(*)
               from tickets
               group by day
               order by day desc
               limit 30
       `)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "volume query"})
			return
		}
		defer rows.Close()
		type dayCount struct {
			Day   time.Time `json:"day"`
			Count int       `json:"count"`
		}
		vol := []dayCount{}
		for rows.Next() {
			var dc dayCount
			if err := rows.Scan(&dc.Day, &dc.Count); err == nil {
				vol = append(vol, dc)
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"sla":               gin.H{"total": total, "met": met, "sla_attainment": attainment},
			"avg_resolution_ms": avg.Float64,
			"volume":            vol,
		})
	}
}

// Agent returns per-agent quick metrics snapshot
func Agent(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) }
}

// Manager returns queue/manager analytics snapshot
func Manager(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) }
}
