package users

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"

    apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
    authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

// GetProfile returns the current user's profile from DB or synthesizes one.
func GetProfile(a *apppkg.App) gin.HandlerFunc {
    return func(c *gin.Context) {
        uVal, ok := c.Get("user")
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
            return
        }
        // Type assertion just to access ExternalID later

        // Look up user row by external_id; if missing, return minimal info
        type profile struct {
            ID          string `json:"id,omitempty"`
            Username    string `json:"username,omitempty"`
            Email       string `json:"email,omitempty"`
            DisplayName string `json:"display_name,omitempty"`
        }
        var p profile
        if a.DB != nil {
            au := uVal.(authpkg.AuthUser)
            row := a.DB.QueryRow(c.Request.Context(), `select id::text, coalesce(username,''), coalesce(email,''), coalesce(display_name,'') from users where external_id=$1`, au.ExternalID)
            _ = row.Scan(&p.ID, &p.Username, &p.Email, &p.DisplayName)
        }
        c.JSON(http.StatusOK, p)
    }
}

// UpdateProfile updates email/display_name for local auth only.
func UpdateProfile(a *apppkg.App) gin.HandlerFunc {
    return func(c *gin.Context) {
        if a.Cfg.AuthMode != "local" {
            c.JSON(http.StatusConflict, gin.H{"error": "profile managed by identity provider"})
            return
        }
        uVal, ok := c.Get("user")
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
            return
        }
        au := uVal.(authpkg.AuthUser)
        var in struct {
            Email       *string `json:"email"`
            DisplayName *string `json:"display_name"`
        }
        if err := c.ShouldBindJSON(&in); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
            return
        }
        if in.Email == nil && in.DisplayName == nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
            return
        }
        if in.Email != nil {
            if _, err := a.DB.Exec(c.Request.Context(), `update users set email=$1 where external_id=$2`, *in.Email, au.ExternalID); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
        }
        if in.DisplayName != nil {
            if _, err := a.DB.Exec(c.Request.Context(), `update users set display_name=$1 where external_id=$2`, *in.DisplayName, au.ExternalID); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
        }
        c.JSON(http.StatusOK, gin.H{"ok": true})
    }
}

// ChangePassword changes password for local auth users.
func ChangePassword(a *apppkg.App) gin.HandlerFunc {
    return func(c *gin.Context) {
        if a.Cfg.AuthMode != "local" {
            c.JSON(http.StatusConflict, gin.H{"error": "password managed by identity provider"})
            return
        }
        uVal, ok := c.Get("user")
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
            return
        }
        au := uVal.(authpkg.AuthUser)
        var in struct {
            OldPassword string `json:"old_password"`
            NewPassword string `json:"new_password"`
        }
        if err := c.ShouldBindJSON(&in); err != nil || in.OldPassword == "" || in.NewPassword == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
            return
        }
        var uid, hash string
        if err := a.DB.QueryRow(c.Request.Context(), `select id::text, coalesce(password_hash,'') from users where external_id=$1`, au.ExternalID).Scan(&uid, &hash); err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
            return
        }
        if hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.OldPassword)) != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid old password"})
            return
        }
        ph, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failure"})
            return
        }
        if _, err := a.DB.Exec(c.Request.Context(), `update users set password_hash=$1 where id=$2`, string(ph), uid); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"ok": true})
    }
}

// List returns users with basic fields and stored roles. Optional q filters by
// email/username/display_name (case-insensitive substring). Limited to 100.
func List(a *apppkg.App) gin.HandlerFunc {
    type user struct {
        ID          string   `json:"id"`
        ExternalID  string   `json:"external_id"`
        Username    string   `json:"username"`
        Email       string   `json:"email"`
        DisplayName string   `json:"display_name"`
        Roles       []string `json:"roles"`
    }
    return func(c *gin.Context) {
        q := strings.TrimSpace(c.Query("q"))
        base := `
select u.id::text, coalesce(u.external_id,''), coalesce(u.username,''), coalesce(u.email,''), coalesce(u.display_name,''),
       coalesce(string_agg(distinct r.name, ','), '') as roles
from users u
left join user_roles ur on ur.user_id=u.id
left join roles r on r.id=ur.role_id`
        where := ""
        args := []any{}
        if q != "" {
            where = " where lower(u.email) like $1 or lower(u.username) like $1 or lower(u.display_name) like $1"
            args = append(args, "%"+strings.ToLower(q)+"%")
        }
        sql := base + where + " group by u.id order by u.display_name nulls last, u.email nulls last limit 100"
        rows, err := a.DB.Query(c.Request.Context(), sql, args...)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        defer rows.Close()
        var out []user
        for rows.Next() {
            var u user
            var roles string
            if err := rows.Scan(&u.ID, &u.ExternalID, &u.Username, &u.Email, &u.DisplayName, &roles); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
            if roles != "" {
                u.Roles = strings.Split(roles, ",")
            } else {
                u.Roles = []string{}
            }
            out = append(out, u)
        }
        c.JSON(http.StatusOK, out)
    }
}

// Get returns a single user by id
func Get(a *apppkg.App) gin.HandlerFunc {
    type user struct {
        ID          string `json:"id"`
        ExternalID  string `json:"external_id"`
        Username    string `json:"username"`
        Email       string `json:"email"`
        DisplayName string `json:"display_name"`
    }
    return func(c *gin.Context) {
        if a.DB == nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
            return
        }
        var u user
        row := a.DB.QueryRow(c.Request.Context(), `select id::text, coalesce(external_id,''), coalesce(username,''), coalesce(email,''), coalesce(display_name,'') from users where id=$1`, c.Param("id"))
        if err := row.Scan(&u.ID, &u.ExternalID, &u.Username, &u.Email, &u.DisplayName); err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
            return
        }
        c.JSON(http.StatusOK, u)
    }
}

// CreateLocal creates a local user with username, email, display_name, password.
func CreateLocal(a *apppkg.App) gin.HandlerFunc {
    return func(c *gin.Context) {
        var in struct {
            Username    string `json:"username"`
            Email       string `json:"email"`
            DisplayName string `json:"display_name"`
            Password    string `json:"password"`
        }
        if err := c.ShouldBindJSON(&in); err != nil || in.Username == "" || in.Password == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
            return
        }
        ph, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failure"})
            return
        }
        const q = `
insert into users (external_id, username, email, display_name, password_hash)
values ($1, $2, $3, $4, $5)
on conflict (username) do update set email=excluded.email, display_name=excluded.display_name
returning id::text`
        var id string
        if err := a.DB.QueryRow(c.Request.Context(), q, "local:"+in.Username, in.Username, in.Email, in.DisplayName, string(ph)).Scan(&id); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusCreated, gin.H{"id": id})
    }
}

// ListRoles returns all role names.
func ListRoles(a *apppkg.App) gin.HandlerFunc {
    return func(c *gin.Context) {
        rows, err := a.DB.Query(c.Request.Context(), `select name from roles order by name`)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        defer rows.Close()
        out := []string{}
        for rows.Next() {
            var name *string
            if err := rows.Scan(&name); err == nil {
                if name != nil && *name != "" {
                    out = append(out, *name)
                }
            }
        }
        c.JSON(http.StatusOK, out)
    }
}
