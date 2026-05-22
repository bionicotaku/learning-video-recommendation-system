package repository

import (
	"context"
	"crypto/rand"
	"fmt"

	apprepo "learning-video-recommendation-system/internal/user/application/repository"
	"learning-video-recommendation-system/internal/user/domain/model"
	usersqlc "learning-video-recommendation-system/internal/user/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedbackWriter struct {
	pool *pgxpool.Pool
}

var _ apprepo.FeedbackWriter = (*FeedbackWriter)(nil)

func NewFeedbackWriter(pool *pgxpool.Pool) *FeedbackWriter {
	return &FeedbackWriter{pool: pool}
}

func (w *FeedbackWriter) SubmitFeedback(ctx context.Context, submission model.FeedbackSubmission) (model.FeedbackSubmissionResult, error) {
	userID, err := stringToUUID(submission.UserID)
	if err != nil {
		return model.FeedbackSubmissionResult{}, err
	}
	clientFeedbackID, err := optionalStringToUUID(submission.ClientFeedbackID)
	if err != nil {
		return model.FeedbackSubmissionResult{}, err
	}
	submissionID, err := newUUID()
	if err != nil {
		return model.FeedbackSubmissionResult{}, err
	}

	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.FeedbackSubmissionResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := usersqlc.New(tx)
	row, err := queries.UpsertFeedbackSubmission(ctx, usersqlc.UpsertFeedbackSubmissionParams{
		ID:               submissionID,
		UserID:           userID,
		ClientFeedbackID: clientFeedbackID,
		Payload:          []byte(submission.Payload),
	})
	if err != nil {
		return model.FeedbackSubmissionResult{}, err
	}

	if row.Inserted {
		for _, image := range submission.Images {
			imageID, err := newUUID()
			if err != nil {
				return model.FeedbackSubmissionResult{}, err
			}
			if err := queries.InsertFeedbackImage(ctx, usersqlc.InsertFeedbackImageParams{
				ID:           imageID,
				SubmissionID: row.ID,
				SortOrder:    image.SortOrder,
				ContentType:  image.ContentType,
				SizeBytes:    image.SizeBytes,
				Sha256:       image.SHA256,
				Width:        image.Width,
				Height:       image.Height,
				ImageData:    image.Data,
			}); err != nil {
				return model.FeedbackSubmissionResult{}, err
			}
		}
	}

	imageCount, err := queries.CountFeedbackImages(ctx, row.ID)
	if err != nil {
		return model.FeedbackSubmissionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.FeedbackSubmissionResult{}, err
	}
	return model.FeedbackSubmissionResult{
		FeedbackID: uuidToString(row.ID),
		ImageCount: imageCount,
		CreatedAt:  timeOrZero(row.CreatedAt),
	}, nil
}

func optionalStringToUUID(value *string) (pgtype.UUID, error) {
	if value == nil || *value == "" {
		return pgtype.UUID{}, nil
	}
	return stringToUUID(*value)
}

func newUUID() (pgtype.UUID, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return pgtype.UUID{}, fmt.Errorf("generate uuid: %w", err)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	var uuid pgtype.UUID
	copy(uuid.Bytes[:], bytes[:])
	uuid.Valid = true
	return uuid, nil
}
