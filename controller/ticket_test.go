package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTicketControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := model.DB
	previousDatabaseType := common.MainDatabaseType()
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Ticket{}, &model.TicketMessage{}))

	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousDatabaseType)
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func performTicketRequest(t *testing.T, userID int, method, path, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", userID)
	c.Set("role", common.RoleCommonUser)
	handler(c)
	return recorder
}

func performTicketRequestWithID(t *testing.T, userID, ticketID int, method, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, fmt.Sprintf("/api/ticket/%d", ticketID), strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", ticketID)}}
	c.Set("id", userID)
	c.Set("role", common.RoleCommonUser)
	handler(c)
	return recorder
}

func performAdminTicketRequestWithID(t *testing.T, adminID, ticketID int, method, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	recorder := performTicketRequestWithID(t, adminID, ticketID, method, body, handler)
	return recorder
}

func TestCreateTicketCreatesConversationForAuthenticatedUser(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	user := model.User{Username: "ticket-user", DisplayName: "Ticket User", Role: common.RoleCommonUser}
	require.NoError(t, db.Create(&user).Error)

	recorder := performTicketRequest(t, user.Id, http.MethodPost, "/api/ticket", `{
		"title":"API requests fail",
		"category":"api",
		"content":"Requests return an unexpected upstream error."
	}`, CreateTicket)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			ID       int `json:"id"`
			UserID   int `json:"user_id"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.NotZero(t, response.Data.ID)
	assert.Equal(t, user.Id, response.Data.UserID)
	require.Len(t, response.Data.Messages, 1)
	assert.Equal(t, "Requests return an unexpected upstream error.", response.Data.Messages[0].Content)
}

func TestListTicketsOnlyReturnsAuthenticatedUsersTickets(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	firstUser := model.User{Username: "first-ticket-user", Role: common.RoleCommonUser, AffCode: "ticket-first"}
	secondUser := model.User{Username: "second-ticket-user", Role: common.RoleCommonUser, AffCode: "ticket-second"}
	require.NoError(t, db.Create(&firstUser).Error)
	require.NoError(t, db.Create(&secondUser).Error)

	createBody := `{"title":"Own ticket","category":"other","content":"Visible to its owner."}`
	firstCreate := performTicketRequest(t, firstUser.Id, http.MethodPost, "/api/ticket", createBody, CreateTicket)
	secondCreate := performTicketRequest(t, secondUser.Id, http.MethodPost, "/api/ticket", createBody, CreateTicket)
	require.Contains(t, firstCreate.Body.String(), `"success":true`)
	require.Contains(t, secondCreate.Body.String(), `"success":true`)

	recorder := performTicketRequest(t, firstUser.Id, http.MethodGet, "/api/ticket?p=1&page_size=10", "", ListTickets)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Total int `json:"total"`
			Items []struct {
				UserID int `json:"user_id"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	assert.Equal(t, firstUser.Id, response.Data.Items[0].UserID)
}

