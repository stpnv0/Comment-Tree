package domain

import "errors"

var (
	// ErrCommentNotFound is returned when the requested comment does not exist.
	ErrCommentNotFound = errors.New("comment not found")

	// ErrParentNotFound is returned when the specified parent comment does not exist.
	ErrParentNotFound = errors.New("parent comment not found")

	// ErrInvalidInput is returned when request input fails validation.
	ErrInvalidInput = errors.New("invalid input")
)
