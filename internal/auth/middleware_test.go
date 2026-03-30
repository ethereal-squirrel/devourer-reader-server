package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/devourer/server/internal/db"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// checkPermission
// ---------------------------------------------------------------------------

func TestCheckPermission_AdminBypassesAll(t *testing.T) {
	perms := db.RolePermissions{IsAdmin: true}
	for _, name := range []string{"AddFile", "DeleteFile", "EditMetadata", "ManageCollections", "ManageLibrary", "CreateUser", "IsAdmin"} {
		assert.True(t, checkPermission(perms, name), "admin should pass %s", name)
	}
}

func TestCheckPermission_SpecificPerms(t *testing.T) {
	perms := db.RolePermissions{AddFile: true}

	assert.True(t, checkPermission(perms, "AddFile"))
	assert.False(t, checkPermission(perms, "DeleteFile"))
	assert.False(t, checkPermission(perms, "EditMetadata"))
	assert.False(t, checkPermission(perms, "ManageCollections"))
	assert.False(t, checkPermission(perms, "ManageLibrary"))
	assert.False(t, checkPermission(perms, "CreateUser"))
}

func TestCheckPermission_UnknownPermissionName(t *testing.T) {
	perms := db.RolePermissions{IsAdmin: false, AddFile: true}
	assert.False(t, checkPermission(perms, "NonExistentPermission"))
}

func TestCheckPermission_AllPermissions(t *testing.T) {
	perms := db.RolePermissions{
		AddFile: true, DeleteFile: true, EditMetadata: true,
		ManageCollections: true, ManageLibrary: true, CreateUser: true,
	}
	for _, name := range []string{"AddFile", "DeleteFile", "EditMetadata", "ManageCollections", "ManageLibrary", "CreateUser"} {
		assert.True(t, checkPermission(perms, name), "expected %s to be allowed", name)
	}
}

// ---------------------------------------------------------------------------
// RequirePermission middleware
// ---------------------------------------------------------------------------

func makeGinContext(w *httptest.ResponseRecorder) (*gin.Context, *gin.Engine) {
	router := gin.New()
	c, _ := gin.CreateTestContext(w)
	return c, router
}

func TestRequirePermission_BlocksWhenNoRolesInContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := makeGinContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler := RequirePermission("AddFile")
	handler(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_BlocksWhenPermissionNotGranted(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := makeGinContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(CtxUserRoles, db.RolePermissions{AddFile: false})

	handler := RequirePermission("AddFile")
	handler(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_AllowsMatchingPermission(t *testing.T) {
	router := gin.New()
	router.GET("/test", RequirePermission("AddFile"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	setupRouter := gin.New()
	setupRouter.GET("/test",
		func(c *gin.Context) {
			c.Set(CtxUserRoles, db.RolePermissions{AddFile: true})
			c.Next()
		},
		RequirePermission("AddFile"),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	setupRouter.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_AdminPassesAll(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	router := gin.New()
	router.GET("/test",
		func(c *gin.Context) {
			c.Set(CtxUserRoles, db.RolePermissions{IsAdmin: true})
			c.Next()
		},
		RequirePermission("DeleteFile"),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