func TestListTicketsReturnsEmptyArrayWhenUserHasNoTickets(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	user := model.User{Username: "empty-ticket-user", Role: common.RoleCommonUser}
	require.NoError(t, db.Create(&user).Error)

	recorder := performTicketRequest(t, user.Id, http.MethodGet,
		"/api/ticket?p=1&page_size=10", "", ListTickets)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Total int            `json:"total"`
			Items []model.Ticket `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, 0, response.Data.Total)
	assert.NotNil(t, response.Data.Items)
	assert.Empty(t, response.Data.Items)
}

func TestGetTicketDoesNotExposeAnotherUsersConversation(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := model.User{Username: "ticket-owner", Role: common.RoleCommonUser, AffCode: "ticket-owner"}
	otherUser := model.User{Username: "ticket-other", Role: common.RoleCommonUser, AffCode: "ticket-other"}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&otherUser).Error)

	createRecorder := performTicketRequest(t, owner.Id, http.MethodPost, "/api/ticket", `{
		"title":"Private billing issue",
		"category":"billing",
		"content":"This conversation belongs to the owner."
	}`, CreateTicket)
	var created struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(createRecorder.Body.Bytes(), &created))
	require.NotZero(t, created.Data.ID)

	recorder := performTicketRequestWithID(t, otherUser.Id, created.Data.ID, http.MethodGet, "", GetTicket)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.NotContains(t, recorder.Body.String(), "This conversation belongs to the owner.")
}

func TestAdminReplyMarksTicketAnsweredAndAddsAdminMessage(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := model.User{Username: "reply-ticket-owner", Role: common.RoleCommonUser, AffCode: "reply-owner"}
	admin := model.User{Username: "reply-ticket-admin", Role: common.RoleAdminUser, AffCode: "reply-admin"}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&admin).Error)

	createRecorder := performTicketRequest(t, owner.Id, http.MethodPost, "/api/ticket", `{
		"title":"Need administrator help",
		"category":"account",
		"content":"Please review this account issue."
	}`, CreateTicket)
	var created struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(createRecorder.Body.Bytes(), &created))

	recorder := performAdminTicketRequestWithID(t, admin.Id, created.Data.ID, http.MethodPost,
		`{"content":"The account has been reviewed."}`, AdminReplyTicket)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Status   int `json:"status"`
			Messages []struct {
				UserID  int    `json:"user_id"`
				IsAdmin bool   `json:"is_admin"`
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, model.TicketStatusAnswered, response.Data.Status)
	require.Len(t, response.Data.Messages, 2)
	assert.Equal(t, admin.Id, response.Data.Messages[1].UserID)
	assert.True(t, response.Data.Messages[1].IsAdmin)
	assert.Equal(t, "The account has been reviewed.", response.Data.Messages[1].Content)
}

func TestUserReplyMarksAnsweredTicketPendingAgain(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := model.User{Username: "user-reply-owner", Role: common.RoleCommonUser, AffCode: "user-reply-owner"}
	admin := model.User{Username: "user-reply-admin", Role: common.RoleAdminUser, AffCode: "user-reply-admin"}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&admin).Error)

	created, firstMessage := model.Ticket{
		UserId: owner.Id, Title: "Reply status", Category: "api",
		Status: model.TicketStatusAnswered, Priority: model.TicketPriorityNormal,
	}, model.TicketMessage{UserId: owner.Id, Content: "Initial message"}
	require.NoError(t, model.CreateTicketWithMessage(&created, &firstMessage))
	require.NoError(t, db.Model(&created).Update("status", model.TicketStatusAnswered).Error)

	recorder := performTicketRequestWithID(t, owner.Id, created.Id, http.MethodPost,
		`{"content":"I still need help with this request."}`, ReplyTicket)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Status   int `json:"status"`
			Messages []struct {
				IsAdmin bool `json:"is_admin"`
			} `json:"messages"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, model.TicketStatusPending, response.Data.Status)
	require.Len(t, response.Data.Messages, 2)
	assert.False(t, response.Data.Messages[1].IsAdmin)
}

func TestClosingTicketPreventsFurtherReplies(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := model.User{Username: "close-ticket-owner", Role: common.RoleCommonUser, AffCode: "close-ticket-owner"}
	require.NoError(t, db.Create(&owner).Error)

	ticket, firstMessage := model.Ticket{
		UserId: owner.Id, Title: "Close this ticket", Category: "other",
		Status: model.TicketStatusPending, Priority: model.TicketPriorityNormal,
	}, model.TicketMessage{UserId: owner.Id, Content: "This can now be closed."}
	require.NoError(t, model.CreateTicketWithMessage(&ticket, &firstMessage))

	closeRecorder := performTicketRequestWithID(t, owner.Id, ticket.Id, http.MethodPatch, "", CloseTicket)
	assert.Contains(t, closeRecorder.Body.String(), `"success":true`)
	assert.Contains(t, closeRecorder.Body.String(), fmt.Sprintf(`"status":%d`, model.TicketStatusClosed))

	replyRecorder := performTicketRequestWithID(t, owner.Id, ticket.Id, http.MethodPost,
		`{"content":"This reply must be rejected."}`, ReplyTicket)
	assert.Contains(t, replyRecorder.Body.String(), `"success":false`)
	assert.Contains(t, replyRecorder.Body.String(), "ticket is closed")
}

