package auth

import "errors"

var ErrMissingPrincipal = errors.New("trusted principal is required")
