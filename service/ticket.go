package service

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

var (
	ErrTicketTitleRequired   = errors.New("ticket title is required")
	ErrTicketTitleTooLong    = errors.New("ticket title must not exceed 120 characters")
	ErrTicketCategoryInvalid = errors.New("ticket category is invalid")
	ErrTicketContentRequired = errors.New("ticket content is required")
	ErrTicketContentTooLong  = errors.New("ticket content must not exceed 5000 characters")
	ErrTicketStatusInvalid   = errors.New("ticket status is invalid")
	ErrTicketPriorityInvalid = errors.New("ticket priority is invalid")
	ErrTicketUpdateRequired  = errors.New("ticket update is required")
)

type TicketDetail struct {
	model.Ticket
	Messages []model.TicketMessage `json:"messages"`
}

type AdminTicketDetail struct {
	TicketDetail
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func CreateTicket(userID int, request dto.CreateTicketRequest) (*TicketDetail, error) {
	title := strings.TrimSpace(request.Title)
	content := strings.TrimSpace(request.Content)
	category := strings.TrimSpace(request.Category)
	if title == "" {
		return nil, ErrTicketTitleRequired
	}
	if utf8.RuneCountInString(title) > 120 {
		return nil, ErrTicketTitleTooLong
	}
	if !isValidTicketCategory(category) {
		return nil, ErrTicketCategoryInvalid
	}
	if content == "" {
		return nil, ErrTicketContentRequired
	}
	if utf8.RuneCountInString(content) > 5000 {
		return nil, ErrTicketContentTooLong
	}

	ticket := model.Ticket{
		UserId:   userID,
		Title:    title,
		Category: category,
		Status:   model.TicketStatusPending,
		Priority: model.TicketPriorityNormal,
	}
	message := model.TicketMessage{UserId: userID, Content: content}
	if err := model.CreateTicketWithMessage(&ticket, &message); err != nil {
		return nil, err
	}
	return &TicketDetail{Ticket: ticket, Messages: []model.TicketMessage{message}}, nil
}

func ListTickets(userID, status, offset, limit int) ([]model.Ticket, int64, error) {
	return model.ListTicketsByUser(userID, status, offset, limit)
}

func AdminListTickets(filter model.TicketListFilter, offset, limit int) ([]model.TicketListItem, int64, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	return model.ListAllTickets(filter, offset, limit)
}

func GetTicket(userID, ticketID int) (*TicketDetail, error) {
	ticket, messages, err := model.GetTicketByIDForUser(ticketID, userID)
	if err != nil {
		return nil, err
	}
	return &TicketDetail{Ticket: *ticket, Messages: messages}, nil
}

func ReplyTicket(actorID, ticketID int, isAdmin bool, request dto.ReplyTicketRequest) (*TicketDetail, error) {
	content := strings.TrimSpace(request.Content)
	if content == "" {
		return nil, ErrTicketContentRequired
	}
	if utf8.RuneCountInString(content) > 5000 {
		return nil, ErrTicketContentTooLong
	}

	ownerID := actorID
	nextStatus := model.TicketStatusPending
	if isAdmin {
		ownerID = 0
		nextStatus = model.TicketStatusAnswered
	}
	message := model.TicketMessage{UserId: actorID, IsAdmin: isAdmin, Content: content}
	if err := model.AddTicketMessage(ticketID, ownerID, &message, nextStatus); err != nil {
		return nil, err
	}
	ticket, messages, err := model.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}
	return &TicketDetail{Ticket: *ticket, Messages: messages}, nil
}

func CloseTicket(userID, ticketID int) (*TicketDetail, error) {
	if err := model.CloseTicket(ticketID, userID); err != nil {
		return nil, err
	}
	return GetTicket(userID, ticketID)
}

func AdminGetTicket(ticketID int) (*AdminTicketDetail, error) {
	ticket, messages, err := model.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}
	username, displayName, err := model.GetTicketOwner(ticket.UserId)
	if err != nil {
		return nil, err
	}
	return &AdminTicketDetail{
		TicketDetail: TicketDetail{Ticket: *ticket, Messages: messages},
		Username:     username, DisplayName: displayName,
	}, nil
}

func AdminUpdateTicket(ticketID int, request dto.AdminUpdateTicketRequest) (*AdminTicketDetail, error) {
	if request.Status == nil && request.Priority == nil {
		return nil, ErrTicketUpdateRequired
	}
	if request.Status != nil && !isValidTicketStatus(*request.Status) {
		return nil, ErrTicketStatusInvalid
	}
	if request.Priority != nil && !isValidTicketPriority(*request.Priority) {
		return nil, ErrTicketPriorityInvalid
	}
	if err := model.UpdateTicketByAdmin(ticketID, request.Status, request.Priority); err != nil {
		return nil, err
	}
	return AdminGetTicket(ticketID)
}

func isValidTicketCategory(category string) bool {
	switch category {
	case "account", "billing", "api", "bug", "suggestion", "other":
		return true
	default:
		return false
	}
}

func isValidTicketStatus(status int) bool {
	return status >= model.TicketStatusPending && status <= model.TicketStatusClosed
}

func isValidTicketPriority(priority int) bool {
	return priority >= model.TicketPriorityLow && priority <= model.TicketPriorityHigh
}
