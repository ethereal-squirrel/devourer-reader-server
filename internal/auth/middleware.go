package auth

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
)

const (
	CtxUserID    = "user_id"
	CtxUserRoles = "user_roles"
)

func CheckAuth(d *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		publicCfg, err := queries.GetConfig(d, "allow_public")
		allowPublic := err == nil && publicCfg.Value == "1"

		if allowPublic {
			c.Set(CtxUserID, int64(0))
			c.Set(CtxUserRoles, RolesData["user"])
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		var userID int64
		userRoles := RolesData["user"]
		authenticated := false

		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := VerifyToken(d, tokenStr)
			if err == nil {
				user, err := queries.GetUserByID(d, claims.UserID)
				if err == nil {
					userID = user.ID
					userRoles = ResolveRoles(user.Roles)
					authenticated = true
				}
			}
		}

		if !authenticated {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status":  false,
				"message": "Authentication required",
			})
			return
		}

		c.Set(CtxUserID, userID)
		c.Set(CtxUserRoles, userRoles)
		c.Next()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get(CtxUserRoles)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"status":  false,
				"message": "You do not have permission to do this.",
			})
			return
		}

		perms, ok := rolesVal.(db.RolePermissions)
		if !ok || !checkPermission(perms, permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"status":  false,
				"message": "You do not have permission to do this.",
			})
			return
		}
		c.Next()
	}
}

func checkPermission(p db.RolePermissions, name string) bool {
	if p.IsAdmin {
		return true
	}
	switch name {
	case "AddFile":
		return p.AddFile
	case "DeleteFile":
		return p.DeleteFile
	case "EditMetadata":
		return p.EditMetadata
	case "ManageCollections":
		return p.ManageCollections
	case "ManageLibrary":
		return p.ManageLibrary
	case "CreateUser":
		return p.CreateUser
	case "IsAdmin":
		return p.IsAdmin
	}
	return false
}

