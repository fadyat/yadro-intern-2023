package apierror

import (
	"errors"
	"fmt"
	"regexp"
)

type ValidationError struct {
	RowNumber int
	UserMsg   string
}

type ValidationFn[T any] func(T) error

func NotMoreThen(i, j int) error {
	if i <= j {
		return nil
	}

	return errors.New(ErrValueTooBig)
}

func MoreThenZero(i int) error {
	if i > 0 {
		return nil
	}

	return errors.New(ErrValueMustBeMoreThanZero)
}

func ValidateName(name string) error {
	rgx := regexp.MustCompile(`^[a-z0-9_]+$`)
	if !rgx.MatchString(name) {
		return errors.New(ErrClientDataInvalidName)
	}

	return nil
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error at row %d: %s", e.RowNumber, e.UserMsg)
}