func TestAdminListTicketsFiltersStatusAndIncludesOwner(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := model.User{Username: "admin-list-owner", DisplayName: "Admin List Owner", Role: common.RoleCommonUser, AffCode: "admin-list-owner"}
	require.NoError(t, db.Create(&owner).Error)

	pending, pendingMessage := model.Ticket{
		UserId: owner.Id, Title: "Pending ticket", Category: "api",
		Status: model.TicketStatusPending, Priority: model.TicketPriorityHigh,
	}, model.TicketMessage{UserId: owner.Id, Content: "Pending content"}
	closed, closedMessage := model.Ticket{
		UserId: owner.Id, Title: "Closed ticket", Category: "billing",
		Status: model.TicketStatusClosed, Priority: model.TicketPriorityNormal,
	}, model.TicketMessage{UserId: owner.Id, Content: "Closed content"}
	require.NoError(t, model.CreateTicketWithMessage(&pending, &pendingMessage))
	require.NoError(t, model.CreateTicketWithMessage(&closed, &closedMessage))
	require.NoError(t, db.Model(&closed).Update("status", model.TicketStatusClosed).Error)

	recorder := performTicketRequest(t, 999, http.MethodGet,
		fmt.Sprintf("/api/ticket/admin?p=1&page_size=10&status=%d", model.TicketStatusPending), "", AdminListTickets)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Total int `json:"total"`
			Items []struct {
				ID          int    `json:"id"`
				Username    string `json:"username"`
				DisplayName string `json:"display_name"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	assert.Equal(t, pending.Id, response.Data.Items[0].ID)
	assert.Equal(t, owner.Username, response.Data.Items[0].Username)
	assert.Equal(t, owner.DisplayName, response.Data.Items[0].DisplayName)
}

func TestAdminListTicketsReturnsEmptyArrayWhenNoTickets(t *testing.T) {
	setupTicketControllerTestDB(t)

	recorder := performTicketRequest(t, 999, http.MethodGet,
		"/api/ticket/admin?p=1&page_size=10", "", AdminListTickets)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Total int            `json:"total"`
			Items []model.Ticket `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, 0, response.Data.Total)
	assert.NotNil(t, response.Data.Items)
	assert.Empty(t, response.Data.Items)
}

func TestAdminListTicketsRejectsInvalidFilters(t *testing.T) {
	setupTicketControllerTestDB(t)

	recorder := performTicketRequest(t, 999, http.MethodGet,
		"/api/ticket/admin?p=1&page_size=10&priority=urgent", "", AdminListTickets)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.Contains(t, recorder.Body.String(), "invalid ticket priority")
}

func TestAdminCanUpdateTicketStatusAndPriority(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := model.User{Username: "admin-update-owner", Role: common.RoleCommonUser, AffCode: "admin-update-owner"}
	require.NoError(t, db.Create(&owner).Error)
	ticket, message := model.Ticket{
		UserId: owner.Id, Title: "Escalate ticket", Category: "bug",
		Status: model.TicketStatusPending, Priority: model.TicketPriorityNormal,
	}, model.TicketMessage{UserId: owner.Id, Content: "This issue needs escalation."}
	require.NoError(t, model.CreateTicketWithMessage(&ticket, &message))

	recorder := performAdminTicketRequestWithID(t, 999, ticket.Id, http.MethodPatch,
		fmt.Sprintf(`{"status":%d,"priority":%d}`, model.TicketStatusClosed, model.TicketPriorityHigh),
		AdminUpdateTicket)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Status     int   `json:"status"`
			Priority   int   `json:"priority"`
			ClosedTime int64 `json:"closed_time"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, model.TicketStatusClosed, response.Data.Status)
	assert.Equal(t, model.TicketPriorityHigh, response.Data.Priority)
	assert.NotZero(t, response.Data.ClosedTime)
}

func TestAdminCanReadTicketAfterOwnerDeletion(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := model.User{Username: "deleted-ticket-owner", Role: common.RoleCommonUser, AffCode: "deleted-ticket-owner"}
	require.NoError(t, db.Create(&owner).Error)
	ticket, message := model.Ticket{
		UserId: owner.Id, Title: "Historical ticket", Category: "account",
		Status: model.TicketStatusPending, Priority: model.TicketPriorityNormal,
	}, model.TicketMessage{UserId: owner.Id, Content: "Keep this conversation available."}
	require.NoError(t, model.CreateTicketWithMessage(&ticket, &message))
	require.NoError(t, db.Unscoped().Delete(&owner).Error)

	recorder := performAdminTicketRequestWithID(t, 999, ticket.Id, http.MethodGet, "", AdminGetTicket)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), "Keep this conversation available.")
}
