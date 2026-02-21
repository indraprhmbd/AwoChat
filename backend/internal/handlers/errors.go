package handlers

// Error is a custom error type for handlers
type Error string

func (e Error) Error() string {
	return string(e)
}
