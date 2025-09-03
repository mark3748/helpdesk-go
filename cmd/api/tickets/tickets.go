package tickets

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

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

// List returns recent tickets using cursor pagination.
func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, gin.H{"tickets": []Ticket{}, "next_cursor": ""})
			return
		}

		// helper for multi-value query params (comma or repeated)
		getMulti := func(key string) []string {
			vals := c.QueryArray(key)
			if len(vals) == 0 {
				if v := strings.TrimSpace(c.Query(key)); v != "" {
					vals = []string{v}
				}
			}
			out := []string{}
			for _, val := range vals {
				for _, v := range strings.Split(val, ",") {
					v = strings.TrimSpace(v)
					if v != "" {
						out = append(out, v)
					}
				}
			}
			return out
		}

		where := []string{}
		args := []any{}

		statuses := getMulti("status")
		if len(statuses) > 0 {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.status = ANY($%d)", n))
			args = append(args, statuses)
		} else {
			where = append(where, "t.status <> 'Closed'")
		}

		if ps := getMulti("priority"); len(ps) > 0 {
			nums := []int{}
			for _, v := range ps {
				if p, err := strconv.Atoi(v); err == nil {
					nums = append(nums, p)
				}
			}
			if len(nums) > 0 {
				n := len(args) + 1
				where = append(where, fmt.Sprintf("t.priority = ANY($%d)", n))
				args = append(args, nums)
			}
		}

		if teams := getMulti("team"); len(teams) > 0 {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.team_id = ANY($%d::uuid[])", n))
			args = append(args, teams)
		}

		assignees := getMulti("assignee")
		if len(assignees) == 0 {
			assignees = getMulti("assignee_id")
		}
		if len(assignees) > 0 {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.assignee_id = ANY($%d::uuid[])", n))
			args = append(args, assignees)
		}

		if reqs := getMulti("requester"); len(reqs) > 0 {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.requester_id = ANY($%d::uuid[])", n))
			args = append(args, reqs)
		}

		if qs := getMulti("queue"); len(qs) > 0 {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("t.queue_id = ANY($%d::uuid[])", n))
			args = append(args, qs)
		}

		if v := strings.TrimSpace(c.Query("search")); v != "" {
			n := len(args) + 1
			where = append(where, fmt.Sprintf("(t.title ILIKE $%d OR t.description ILIKE $%d)", n, n))
			args = append(args, "%"+v+"%")
		}

		// cursor handling
		if cur := strings.TrimSpace(c.Query("cursor")); cur != "" {
			if b, err := base64.StdEncoding.DecodeString(cur); err == nil {
				parts := strings.SplitN(string(b), ",", 2)
				if len(parts) == 2 {
					if ts, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
						n := len(args) + 1
						where = append(where, fmt.Sprintf("(t.updated_at, t.id) < ($%d, $%d)", n, n+1))
						args = append(args, ts, parts[1])
					}
				}
			}
		}

		limit := 100
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
				limit = l
			}
		}

		sql := "select t.id::text, t.number, t.title, t.status, t.assignee_id::text, t.priority, t.requester_id::text, coalesce(r.name, r.email, '') as requester, t.updated_at from tickets t left join requesters r on r.id=t.requester_id"
		if len(where) > 0 {
			sql += " where " + strings.Join(where, " and ")
		}
		sql += " order by t.updated_at desc, t.id desc limit $" + strconv.Itoa(len(args)+1)
		args = append(args, limit+1)

		rows, err := a.DB.Query(c.Request.Context(), sql, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		out := []Ticket{}
		ups := []time.Time{}
		for rows.Next() {
			var t Ticket
			var assignee *string
			var number any
			var updated time.Time
			if err := rows.Scan(&t.ID, &number, &t.Title, &t.Status, &assignee, &t.Priority, &t.RequesterID, &t.Requester, &updated); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			t.Number = number
			t.AssigneeID = assignee
			out = append(out, t)
			ups = append(ups, updated)
		}

		var next string
		if len(out) > limit {
			last := out[limit-1]
			lastUp := ups[limit-1]
			next = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s,%s", lastUp.UTC().Format(time.RFC3339Nano), last.ID)))
			out = out[:limit]
		}

		c.JSON(http.StatusOK, gin.H{"tickets": out, "next_cursor": next})
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
