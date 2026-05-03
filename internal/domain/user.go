package domain

import (
	"errors"
	"regexp"

	"github.com/arjunjgowda/rate-limitting/pkg/validator"
)

var (
	// Alphanumeric regex
	alphaNumericRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

// User is the central entity of our system.
type User struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	Password string  `json:"password,omitempty"` // Added missing field
	Balance  float64 `json:"balance"`
	Email    string  `json:"email"`
}

// Validate ensures the user meets the business requirements.
func (u *User) Validate() error {
	if len(u.Username) < 5 {
		return errors.New("username must be at least 5 characters long")
	}

	if !alphaNumericRegex.MatchString(u.Username) {
		return errors.New("username must be alphanumeric")
	}

	// Using the shared validator
	if u.ID != "" && !validator.IsUUID(u.ID) {
		return errors.New("id must be a valid UUID")
	}

	return nil
}

// CanWithdraw is a business rule that belongs in the domain.
func (u *User) CanWithdraw(amount float64) bool {
	return u.Balance >= amount
}

// Deposit adds money to the user's balance.
func (u *User) Deposit(amount float64) {
	u.Balance += amount
}

// Withdraw removes money from the user's balance if they have enough funds.
func (u *User) Withdraw(amount float64) error {
	if !u.CanWithdraw(amount) {
		return errors.New("insufficient balance")
	}
	u.Balance -= amount
	return nil
}
