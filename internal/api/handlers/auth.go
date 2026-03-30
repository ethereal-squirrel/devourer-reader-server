package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/audit"
	"github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
)

func (h *Handlers) Login(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Username == "" || body.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Username and password are required"})
		return
	}

	resp, err := auth.HandleLogin(h.DB, body.Username, body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	if !resp["status"].(bool) {
		audit.LoginFailed(c.ClientIP(), body.Username)
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	audit.Login(c.ClientIP(), body.Username)
	c.JSON(http.StatusOK, resp)
}

// CreateUser handles POST /users
func (h *Handlers) CreateUser(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Username == "" || body.Password == "" || body.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "All fields are required"})
		return
	}

	resp, err := auth.HandleRegister(h.DB, body.Username, body.Password, body.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	if !resp["status"].(bool) {
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	audit.UserCreated(c.ClientIP(), body.Username, body.Role)
	c.JSON(http.StatusOK, resp)
}

// GetRoles handles GET /roles
func (h *Handlers) GetRoles(c *gin.Context) {
	userID, _ := c.Get(auth.CtxUserID)
	rolesVal, _ := c.Get(auth.CtxUserRoles)

	uid, _ := userID.(int64)
	user, err := queries.GetUserByID(h.DB, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "User not found"})
		return
	}

	perms, _ := rolesVal.(db.RolePermissions)
	c.JSON(http.StatusOK, gin.H{
		"status":   true,
		"roles":    perms,
		"username": user.Email,
	})
}

// ListUsers handles GET /users
func (h *Handlers) ListUsers(c *gin.Context) {
	users, err := queries.ListUsers(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}

	type userOut struct {
		ID          int64  `json:"id"`
		Email       string `json:"email"`
		Roles       any    `json:"roles"`
		Collections int    `json:"collections"`
	}
	out := make([]userOut, 0, len(users))
	for _, u := range users {
		count, _ := queries.CountCollectionsByUser(h.DB, u.ID)
		out = append(out, userOut{
			ID:          u.ID,
			Email:       u.Email,
			Roles:       u.Roles,
			Collections: count,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "users": out})
}

// DeleteUser handles DELETE /user/:id
func (h *Handlers) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid user id"})
		return
	}

	resp, err := auth.HandleDeleteUser(h.DB, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	if !resp["status"].(bool) {
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	audit.UserDeleted(c.ClientIP(), id)
	c.JSON(http.StatusOK, resp)
}

// EditUser handles PATCH /user/:id
func (h *Handlers) EditUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid user id"})
		return
	}

	var body struct {
		Role     string  `json:"role"`
		Password *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Role is required"})
		return
	}

	resp, err := auth.HandleEditUser(h.DB, id, body.Role, body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	if !resp["status"].(bool) {
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	audit.UserEdited(c.ClientIP(), id, body.Role)
	c.JSON(http.StatusOK, resp)
}
