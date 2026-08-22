package clock

import "time"

type Clock interface {
	Now() time.Time
}

type System struct{}

func (System) Now() time.Time {
	return time.Now().UTC()
}

type Fixed struct {
	Value time.Time
}

func (c Fixed) Now() time.Time {
	return c.Value.UTC()
}

func UTC(value time.Time) time.Time {
	return value.UTC().Truncate(time.Millisecond)
}
