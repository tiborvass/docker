package system

type Time interface {
	// Number of seconds since epoch
	Unix() int64
	// Nanosecond part
	Nanosecond() int
}
