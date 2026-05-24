package wordfavorites

import "time"

import catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"

type wordFavoriteIdentityRequest struct {
	CoarseUnitID  *int64  `json:"coarse_unit_id"`
	Text          string  `json:"text"`
	Source        string  `json:"source"`
	VideoID       *string `json:"video_id"`
	SentenceIndex *int32  `json:"sentence_index"`
	TokenIndex    *int32  `json:"token_index"`
	OccurredAt    string  `json:"occurred_at"`
}

type wordFavoriteStatusRequest struct {
	CoarseUnitID        *int64  `json:"coarse_unit_id"`
	Text                string  `json:"text"`
	Source              string  `json:"source"`
	VideoID             *string `json:"video_id"`
	SentenceIndex       *int32  `json:"sentence_index"`
	TokenIndex          *int32  `json:"token_index"`
	IncludeVideoContext bool    `json:"include_video_context"`
}

func (request wordFavoriteIdentityRequest) setDTO(userID string, occurredAt time.Time) catalogdto.SetWordFavoriteRequest {
	return catalogdto.SetWordFavoriteRequest{
		UserID:        userID,
		CoarseUnitID:  request.CoarseUnitID,
		Text:          request.Text,
		Source:        request.Source,
		VideoID:       request.VideoID,
		SentenceIndex: request.SentenceIndex,
		TokenIndex:    request.TokenIndex,
		OccurredAt:    occurredAt,
	}
}

func (request wordFavoriteIdentityRequest) unsetDTO(userID string, occurredAt time.Time) catalogdto.UnsetWordFavoriteRequest {
	return catalogdto.UnsetWordFavoriteRequest{
		UserID:        userID,
		CoarseUnitID:  request.CoarseUnitID,
		Text:          request.Text,
		Source:        request.Source,
		VideoID:       request.VideoID,
		SentenceIndex: request.SentenceIndex,
		TokenIndex:    request.TokenIndex,
		OccurredAt:    occurredAt,
	}
}

func (request wordFavoriteStatusRequest) dto(userID string) catalogdto.GetWordFavoriteStatusRequest {
	return catalogdto.GetWordFavoriteStatusRequest{
		UserID:          userID,
		CoarseUnitID:    request.CoarseUnitID,
		Text:            request.Text,
		Source:          request.Source,
		VideoID:         request.VideoID,
		SentenceIndex:   request.SentenceIndex,
		TokenIndex:      request.TokenIndex,
		IncludeVideoCtx: request.IncludeVideoContext,
	}
}
