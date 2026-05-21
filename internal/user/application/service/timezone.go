package service

import (
	"strings"
	"time"

	"learning-video-recommendation-system/internal/user/domain/model"
)

func validTimezone(value string) (string, *time.Location, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, false
	}
	location, err := time.LoadLocation(value)
	if err != nil {
		return "", nil, false
	}
	return value, location, true
}

func resolveTimezone(clientTimezone string, profileTimezone *string) (string, *time.Location) {
	if value, location, ok := validTimezone(clientTimezone); ok {
		return value, location
	}
	if profileTimezone != nil {
		if value, location, ok := validTimezone(*profileTimezone); ok {
			return value, location
		}
	}
	location, _ := time.LoadLocation(model.DefaultTimezone)
	return model.DefaultTimezone, location
}
