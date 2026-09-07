package middleware

import (
	"net/http"
	"net/http/httptest"
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
	db, err := gorm.Open(sqlite.Open("file:ticket-auth?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}))
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
