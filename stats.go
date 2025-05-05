package main

import (
	"sync"
)

// constants
const MB = 1024 * 1024 // 1 MB

// Stats maintains the aggregate file transfer statistics
// across all workers.
type Stats struct {
	TotalBytes int64      // Total number of bytes copied
	MBytes     float64    // Total number of megabytes copied
	mutex      sync.Mutex // Mutex to protect concurrent access
}

// NewStats initializes a new Stats instance with the current time
func NewStats() *Stats {
	return &Stats{}
}

// AddBytes adds the number of bytes copied to the total and
// updates the megabytes copied. Access to these shared stats
// is protected.
func (s *Stats) AddBytes(n int64) {
	// lock the mutex
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// update shared stats
	s.TotalBytes += n
	s.MBytes = float64(s.TotalBytes) / MB
}
