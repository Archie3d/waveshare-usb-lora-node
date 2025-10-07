package types

type TimeoutError struct{}

func (e *TimeoutError) Error() string {
	return "timeout"
}

type BusyError struct{}

func (e *BusyError) Error() string {
	return "busy"
}
