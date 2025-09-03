package main

import (
	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	attachmentspkg "github.com/mark3748/helpdesk-go/cmd/api/attachments"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	commentspkg "github.com/mark3748/helpdesk-go/cmd/api/comments"
	exportspkg "github.com/mark3748/helpdesk-go/cmd/api/exports"
	metricspkg "github.com/mark3748/helpdesk-go/cmd/api/metrics"
	ticketspkg "github.com/mark3748/helpdesk-go/cmd/api/tickets"
	watcherspkg "github.com/mark3748/helpdesk-go/cmd/api/watchers"
)

func main() {
	cfg := apppkg.GetConfig()
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)

	a.R.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	auth := a.R.Group("/")
	auth.Use(authpkg.Middleware(a))
	auth.GET("/me", authpkg.Me)
	auth.GET("/tickets", ticketspkg.List(a))
	auth.POST("/tickets", ticketspkg.Create(a))
	auth.GET("/tickets/:id", ticketspkg.Get(a))
	auth.PATCH("/tickets/:id", authpkg.RequireRole("agent", "manager"), ticketspkg.Update(a))
	auth.GET("/tickets/:id/comments", commentspkg.List(a))
	auth.POST("/tickets/:id/comments", commentspkg.Add(a))
	auth.GET("/tickets/:id/attachments", attachmentspkg.List(a))
	auth.POST("/tickets/:id/attachments", attachmentspkg.Upload(a))
	auth.GET("/tickets/:id/attachments/:attID", attachmentspkg.Get(a))
	auth.DELETE("/tickets/:id/attachments/:attID", attachmentspkg.Delete(a))
	auth.GET("/tickets/:id/watchers", watcherspkg.List(a))
	auth.POST("/tickets/:id/watchers", watcherspkg.Add(a))
	auth.DELETE("/tickets/:id/watchers/:userID", watcherspkg.Remove(a))
	auth.GET("/metrics/sla", authpkg.RequireRole("agent"), metricspkg.SLA(a))
	auth.GET("/metrics/resolution", authpkg.RequireRole("agent"), metricspkg.Resolution(a))
	auth.GET("/metrics/tickets", authpkg.RequireRole("agent"), metricspkg.TicketVolume(a))
	auth.POST("/exports/tickets", authpkg.RequireRole("agent"), exportspkg.Tickets(a))

	a.R.Run(cfg.Addr)
}
