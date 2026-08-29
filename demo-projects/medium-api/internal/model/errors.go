package model

import "errors"

var (
	// ErrCustomerNotFound is returned when a customer record cannot be located.
	ErrCustomerNotFound = errors.New("customer not found")

	// ErrOrderNotFound is returned when an order record cannot be located.
	ErrOrderNotFound = errors.New("order not found")

	// ErrDuplicateEmail is returned when attempting to register an existing email.
	ErrDuplicateEmail = errors.New("customer with this email already exists")

	// ErrInvalidStatusTransition is returned when an order state transition violates the state machine.
	ErrInvalidStatusTransition = errors.New("invalid order status transition")

	// ErrOrderCannotBeCancelled is returned when an order in terminal state (SHIPPED/DELIVERED/CANCELLED) is attempted to be cancelled.
	ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled in its current state")

	// ErrEmptyOrderItems is returned when creating an order with zero items.
	ErrEmptyOrderItems = errors.New("order must have at least one item")

	// ErrUnauthorized is returned when authentication is missing or invalid.
	ErrUnauthorized = errors.New("unauthorized: missing or invalid authorization token")

	// ErrInvalidCredentials is returned on bad login credentials.
	ErrInvalidCredentials = errors.New("invalid email or password")
)
