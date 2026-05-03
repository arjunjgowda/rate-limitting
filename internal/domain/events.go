package domain

// UserCreatedEvent represents the data payload for the 'user-created' Kafka event.
type UserCreatedEvent struct {
	UserID   string `json:"userID"`
	Username string `json:"username"`
	Event    string `json:"event"`
}

// NewUserCreatedEvent is a factory function to ensure the event is consistently populated.
func NewUserCreatedEvent(userID, username string) UserCreatedEvent {
	return UserCreatedEvent{
		UserID:   userID,
		Username: username,
		Event:    "USER_CREATED",
	}
}
