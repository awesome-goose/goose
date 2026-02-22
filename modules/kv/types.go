package kv

import "errors"

// Common errors
var (
	ErrKeyNotFound = errors.New("key not found")
	ErrKeyExpired  = errors.New("key expired")
	ErrKeyExists   = errors.New("key already exists")
)

// Status constants
const (
	StatusActive  = "active"
	StatusDeleted = "deleted"
)

// DefaultGroup is used when no group is specified
const DefaultGroup = "default"
