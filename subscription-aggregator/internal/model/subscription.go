package model

type Subscription struct {
	ID string `json:"id"`

	ServiceName string `json:"service_name"`

	Price int `json:"price"`

	UserID string `json:"user_id"`

	StartDate string `json:"start_date"`

	EndDate *string `json:"end_date,omitempty"`
}
