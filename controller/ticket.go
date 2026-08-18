package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func CreateTicket(c *gin.Context) {
	var request dto.CreateTicketRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if c.GetInt("status") == common.UserStatusRiskBanned && request.Category != "account" {
		common.ApiErrorMsg(c, "risk-banned users may only create account appeal tickets")
		return
	}
	ticket, err := service.CreateTicket(c.GetInt("id"), request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

func ListTickets(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := 0
	if rawStatus := c.Query("status"); rawStatus != "" {
		parsedStatus, err := strconv.Atoi(rawStatus)
		if err != nil || parsedStatus < model.TicketStatusPending || parsedStatus > model.TicketStatusClosed {
			common.ApiErrorMsg(c, "invalid ticket status")
			return
		}
		status = parsedStatus
	}
	tickets, total, err := service.ListTickets(c.GetInt("id"), status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tickets)
	common.ApiSuccess(c, pageInfo)
}

func AdminListTickets(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := 0
	if rawStatus := c.Query("status"); rawStatus != "" {
		parsedStatus, err := strconv.Atoi(rawStatus)
		if err != nil || parsedStatus < model.TicketStatusPending || parsedStatus > model.TicketStatusClosed {
			common.ApiErrorMsg(c, "invalid ticket status")
			return
		}
		status = parsedStatus
	}
	priority := 0
	if rawPriority := c.Query("priority"); rawPriority != "" {
		parsedPriority, err := strconv.Atoi(rawPriority)
		if err != nil || parsedPriority < model.TicketPriorityLow || parsedPriority > model.TicketPriorityHigh {
			common.ApiErrorMsg(c, "invalid ticket priority")
			return
		}
		priority = parsedPriority
	}
	filter := model.TicketListFilter{
		Status: status, Priority: priority, Keyword: c.Query("keyword"),
	}
	tickets, total, err := service.AdminListTickets(filter, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tickets)
	common.ApiSuccess(c, pageInfo)
}

func GetTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketID <= 0 {
		common.ApiErrorMsg(c, "invalid ticket id")
		return
	}
	ticket, err := service.GetTicket(c.GetInt("id"), ticketID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

func AdminReplyTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketID <= 0 {
		common.ApiErrorMsg(c, "invalid ticket id")
		return
	}
	var request dto.ReplyTicketRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	ticket, err := service.ReplyTicket(c.GetInt("id"), ticketID, true, request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

func ReplyTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketID <= 0 {
		common.ApiErrorMsg(c, "invalid ticket id")
		return
	}
	var request dto.ReplyTicketRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if c.GetInt("status") == common.UserStatusRiskBanned {
		ticket, err := service.GetTicket(c.GetInt("id"), ticketID)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if ticket.Category != "account" {
			common.ApiErrorMsg(c, "risk-banned users may only reply to account appeal tickets")
			return
		}
	}
	ticket, err := service.ReplyTicket(c.GetInt("id"), ticketID, false, request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

func CloseTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketID <= 0 {
		common.ApiErrorMsg(c, "invalid ticket id")
		return
	}
	ticket, err := service.CloseTicket(c.GetInt("id"), ticketID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

func AdminGetTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketID <= 0 {
		common.ApiErrorMsg(c, "invalid ticket id")
		return
	}
	ticket, err := service.AdminGetTicket(ticketID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

func AdminUpdateTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil || ticketID <= 0 {
		common.ApiErrorMsg(c, "invalid ticket id")
		return
	}
	var request dto.AdminUpdateTicketRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	ticket, err := service.AdminUpdateTicket(ticketID, request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}
