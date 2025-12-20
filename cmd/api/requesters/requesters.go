package requesters

import (
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// Requester represents a requester record.
type Requester struct {
	ID    string `json:"id"`
	Email string `json:"email,omitempty"`
	Name  string `json:"display_name,omitempty"`
	Phone string `json:"phone,omitempty"`
}

var phoneRe = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

// ValidEmail validates basic email format.
func ValidEmail(e string) bool {
	if e == "" {
		return false
	}
	_, err := mail.ParseAddress(e)
	return err == nil
}

// ValidPhone validates a simple international phone number.
func ValidPhone(p string) bool {
	if p == "" {
		return false
	}
	return phoneRe.MatchString(p)
}

// Create inserts a requester.
func Create(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in struct {
			Email string `json:"email"`
			Name  string `json:"display_name"`
			Phone string `json:"phone"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if in.Email == "" && in.Phone == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email_or_phone_required"})
			return
		}
		if in.Email != "" && !ValidEmail(in.Email) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_email"})
			return
		}
		if in.Phone != "" && !ValidPhone(in.Phone) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_phone"})
			return
		}
		if a.DB == nil {
			c.JSON(http.StatusCreated, Requester{Email: in.Email, Name: in.Name, Phone: in.Phone})
			return
		}
		const q = `
        insert into requesters (email, name, phone)
        values (nullif(lower($1),''), nullif($2,''), nullif($3,''))
        on conflict (email) do update set name = coalesce(excluded.name, requesters.name)
        returning id::text, coalesce(email,''), coalesce(name,''), coalesce(phone,'')`
		var r Requester
		if err := a.DB.QueryRow(c.Request.Context(), q, strings.ToLower(in.Email), in.Name, in.Phone).Scan(&r.ID, &r.Email, &r.Name, &r.Phone); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, r)
	}
}

// Get returns a requester by id.
func Get(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, Requester{ID: c.Param("id")})
			return
		}
		const q = `select id::text, coalesce(email,''), coalesce(name,''), coalesce(phone,'') from requesters where id=$1`
		var r Requester
		if err := a.DB.QueryRow(c.Request.Context(), q, c.Param("id")).Scan(&r.ID, &r.Email, &r.Name, &r.Phone); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, r)
	}
}

// Update modifies fields on a requester.
func Update(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in struct {
			Email *string `json:"email"`
			Name  *string `json:"display_name"`
			Phone *string `json:"phone"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if in.Email == nil && in.Name == nil && in.Phone == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
			return
		}
		set := []string{}
		args := []any{}
		idx := 1
		if in.Email != nil {
			if *in.Email != "" && !ValidEmail(*in.Email) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_email"})
				return
			}
			set = append(set, fmt.Sprintf("email=$%d", idx))
			args = append(args, strings.ToLower(*in.Email))
			idx++
		}
		if in.Name != nil {
			set = append(set, fmt.Sprintf("name=$%d", idx))
			args = append(args, *in.Name)
			idx++
		}
		if in.Phone != nil {
			if *in.Phone != "" && !ValidPhone(*in.Phone) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_phone"})
				return
			}
			// Store empty string as NULL for consistency with Create
			set = append(set, fmt.Sprintf("phone=nullif($%d,'')", idx))
			args = append(args, *in.Phone)
			idx++
		}
		if a.DB == nil {
			c.JSON(http.StatusOK, Requester{ID: c.Param("id")})
			return
		}
		args = append(args, c.Param("id"))
		sql := fmt.Sprintf("update requesters set %s where id=$%d returning id::text, coalesce(email,''), coalesce(name,''), coalesce(phone,'')", strings.Join(set, ","), idx)
		var r Requester
		if err := a.DB.QueryRow(c.Request.Context(), sql, args...).Scan(&r.ID, &r.Email, &r.Name, &r.Phone); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, r)
	}
}

// Search finds requesters by name or email.
func Search(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, []Requester{})
			return
		}
		q := strings.TrimSpace(c.Query("q"))
		if q == "" {
			c.JSON(http.StatusOK, []Requester{})
			return
		}
		// Sanitize input to prevent wildcard abuse (performance)
		q = strings.ReplaceAll(q, "%", "\\%")
		q = strings.ReplaceAll(q, "_", "\\_")
		pattern := "%" + q + "%"
		// Limit to 20 results
		const sql = `
			select id::text, coalesce(email,''), coalesce(name,''), coalesce(phone,'')
			from requesters
			where name ILIKE $1 or email ILIKE $1
			order by name asc, email asc
			limit 20`
		rows, err := a.DB.Query(c.Request.Context(), sql, pattern)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []Requester{}
		for rows.Next() {
			var r Requester
			if err := rows.Scan(&r.ID, &r.Email, &r.Name, &r.Phone); err != nil {
				continue
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}
