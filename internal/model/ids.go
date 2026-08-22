package model

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

var idSequence uint64

func NewID(prefix string) string {
	value := atomic.AddUint64(&idSequence, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), value)
}

func CleanID(value string) string {
	return strings.TrimSpace(value)
}

func IsID(value string) bool {
	value = CleanID(value)
	return value != "" && len(value) <= 120 && !strings.ContainsAny(value, " /\\\t\r\n")
}

func EnsureTime(value time.Time) time.Time {
	return value.UTC().Truncate(time.Millisecond)
}
