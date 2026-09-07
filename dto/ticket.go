package dto

type CreateTicketRequest struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Content  string `json:"content"`
}

type ReplyTicketRequest struct {
	Content string `json:"content"`
}

type AdminUpdateTicketRequest struct {
	Status   *int `json:"status"`
	Priority *int `json:"priority"`
}
