package tickets

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

type Ticket struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	RequesterID string      `json:"requester_id"`
	Priority    int16       `json:"priority"`
	CustomJSON  interface{} `json:"custom_json"`
}

// createTicketReq mirrors the JSON body for creating a ticket.
type createTicketReq struct {
	Title       string          `json:"title" binding:"required,min=3"`
	Description string          `json:"description"`
	RequesterID string          `json:"requester_id" binding:"required"`
	Priority    int16           `json:"priority" binding:"required,min=1,max=4"`
	Urgency     *int16          `json:"urgency" binding:"omitempty,min=1,max=4"`
	Category    *string         `json:"category"`
	Subcategory *string         `json:"subcategory"`
	Status      string          `json:"status"`
	ScheduledAt *string         `json:"scheduled_at"`
	DueAt       *string         `json:"due_at"`
	Source      string          `json:"source"`
	CustomJSON  json.RawMessage `json:"custom_json"`
}

// Create handles POST /tickets with minimal validation logic used in tests.
func Create(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in createTicketReq
		if err := c.ShouldBindJSON(&in); err != nil {
			errs := map[string]string{}
			if ve, ok := err.(validator.ValidationErrors); ok {
				for _, fe := range ve {
					errs[strings.ToLower(fe.Field())] = fe.Tag()
				}
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": errs})
			return
		}
		if len(in.CustomJSON) > 0 {
			var tmp interface{}
			if err := json.Unmarshal(in.CustomJSON, &tmp); err != nil || reflect.ValueOf(tmp).Kind() != reflect.Map {
				c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"custom_json": "must be object"}})
				return
			}
		}
		c.JSON(http.StatusCreated, Ticket{Title: in.Title, RequesterID: in.RequesterID, Priority: in.Priority})
	}
}

// The remaining ticket-related handlers are placeholders.
func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, []Ticket{}) }
}
func Get(a *app.App) gin.HandlerFunc { return func(c *gin.Context) { c.JSON(http.StatusOK, Ticket{}) } }
func Update(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, Ticket{}) }
}
