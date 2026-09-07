package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

var ErrTicketClosed = errors.New("ticket is closed")

const (
	TicketStatusPending  = 1
	TicketStatusAnswered = 2
	TicketStatusClosed   = 3
)

const (
	TicketPriorityLow    = 1
	TicketPriorityNormal = 2
	TicketPriorityHigh   = 3
)

type Ticket struct {
	Id            int    `json:"id"`
	UserId        int    `json:"user_id" gorm:"index;not null"`
	Title         string `json:"title" gorm:"size:120;not null"`
	Category      string `json:"category" gorm:"size:32;not null"`
	Status        int    `json:"status" gorm:"index;not null"`
	Priority      int    `json:"priority" gorm:"index;not null"`
	CreatedTime   int64  `json:"created_time" gorm:"bigint;index"`
	UpdatedTime   int64  `json:"updated_time" gorm:"bigint"`
	ClosedTime    int64  `json:"closed_time" gorm:"bigint"`
	LastReplyTime int64  `json:"last_reply_time" gorm:"bigint;index"`
	LastReplyBy   int    `json:"last_reply_by"`
}

type TicketMessage struct {
	Id          int    `json:"id"`
	TicketId    int    `json:"ticket_id" gorm:"index;not null"`
	UserId      int    `json:"user_id" gorm:"index;not null"`
	IsAdmin     bool   `json:"is_admin"`
	Content     string `json:"content" gorm:"type:text;not null"`
	CreatedTime int64  `json:"created_time" gorm:"bigint;index"`
}

type TicketListFilter struct {
	Status   int
	Priority int
	Keyword  string
}

type TicketListItem struct {
	Ticket
	Username    string `json:"username" gorm:"column:username"`
	DisplayName string `json:"display_name" gorm:"column:display_name"`
}

func CreateTicketWithMessage(ticket *Ticket, message *TicketMessage) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		ticket.CreatedTime = now
		ticket.UpdatedTime = now
		ticket.LastReplyTime = now
		ticket.LastReplyBy = ticket.UserId
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}
		message.TicketId = ticket.Id
		message.CreatedTime = now
		return tx.Create(message).Error
	})
}

func ListTicketsByUser(userID, status, offset, limit int) ([]Ticket, int64, error) {
	query := DB.Model(&Ticket{}).Where("user_id = ?", userID)
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	// Keep an empty result JSON-serializable as [] rather than null. The
	// dashboard treats these fields as collections and calls .length/.map on
	// them, so a nil slice would turn a valid empty page into a client error.
	tickets := make([]Ticket, 0)
	if err := query.Order("updated_time DESC, id DESC").Offset(offset).Limit(limit).Find(&tickets).Error; err != nil {
		return nil, 0, err
	}
	return tickets, total, nil
}

func ListAllTickets(filter TicketListFilter, offset, limit int) ([]TicketListItem, int64, error) {
	query := DB.Table("tickets").Joins("LEFT JOIN users ON users.id = tickets.user_id")
	if filter.Status > 0 {
		query = query.Where("tickets.status = ?", filter.Status)
	}
	if filter.Priority > 0 {
		query = query.Where("tickets.priority = ?", filter.Priority)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("tickets.title LIKE ? OR users.username LIKE ? OR users.display_name LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	// See ListTicketsByUser: always return a non-nil collection for API pages.
	tickets := make([]TicketListItem, 0)
	if err := query.Select("tickets.*, users.username, users.display_name").
		Order("tickets.updated_time DESC, tickets.id DESC").Offset(offset).Limit(limit).Scan(&tickets).Error; err != nil {
		return nil, 0, err
	}
	return tickets, total, nil
}

func GetTicketByIDForUser(ticketID, userID int) (*Ticket, []TicketMessage, error) {
	var ticket Ticket
	if err := DB.Where("id = ? AND user_id = ?", ticketID, userID).First(&ticket).Error; err != nil {
		return nil, nil, err
	}
	messages := make([]TicketMessage, 0)
	if err := DB.Where("ticket_id = ?", ticketID).Order("created_time ASC, id ASC").Find(&messages).Error; err != nil {
		return nil, nil, err
	}
	return &ticket, messages, nil
}

func GetTicketByID(ticketID int) (*Ticket, []TicketMessage, error) {
	var ticket Ticket
	if err := DB.First(&ticket, ticketID).Error; err != nil {
		return nil, nil, err
	}
	messages := make([]TicketMessage, 0)
	if err := DB.Where("ticket_id = ?", ticketID).Order("created_time ASC, id ASC").Find(&messages).Error; err != nil {
		return nil, nil, err
	}
	return &ticket, messages, nil
}

func AddTicketMessage(ticketID, ownerID int, message *TicketMessage, nextStatus int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		query := lockForUpdate(tx).Where("id = ?", ticketID)
		if ownerID > 0 {
			query = query.Where("user_id = ?", ownerID)
		}
		var ticket Ticket
		if err := query.First(&ticket).Error; err != nil {
			return err
		}
		if ticket.Status == TicketStatusClosed {
			return ErrTicketClosed
		}

		now := common.GetTimestamp()
		message.TicketId = ticketID
		message.CreatedTime = now
		if err := tx.Create(message).Error; err != nil {
			return err
		}
		return tx.Model(&ticket).Updates(map[string]any{
			"status":          nextStatus,
			"updated_time":    now,
			"last_reply_time": now,
			"last_reply_by":   message.UserId,
		}).Error
	})
}

func CloseTicket(ticketID, ownerID int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var ticket Ticket
		if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", ticketID, ownerID).First(&ticket).Error; err != nil {
			return err
		}
		if ticket.Status == TicketStatusClosed {
			return nil
		}
		now := common.GetTimestamp()
		return tx.Model(&ticket).Updates(map[string]any{
			"status":       TicketStatusClosed,
			"closed_time":  now,
			"updated_time": now,
		}).Error
	})
}

func UpdateTicketByAdmin(ticketID int, status, priority *int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var ticket Ticket
		if err := lockForUpdate(tx).First(&ticket, ticketID).Error; err != nil {
			return err
		}
		updates := map[string]any{"updated_time": common.GetTimestamp()}
		if status != nil {
			updates["status"] = *status
			if *status == TicketStatusClosed {
				updates["closed_time"] = common.GetTimestamp()
			} else {
				updates["closed_time"] = 0
			}
		}
		if priority != nil {
			updates["priority"] = *priority
		}
		return tx.Model(&ticket).Updates(updates).Error
	})
}

func GetTicketOwner(userID int) (string, string, error) {
	var owner struct {
		Username    string
		DisplayName string
	}
	if err := DB.Unscoped().Model(&User{}).Select("username", "display_name").Where("id = ?", userID).First(&owner).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", nil
		}
		return "", "", err
	}
	return owner.Username, owner.DisplayName, nil
}
