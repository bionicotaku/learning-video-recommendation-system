package service

import "fmt"

type validationError string

func (e validationError) Error() string {
	return string(e)
}

func ValidationError(format string, args ...any) error {
	return validationError(fmt.Sprintf(format, args...))
}

func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(validationError)
	return ok
}
