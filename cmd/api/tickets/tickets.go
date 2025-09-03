package tickets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	requesterspkg "github.com/mark3748/helpdesk-go/cmd/api/requesters"
)

type Ticket struct {
	ID          string      `json:"id"`
	Number      any         `json:"number,omitempty"`
	Title       string      `json:"title,omitempty"`
	Status      string      `json:"status,omitempty"`
	AssigneeID  *string     `json:"assignee_id,omitempty"`
	Priority    int16       `json:"priority,omitempty"`
	CustomJSON  interface{} `json:"custom_json,omitempty"`
	RequesterID string      `json:"requester_id,omitempty"`
	Requester   string      `json:"requester,omitempty"`
}

// createTicketReq mirrors the JSON body for creating a ticket.
type createTicketReq struct {
	Title       string `json:"title" binding:"required,min=3"`
	Description string `json:"description"`
	RequesterID string `json:"requester_id"`
	Requester   *struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Phone string `json:"phone"`
	} `json:"requester"`
	Priority    int16           `json:"priority" binding:"required,min=1,max=4"`
	AssigneeID  *string         `json:"assignee_id"`
	Urgency     *int16          `json:"urgency" binding:"omitempty,min=1,max=4"`
	Category    *string         `json:"category"`
	Subcategory *string         `json:"subcategory"`
	Status      string          `json:"status"`
	ScheduledAt *string         `json:"scheduled_at"`
	DueAt       *string         `json:"due_at"`
	Source      string          `json:"source"`
	CustomJSON  json.RawMessage `json:"custom_json"`
}

// Create inserts a new ticket and returns a summary.
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
			if err := json.Unmarshal(in.CustomJSON, &tmp); err != nil || (tmp != nil && reflect.ValueOf(tmp).Kind() != reflect.Map) {
				c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"custom_json": "must be object"}})
				return
			}
		}
		if in.RequesterID == "" {
			if in.Requester == nil {
				c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"requester": "required"}})
				return
			}
			if in.Requester.Email == "" && in.Requester.Phone == "" {
				c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"requester": "email_or_phone"}})
				return
			}
			if in.Requester.Email != "" && !requesterspkg.ValidEmail(in.Requester.Email) {
				c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"email": "invalid"}})
				return
			}
			if in.Requester.Phone != "" && !requesterspkg.ValidPhone(in.Requester.Phone) {
				c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"phone": "invalid"}})
				return
			}
			if a.DB == nil {
				in.RequesterID = "1"
			} else {
				const rq = `insert into requesters (email, name, phone) values (nullif($1,''), nullif($2,''), nullif($3,'')) returning id::text`
				if err := a.DB.QueryRow(c.Request.Context(), rq, strings.ToLower(in.Requester.Email), in.Requester.Name, in.Requester.Phone).Scan(&in.RequesterID); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
		if in.RequesterID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"requester_id": "required"}})
			return
		}
		// Test mode: no DB attached, mimic previous behavior
		if a.DB == nil {
			c.JSON(http.StatusCreated, Ticket{Title: in.Title, Priority: in.Priority})
			return
		}
		if in.Source == "" {
			in.Source = "web"
		}
		// Determine default assignee: if current user has the agent role, assign to them
		var defaultAssignee string
		if v, ok := c.Get("user"); ok {
			if u, ok := v.(authpkg.AuthUser); ok {
				for _, r := range u.Roles {
					if r == "agent" {
						defaultAssignee = u.ID
						break
					}
				}
			}
		}
		if in.AssigneeID != nil && *in.AssigneeID != "" {
			defaultAssignee = *in.AssigneeID
		}
		// Fallback: if still empty, check DB roles for this user id
		if defaultAssignee == "" && a.DB != nil {
			if v, ok := c.Get("user"); ok {
				if u, ok := v.(authpkg.AuthUser); ok && u.ID != "" {
					var hasAgent bool
					_ = a.DB.QueryRow(c.Request.Context(), `select exists(select 1 from user_roles ur join roles r on r.id=ur.role_id where ur.user_id=$1 and r.name='agent')`, u.ID).Scan(&hasAgent)
					if hasAgent {
						defaultAssignee = u.ID
					}
				}
			}
		}

		// Insert ticket
		const q = `with s as (select nextval('ticket_seq') n)
insert into tickets (number, title, description, requester_id, priority, status, source, custom_json)
values ((select 'HD-'||n from s), $1, $2, $3, $4, coalesce(nullif($5,''),'New'), $6, coalesce(nullif($7,''),'{}')::jsonb)
returning id::text, number, title, status, assignee_id::text, priority`
		const qAssign = `with s as (select nextval('ticket_seq') n)
insert into tickets (number, title, description, requester_id, assignee_id, priority, status, source, custom_json)
values ((select 'HD-'||n from s), $1, $2, $3, $4, $5, coalesce(nullif($6,''),'New'), $7, coalesce(nullif($8,''),'{}')::jsonb)
returning id::text, number, title, status, assignee_id::text, priority`
		var t Ticket
		var assignee *string
		var number any
		var status string
		var row = a.DB.QueryRow(c.Request.Context(), q, in.Title, in.Description, in.RequesterID, in.Priority, in.Status, in.Source, string(in.CustomJSON))
		if defaultAssignee != "" {
			row = a.DB.QueryRow(c.Request.Context(), qAssign, in.Title, in.Description, in.RequesterID, defaultAssignee, in.Priority, in.Status, in.Source, string(in.CustomJSON))
		}
		if err := row.Scan(&t.ID, &number, &t.Title, &status, &assignee, &t.Priority); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		t.Number = number
		t.Status = status
		t.AssigneeID = assignee
		t.RequesterID = in.RequesterID
    // Best-effort fill requester label
    if a.DB != nil {
        var name, email string
        _ = a.DB.QueryRow(c.Request.Context(), `select coalesce(name,''), coalesce(email,'') from requesters where id=$1`, in.RequesterID).Scan(&name, &email)
        if name != "" && email != "" {
            t.Requester = fmt.Sprintf("%s <%s>", name, email)
        } else if name != "" {
            t.Requester = name
        } else {
            t.Requester = email
        }
    }
		c.JSON(http.StatusCreated, t)
	}
}

