package me

import (
	"bytes"
	"encoding/json"
	"net/http"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
)

var allowedUpdateProfileFields = map[string]struct{}{
	"display_name":    {},
	"birth_date":      {},
	"gender":          {},
	"education_stage": {},
	"timezone":        {},
}

func (h *Handler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := request.RequireJSONContentType(r); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	command, err := parseUpdateProfileRequest(r, principal.UserID)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	result, err := h.updateProfile.Execute(r.Context(), command)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func parseUpdateProfileRequest(r *http.Request, userID string) (userdto.UpdateMeProfileRequest, error) {
	var body map[string]json.RawMessage
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		return userdto.UpdateMeProfileRequest{}, invalidRequest(err)
	}
	if len(body) == 0 {
		return userdto.UpdateMeProfileRequest{}, apiservice.InvalidRequestError("profile patch must contain at least one field")
	}

	command := userdto.UpdateMeProfileRequest{UserID: userID}
	for field, raw := range body {
		if _, ok := allowedUpdateProfileFields[field]; !ok {
			return userdto.UpdateMeProfileRequest{}, apiservice.InvalidRequestError("unexpected field: " + field)
		}
		switch field {
		case "display_name":
			value, err := parseRequiredStringField(field, raw)
			if err != nil {
				return userdto.UpdateMeProfileRequest{}, err
			}
			command.SetDisplayName = true
			command.DisplayName = value
		case "birth_date":
			value, err := parseNullableStringField(field, raw)
			if err != nil {
				return userdto.UpdateMeProfileRequest{}, err
			}
			command.SetBirthDate = true
			command.BirthDate = value
		case "gender":
			value, err := parseNullableStringField(field, raw)
			if err != nil {
				return userdto.UpdateMeProfileRequest{}, err
			}
			command.SetGender = true
			command.Gender = value
		case "education_stage":
			value, err := parseNullableStringField(field, raw)
			if err != nil {
				return userdto.UpdateMeProfileRequest{}, err
			}
			command.SetEducationStage = true
			command.EducationStage = value
		case "timezone":
			value, err := parseNullableStringField(field, raw)
			if err != nil {
				return userdto.UpdateMeProfileRequest{}, err
			}
			command.SetTimezone = true
			command.Timezone = value
		}
	}
	return command, nil
}

func parseRequiredStringField(field string, raw json.RawMessage) (string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", apiservice.InvalidRequestError(field + " must not be null")
	}
	value, err := parseString(raw)
	if err != nil {
		return "", apiservice.InvalidRequestError(field + " must be a string")
	}
	return value, nil
}

func parseNullableStringField(field string, raw json.RawMessage) (*string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	value, err := parseString(raw)
	if err != nil {
		return nil, apiservice.InvalidRequestError(field + " must be a string or null")
	}
	return &value, nil
}

func parseString(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}

func invalidRequest(err error) error {
	if err == nil {
		return nil
	}
	return httperror.InvalidRequest(err)
}
