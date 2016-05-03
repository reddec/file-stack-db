package api

import "time"

type err string

func (e err) Error() string { return string(e) }

// Commond non-system error
const (
	ErrSectionNotFound = err("Section not found")
	ErrStackIsEmpty    = err("Section is empty")
)

// Message represenation in stack
type Message struct {
	Headers map[string]string // Headers are decoded to JSON (may be changed in future)
	Body    []byte
}

// PushArgs - arguments for PUSH operation
type PushArgs struct {
	Message        // Message content
	Section string // Stack name
}

// DataResult - result of PUSH and PEAK operation
type DataResult struct {
	Message        // Message content
	DepthIndex int // Current stack depth (before operation) - non-atomic op.
}

// Section (stack) basic info
type Section struct {
	Name       string
	Depth      int
	LastAccess time.Time
}

// Service API
type Service interface {
	Sections(prefix string, result *[]Section) error
	Push(msg PushArgs, resultDepthIndex *int) error
	Peak(section string, result *DataResult) error
	Pop(section string, result *DataResult) error
}