// List returns recent tickets.
func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, []Ticket{})
			return
		}
		// Basic filters
		where := []string{}
		args := []any{}
		if v := strings.TrimSpace(c.Query("status")); v != "" {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.status = $%d", n))
			args = append(args, v)
		}
		if v := strings.TrimSpace(c.Query("priority")); v != "" {
			if p, err := strconv.Atoi(v); err == nil {
				n := len(args) + 1
				where = append(where, fmt.Sprintf("t.priority = $%d", n))
				args = append(args, p)
			}
		}
		if v := strings.TrimSpace(c.Query("team")); v != "" {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.team_id = $%d", n))
			args = append(args, v)
		}
		assignee := strings.TrimSpace(c.Query("assignee"))
		if assignee == "" {
			assignee = strings.TrimSpace(c.Query("assignee_id"))
		}
		if assignee != "" {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.assignee_id = $%d", n))
			args = append(args, assignee)
		}
		if v := strings.TrimSpace(c.Query("requester")); v != "" {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.requester_id = $%d", n))
			args = append(args, v)
		}
		if v := strings.TrimSpace(c.Query("queue")); v != "" {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.queue_id = $%d", n))
			args = append(args, v)
		}
		if v := strings.TrimSpace(c.Query("search")); v != "" {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("(t.title ILIKE $%d OR t.description ILIKE $%d)", n, n))
			args = append(args, "%"+v+"%")
		}
	sql := "select t.id::text, t.number, t.title, t.status, t.assignee_id::text, t.priority, t.requester_id::text, coalesce(r.name, r.email, '') as requester from tickets t left join requesters r on r.id=t.requester_id"
		if len(where) > 0 {
			sql += " where " + strings.Join(where, " and ")
		}
		sql += " order by t.created_at desc limit 100"
		rows, err := a.DB.Query(c.Request.Context(), sql, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []Ticket{}
		for rows.Next() {
			var t Ticket
			var assignee *string
			var number any
			if err := rows.Scan(&t.ID, &number, &t.Title, &t.Status, &assignee, &t.Priority, &t.RequesterID, &t.Requester); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			t.Number = number
			t.AssigneeID = assignee
			out = append(out, t)
		}
		c.JSON(http.StatusOK, out)
	}
}

// Get returns a ticket by id
func Get(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, Ticket{})
			return
		}
	const q = `select t.id::text, t.number, t.title, t.status, t.assignee_id::text, t.priority, t.requester_id::text, coalesce(r.name, r.email, '') as requester from tickets t left join requesters r on r.id=t.requester_id where t.id=$1`
		var t Ticket
		var assignee *string
		var number any
		row := a.DB.QueryRow(c.Request.Context(), q, c.Param("id"))
		if err := row.Scan(&t.ID, &number, &t.Title, &t.Status, &assignee, &t.Priority, &t.RequesterID, &t.Requester); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		t.Number = number
		t.AssigneeID = assignee
		c.JSON(http.StatusOK, t)
	}
}

// Update allows changing assignee and/or priority and status
func Update(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, Ticket{})
			return
		}
		var in struct {
			AssigneeID *string `json:"assignee_id"`
			Priority   *int16  `json:"priority"`
			Status     *string `json:"status"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		set := []string{}
		args := []any{}
		idx := 1
		if in.AssigneeID != nil {
			set = append(set, fmt.Sprintf("assignee_id=$%d", idx))
			args = append(args, *in.AssigneeID)
			idx++
		}
		if in.Priority != nil {
			set = append(set, fmt.Sprintf("priority=$%d", idx))
			args = append(args, *in.Priority)
			idx++
		}
		if in.Status != nil {
			set = append(set, fmt.Sprintf("status=$%d", idx))
			args = append(args, *in.Status)
			idx++
		}
		if len(set) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
			return
		}
		args = append(args, c.Param("id"))
		sql := fmt.Sprintf("update tickets set %s, updated_at=now() where id=$%d returning id::text, number, title, status, assignee_id::text, priority", strings.Join(set, ","), idx)
		var t Ticket
		var assignee *string
		var number any
		row := a.DB.QueryRow(c.Request.Context(), sql, args...)
		if err := row.Scan(&t.ID, &number, &t.Title, &t.Status, &assignee, &t.Priority); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		t.Number = number
		t.AssigneeID = assignee
		c.JSON(http.StatusOK, t)
	}
}
