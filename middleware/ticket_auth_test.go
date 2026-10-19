package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTicketAuthAllowsRiskBannedSessionButUserAuthRejectsIt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ticket-auth.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	oldDB := model.DB
	oldRedisEnabled := common.RedisEnabled
	model.DB = db
	common.RedisEnabled = false
	t.Cleanup(func() { model.DB = oldDB; common.RedisEnabled = oldRedisEnabled })
	user := model.User{Username: "appeal", Role: common.RoleCommonUser, Status: common.UserStatusRiskBanned, AuthVersion: 1}
	require.NoError(t, db.Create(&user).Error)
	bundle, err := service.CreateLoginSession(user.Id, "password", "127.0.0.1", "test")
	require.NoError(t, err)

	request := func(auth gin.HandlerFunc) int {
		engine := gin.New()
		engine.GET("/", auth, func(c *gin.Context) { c.Status(http.StatusNoContent) })
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+bundle.AccessToken)
		engine.ServeHTTP(recorder, req)
		return recorder.Code
	}
	assert.Equal(t, http.StatusNoContent, request(TicketAuth()))
	assert.Equal(t, http.StatusUnauthorized, request(UserAuth()))
}

func TestTonkicAccessTokenScopes(t *testing.T) {
	setupDashboardAuthMiddlewareTest(t)
	gin.SetMode(gin.TestMode)
	user := createMiddlewarePATUser(t, "ticket-scoped-user", "ticket-legacy-user")
	admin := createMiddlewarePATUser(t, "ticket-scoped-admin", "ticket-legacy-admin")
	require.NoError(t, model.DB.Model(admin).Update("role", common.RoleAdminUser).Error)
	read, _ := createMiddlewareScopedToken(t, user.Id, 0, "ticket:read")
	write, _ := createMiddlewareScopedToken(t, user.Id, 0, "ticket:write")
	profile, _ := createMiddlewareScopedToken(t, user.Id, 0, "profile:read")
	forgedAdmin, _ := createMiddlewareScopedToken(t, user.Id, 0, "ticket_admin:read", "risk:read")
	riskRead, _ := createMiddlewareScopedToken(t, admin.Id, 0, "risk:read")
	riskWrite, _ := createMiddlewareScopedToken(t, admin.Id, 0, "risk:write")
	ticketAdmin, _ := createMiddlewareScopedToken(t, admin.Id, 0, "ticket_admin:read")
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	engine := gin.New()
	engine.GET("/api/ticket", TicketAuth(), ok)
	engine.POST("/api/ticket", TicketAuth(), ok)
	engine.GET("/api/ticket/admin", AdminAuth(), ok)
	engine.GET("/api/risk/cases", AdminAuth(), ok)
	engine.POST("/api/risk/scan", AdminAuth(), ok)
	engine.GET("/api/status/models", UserAuth(), ok)
	for _, tc := range []struct {
		name, method, path, token string
		status                    int
	}{
		{"ticket read", http.MethodGet, "/api/ticket", read, http.StatusNoContent},
		{"ticket write denied to read", http.MethodPost, "/api/ticket", read, http.StatusForbidden},
		{"ticket write", http.MethodPost, "/api/ticket", write, http.StatusNoContent},
		{"unrelated scope", http.MethodGet, "/api/ticket", profile, http.StatusForbidden},
		{"ticket admin role required", http.MethodGet, "/api/ticket/admin", forgedAdmin, http.StatusForbidden},
		{"risk admin role required", http.MethodGet, "/api/risk/cases", forgedAdmin, http.StatusForbidden},
		{"risk read", http.MethodGet, "/api/risk/cases", riskRead, http.StatusNoContent},
		{"risk write denied to read", http.MethodPost, "/api/risk/scan", riskRead, http.StatusForbidden},
		{"risk write", http.MethodPost, "/api/risk/scan", riskWrite, http.StatusNoContent},
		{"ticket admin read", http.MethodGet, "/api/ticket/admin", ticketAdmin, http.StatusNoContent},
		{"model status", http.MethodGet, "/api/status/models", profile, http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, req)
			assert.Equal(t, tc.status, response.Code, response.Body.String())
		})
	}
	_, err := service.NormalizeAccessTokenScopes(user.Id, common.RoleCommonUser, []string{"ticket:read", "ticket:write"})
	require.NoError(t, err)
	for _, scope := range []string{"ticket_admin:read", "ticket_admin:write", "risk:read", "risk:write"} {
		_, err = service.NormalizeAccessTokenScopes(user.Id, common.RoleCommonUser, []string{scope})
		assert.ErrorIs(t, err, service.ErrAccessTokenScopeForbidden, scope)
	}
}
