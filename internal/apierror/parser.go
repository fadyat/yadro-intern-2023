package apierror

import "fmt"

type ParseError struct {
	RowNumber int
	UserMsg   string
	BaseErr   error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse row %d: %s", e.RowNumber, e.UserMsg)
}
