package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

func setupTestCardService(
	taskRepo *mockCardTaskRepository,
	cardRepo *mockCardRepository,
	progressRepo *mockCardProgressRepository,
	chunkRepo *mockTextbookChunkRepository,
	textbookRepo *mockTextbookRepository,
	uploadRepo *mockUploadRepository,
	reportRepo *mockReportRepository,
	storage *mockObjectStorage,
	llmProvider *mockLLMProvider,
	enqueuer *mockTaskEnqueuer,
) *CardService {
	return setupTestCardServiceFull(taskRepo, cardRepo, progressRepo, nil, nil, chunkRepo, textbookRepo, uploadRepo, reportRepo, storage, llmProvider, enqueuer)
}

func setupTestCardServiceFull(
	taskRepo *mockCardTaskRepository,
	cardRepo *mockCardRepository,
	progressRepo *mockCardProgressRepository,
	favoriteRepo *mockCardFavoriteRepository,
	ratingRepo *mockCardRatingRepository,
	chunkRepo *mockTextbookChunkRepository,
	textbookRepo *mockTextbookRepository,
	uploadRepo *mockUploadRepository,
	reportRepo *mockReportRepository,
	storage *mockObjectStorage,
	llmProvider *mockLLMProvider,
	enqueuer *mockTaskEnqueuer,
) *CardService {
	if taskRepo == nil {
		taskRepo = &mockCardTaskRepository{}
	}
	if cardRepo == nil {
		cardRepo = &mockCardRepository{}
	}
	if progressRepo == nil {
		progressRepo = &mockCardProgressRepository{}
	}
	if favoriteRepo == nil {
		favoriteRepo = &mockCardFavoriteRepository{}
	}
	if ratingRepo == nil {
		ratingRepo = &mockCardRatingRepository{}
	}
	if chunkRepo == nil {
		chunkRepo = &mockTextbookChunkRepository{}
	}
	if textbookRepo == nil {
		textbookRepo = &mockTextbookRepository{}
	}
	if uploadRepo == nil {
		uploadRepo = &mockUploadRepository{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Upload, error) {
				return &models.Upload{ID: id, UploadType: "pdf", S3Key: "uploads/pdf/x.pdf"}, nil
			},
		}
	}
	if reportRepo == nil {
		reportRepo = &mockReportRepository{}
	}
	if storage == nil {
		storage = &mockObjectStorage{}
	}
	if llmProvider == nil {
		llmProvider = &mockLLMProvider{}
	}
	if enqueuer == nil {
		enqueuer = &mockTaskEnqueuer{}
	}
	return NewCardService(taskRepo, cardRepo, progressRepo, favoriteRepo, ratingRepo, chunkRepo, textbookRepo, uploadRepo, reportRepo, storage, llmProvider, llmProvider, enqueuer, &mockPushNotifier{})
}

// ==================== CreateTask ====================

func TestCardService_CreateTask_RateLimited(t *testing.T) {
	taskRepo := &mockCardTaskRepository{
		countActiveFn: func(ctx context.Context, userID uuid.UUID) (int, error) { return 3, nil },
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.CreateTask(context.Background(), uuid.New(), dto.CreateCardTaskRequest{FileID: uuid.NewString(), Topic: "тема"})
	if !errors.Is(err, ErrTooManyActiveTasks) {
		t.Fatalf("CreateTask() error = %v, want ErrTooManyActiveTasks", err)
	}
}

func TestCardService_CreateTask_UploadNotFound(t *testing.T) {
	uploadRepo := &mockUploadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Upload, error) { return nil, models.ErrUploadNotFound },
	}
	svc := setupTestCardService(nil, nil, nil, nil, nil, uploadRepo, nil, nil, nil, nil)

	_, err := svc.CreateTask(context.Background(), uuid.New(), dto.CreateCardTaskRequest{FileID: uuid.NewString(), Topic: "тема"})
	if !errors.Is(err, ErrPDFUploadNotFound) {
		t.Fatalf("CreateTask() error = %v, want ErrPDFUploadNotFound", err)
	}
}

func TestCardService_CreateTask_UploadWrongType(t *testing.T) {
	uploadRepo := &mockUploadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Upload, error) {
			return &models.Upload{ID: id, UploadType: "image", S3Key: "uploads/image/x.png"}, nil
		},
	}
	svc := setupTestCardService(nil, nil, nil, nil, nil, uploadRepo, nil, nil, nil, nil)

	_, err := svc.CreateTask(context.Background(), uuid.New(), dto.CreateCardTaskRequest{FileID: uuid.NewString(), Topic: "тема"})
	if !errors.Is(err, ErrPDFUploadWrongType) {
		t.Fatalf("CreateTask() error = %v, want ErrPDFUploadWrongType", err)
	}
}

func TestCardService_CreateTask_UserUpload_EnqueuesAndStaysPending(t *testing.T) {
	var enqueuedType string
	enqueuer := &mockTaskEnqueuer{
		enqueueFn: func(typename string, payload []byte) error {
			enqueuedType = typename
			return nil
		},
	}
	svc := setupTestCardService(nil, nil, nil, nil, nil, nil, nil, nil, nil, enqueuer)

	task, err := svc.CreateTask(context.Background(), uuid.New(), dto.CreateCardTaskRequest{FileID: uuid.NewString(), Topic: "Строение сердца"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if task.Status != models.CardTaskPending {
		t.Errorf("Status = %v, want pending", task.Status)
	}
	if enqueuedType != TaskTypeGenerateCards {
		t.Errorf("enqueued type = %q, want %q", enqueuedType, TaskTypeGenerateCards)
	}
}

func TestCardService_CreateTask_CatalogTextbook_CacheMiss_EnqueuesWithCacheKey(t *testing.T) {
	var createdTask *models.CardTask
	var enqueued bool
	textbookID := uuid.New()

	taskRepo := &mockCardTaskRepository{
		createFn: func(ctx context.Context, t *models.CardTask) (*models.CardTask, error) {
			cp := *t
			cp.ID = uuid.New()
			cp.Status = models.CardTaskPending
			createdTask = &cp
			return &cp, nil
		},
		findDoneByCacheKeyFn: func(ctx context.Context, cacheKey string) (*models.CardTask, error) {
			return nil, models.ErrCardTaskNotFound
		},
	}
	textbookRepo := &mockTextbookRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Textbook, error) {
			return &models.Textbook{ID: id}, nil
		},
	}
	enqueuer := &mockTaskEnqueuer{enqueueFn: func(typename string, payload []byte) error { enqueued = true; return nil }}

	svc := setupTestCardService(taskRepo, nil, nil, nil, textbookRepo, nil, nil, nil, nil, enqueuer)

	tid := textbookID.String()
	_, err := svc.CreateTask(context.Background(), uuid.New(), dto.CreateCardTaskRequest{
		FileID: uuid.NewString(), TextbookID: &tid, Topic: "Тема", CardsCount: 5,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if !enqueued {
		t.Error("expected task to be enqueued on cache miss")
	}
	if createdTask.CacheKey == nil {
		t.Fatal("expected cache_key to be set for catalog_textbook source")
	}
	if createdTask.SourceType != models.SourceCatalogTextbook {
		t.Errorf("SourceType = %v, want catalog_textbook", createdTask.SourceType)
	}
}

func TestCardService_CreateTask_CatalogTextbook_CacheHit_ClonesAndSkipsEnqueue(t *testing.T) {
	textbookID := uuid.New()
	existingTaskID := uuid.New()
	var newTaskID uuid.UUID
	var enqueued bool
	var markedDoneCount int
	var progressCreatedFor []uuid.UUID
	clonedCards := []models.Card{{ID: uuid.New()}, {ID: uuid.New()}}

	taskRepo := &mockCardTaskRepository{
		createFn: func(ctx context.Context, t *models.CardTask) (*models.CardTask, error) {
			cp := *t
			cp.ID = uuid.New()
			cp.Status = models.CardTaskPending
			newTaskID = cp.ID
			return &cp, nil
		},
		findDoneByCacheKeyFn: func(ctx context.Context, cacheKey string) (*models.CardTask, error) {
			return &models.CardTask{ID: existingTaskID, Status: models.CardTaskDone}, nil
		},
		markDoneFn: func(ctx context.Context, id uuid.UUID, cardsCount int) error {
			markedDoneCount = cardsCount
			return nil
		},
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: id, Status: models.CardTaskDone, CardsCount: &markedDoneCount}, nil
		},
	}
	cardRepo := &mockCardRepository{
		cloneForTaskFn: func(ctx context.Context, sourceTaskID, newTaskID uuid.UUID) ([]models.Card, error) {
			if sourceTaskID != existingTaskID {
				t.Errorf("CloneForTask sourceTaskID = %v, want %v", sourceTaskID, existingTaskID)
			}
			return clonedCards, nil
		},
	}
	progressRepo := &mockCardProgressRepository{
		createBatchDefaultFn: func(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) error {
			progressCreatedFor = cardIDs
			return nil
		},
	}
	textbookRepo := &mockTextbookRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Textbook, error) {
			return &models.Textbook{ID: id}, nil
		},
	}
	enqueuer := &mockTaskEnqueuer{enqueueFn: func(typename string, payload []byte) error { enqueued = true; return nil }}

	svc := setupTestCardService(taskRepo, cardRepo, progressRepo, nil, textbookRepo, nil, nil, nil, nil, enqueuer)

	tid := textbookID.String()
	task, err := svc.CreateTask(context.Background(), uuid.New(), dto.CreateCardTaskRequest{
		FileID: uuid.NewString(), TextbookID: &tid, Topic: "Тема",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if enqueued {
		t.Error("cache-hit must not enqueue a new generation job")
	}
	if markedDoneCount != len(clonedCards) {
		t.Errorf("MarkDone cardsCount = %d, want %d", markedDoneCount, len(clonedCards))
	}
	if len(progressCreatedFor) != len(clonedCards) {
		t.Errorf("progress created for %d cards, want %d", len(progressCreatedFor), len(clonedCards))
	}
	if task.Status != models.CardTaskDone {
		t.Errorf("Status = %v, want done (immediate cache-hit completion)", task.Status)
	}
	_ = newTaskID
}

// ==================== RateCard / ReportCard ====================

func TestCardService_RateCard_AppliesSM2AndPersists(t *testing.T) {
	userID, cardID := uuid.New(), uuid.New()
	var saved *models.CardProgress

	progressRepo := &mockCardProgressRepository{
		findByUserAndCardFn: func(ctx context.Context, gotUser, gotCard uuid.UUID) (*models.CardProgress, error) {
			return &models.CardProgress{ID: uuid.New(), UserID: userID, CardID: cardID, EaseFactor: 2.5, IntervalDays: 0, Repetitions: 0}, nil
		},
		updateFn: func(ctx context.Context, p *models.CardProgress) error {
			saved = p
			return nil
		},
	}
	svc := setupTestCardService(nil, nil, progressRepo, nil, nil, nil, nil, nil, nil, nil)

	info, err := svc.RateCard(context.Background(), userID, cardID, 3)
	if err != nil {
		t.Fatalf("RateCard() error = %v", err)
	}
	if info.EaseFactor < 2.59 || info.EaseFactor > 2.61 {
		t.Errorf("EaseFactor = %v, want ~2.6", info.EaseFactor)
	}
	if info.IntervalDays != 1 || info.Repetitions != 1 {
		t.Errorf("IntervalDays/Repetitions = %d/%d, want 1/1", info.IntervalDays, info.Repetitions)
	}
	if saved == nil || saved.LastGrade == nil || *saved.LastGrade != 3 {
		t.Fatalf("persisted progress = %+v, want LastGrade=3", saved)
	}
}

func TestCardService_RateCard_NotFound(t *testing.T) {
	progressRepo := &mockCardProgressRepository{
		findByUserAndCardFn: func(ctx context.Context, userID, cardID uuid.UUID) (*models.CardProgress, error) {
			return nil, models.ErrCardProgressNotFound
		},
	}
	svc := setupTestCardService(nil, nil, progressRepo, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.RateCard(context.Background(), uuid.New(), uuid.New(), 2)
	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("RateCard() error = %v, want ErrCardNotFound", err)
	}
}

func TestCardService_ReportCard_IncrementsAndCreatesReport(t *testing.T) {
	cardID := uuid.New()
	var incremented bool
	var reportTargetType string

	cardRepo := &mockCardRepository{
		findByIDFn:             func(ctx context.Context, id uuid.UUID) (*models.Card, error) { return &models.Card{ID: id}, nil },
		incrementReportCountFn: func(ctx context.Context, id uuid.UUID) error { incremented = true; return nil },
	}
	reportRepo := &mockReportRepository{
		createFn: func(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*models.Report, error) {
			reportTargetType = targetType
			return &models.Report{ID: uuid.New(), TargetType: targetType, TargetID: targetID, Reason: reason}, nil
		},
	}
	svc := setupTestCardService(nil, cardRepo, nil, nil, nil, nil, reportRepo, nil, nil, nil)

	_, err := svc.ReportCard(context.Background(), uuid.New(), cardID, "неверный ответ")
	if err != nil {
		t.Fatalf("ReportCard() error = %v", err)
	}
	if !incremented {
		t.Error("expected IncrementReportCount to be called")
	}
	if reportTargetType != "card" {
		t.Errorf("report target_type = %q, want card", reportTargetType)
	}
}

// ==================== ownership ====================

func TestCardService_GetTask_ForbiddenForNonOwner_UserUpload(t *testing.T) {
	owner := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: id, UserID: owner, SourceType: models.SourceUserUpload, Status: models.CardTaskDone}, nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetTask(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("GetTask() error = %v, want ErrForbidden (личная user_upload-задача чужого пользователя)", err)
	}
}

// TestCardService_GetTask_AllowedForNonOwner_CatalogTextbook - ослабление
// владения: catalog_textbook-задачи переиспользуются между пользователями
// (см. cache_key/CloneForTask), поэтому доступны любому авторизованному, не
// только автору.
func TestCardService_GetTask_AllowedForNonOwner_CatalogTextbook(t *testing.T) {
	owner := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: id, UserID: owner, SourceType: models.SourceCatalogTextbook, Status: models.CardTaskDone}, nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetTask(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("GetTask() error = %v, want nil (catalog_textbook доступна не только автору)", err)
	}
}

// ==================== ListTaskCards / enrichCards ====================

func TestCardService_ListTaskCards_ForbiddenForNonOwner_UserUpload(t *testing.T) {
	owner := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: id, UserID: owner, SourceType: models.SourceUserUpload, Status: models.CardTaskDone}, nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, _, err := svc.ListTaskCards(context.Background(), uuid.New(), uuid.New(), 1, 50)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ListTaskCards() error = %v, want ErrForbidden", err)
	}
}

func TestCardService_ListTaskCards_EnrichesWithFavoriteAndRating(t *testing.T) {
	cardID := uuid.New()
	taskID := uuid.New()
	viewerID := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: taskID, UserID: uuid.New(), SourceType: models.SourceCatalogTextbook, Status: models.CardTaskDone}, nil
		},
	}
	cardRepo := &mockCardRepository{
		listByTaskFn: func(ctx context.Context, tID uuid.UUID, page, limit int) ([]models.Card, int, error) {
			return []models.Card{{ID: cardID, TaskID: taskID, Question: "q", Answer: "a"}}, 1, nil
		},
	}
	favoriteRepo := &mockCardFavoriteRepository{
		isFavoritedBatchFn: func(ctx context.Context, uID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{cardID: true}, nil
		},
	}
	ratingRepo := &mockCardRatingRepository{
		aggregateForCardsBatchFn: func(ctx context.Context, cardIDs []uuid.UUID) (map[uuid.UUID]models.CardRatingAggregate, error) {
			return map[uuid.UUID]models.CardRatingAggregate{cardID: {AverageStars: 4.5, RatingsCount: 2}}, nil
		},
		myRatingsBatchFn: func(ctx context.Context, uID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]int, error) {
			return map[uuid.UUID]int{cardID: 5}, nil
		},
	}
	svc := setupTestCardServiceFull(taskRepo, cardRepo, nil, favoriteRepo, ratingRepo, nil, nil, nil, nil, nil, nil, nil)

	_, items, err := svc.ListTaskCards(context.Background(), viewerID, taskID, 1, 50)
	if err != nil {
		t.Fatalf("ListTaskCards() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	item := items[0]
	if item.IsFavorite == nil || !*item.IsFavorite {
		t.Errorf("IsFavorite = %v, want true", item.IsFavorite)
	}
	if item.AverageStars == nil || *item.AverageStars != 4.5 {
		t.Errorf("AverageStars = %v, want 4.5", item.AverageStars)
	}
	if item.RatingsCount == nil || *item.RatingsCount != 2 {
		t.Errorf("RatingsCount = %v, want 2", item.RatingsCount)
	}
	if item.MyStars == nil || *item.MyStars != 5 {
		t.Errorf("MyStars = %v, want 5", item.MyStars)
	}
}

// ==================== Избранное ====================

func TestCardService_FavoriteCard_ForbiddenForNonOwner_UserUpload(t *testing.T) {
	owner := uuid.New()
	taskID := uuid.New()
	cardID := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: taskID, UserID: owner, SourceType: models.SourceUserUpload, Status: models.CardTaskDone}, nil
		},
	}
	cardRepo := &mockCardRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Card, error) {
			return &models.Card{ID: cardID, TaskID: taskID}, nil
		},
	}
	svc := setupTestCardServiceFull(taskRepo, cardRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	err := svc.FavoriteCard(context.Background(), uuid.New(), cardID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("FavoriteCard() error = %v, want ErrForbidden", err)
	}
}

func TestCardService_FavoriteCard_Success_CatalogTextbook(t *testing.T) {
	taskID := uuid.New()
	cardID := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: taskID, UserID: uuid.New(), SourceType: models.SourceCatalogTextbook, Status: models.CardTaskDone}, nil
		},
	}
	cardRepo := &mockCardRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Card, error) {
			return &models.Card{ID: cardID, TaskID: taskID}, nil
		},
	}
	var addedUser, addedCard uuid.UUID
	favoriteRepo := &mockCardFavoriteRepository{
		addFn: func(ctx context.Context, userID, cID uuid.UUID) error {
			addedUser, addedCard = userID, cID
			return nil
		},
	}
	svc := setupTestCardServiceFull(taskRepo, cardRepo, nil, favoriteRepo, nil, nil, nil, nil, nil, nil, nil, nil)

	viewerID := uuid.New()
	if err := svc.FavoriteCard(context.Background(), viewerID, cardID); err != nil {
		t.Fatalf("FavoriteCard() error = %v", err)
	}
	if addedUser != viewerID || addedCard != cardID {
		t.Errorf("Add() called with (%v, %v), want (%v, %v)", addedUser, addedCard, viewerID, cardID)
	}
}

func TestCardService_UnfavoriteCard_NotFound(t *testing.T) {
	favoriteRepo := &mockCardFavoriteRepository{
		removeFn: func(ctx context.Context, userID, cardID uuid.UUID) error {
			return models.ErrCardFavoriteNotFound
		},
	}
	svc := setupTestCardServiceFull(nil, nil, nil, favoriteRepo, nil, nil, nil, nil, nil, nil, nil, nil)

	err := svc.UnfavoriteCard(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrCardFavoriteNotFound) {
		t.Fatalf("UnfavoriteCard() error = %v, want ErrCardFavoriteNotFound", err)
	}
}

// ==================== Рейтинг звёзд ====================

func TestCardService_RateCardStars_ForbiddenForNonOwner_UserUpload(t *testing.T) {
	owner := uuid.New()
	taskID := uuid.New()
	cardID := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: taskID, UserID: owner, SourceType: models.SourceUserUpload, Status: models.CardTaskDone}, nil
		},
	}
	cardRepo := &mockCardRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Card, error) {
			return &models.Card{ID: cardID, TaskID: taskID}, nil
		},
	}
	svc := setupTestCardServiceFull(taskRepo, cardRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	err := svc.RateCardStars(context.Background(), uuid.New(), cardID, 5)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("RateCardStars() error = %v, want ErrForbidden", err)
	}
}

func TestCardService_RateCardStars_Success(t *testing.T) {
	taskID := uuid.New()
	cardID := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: taskID, UserID: uuid.New(), SourceType: models.SourceCatalogTextbook, Status: models.CardTaskDone}, nil
		},
	}
	cardRepo := &mockCardRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Card, error) {
			return &models.Card{ID: cardID, TaskID: taskID}, nil
		},
	}
	var gotStars int
	ratingRepo := &mockCardRatingRepository{
		upsertFn: func(ctx context.Context, userID, cID uuid.UUID, stars int) error {
			gotStars = stars
			return nil
		},
	}
	svc := setupTestCardServiceFull(taskRepo, cardRepo, nil, nil, ratingRepo, nil, nil, nil, nil, nil, nil, nil)

	if err := svc.RateCardStars(context.Background(), uuid.New(), cardID, 4); err != nil {
		t.Fatalf("RateCardStars() error = %v", err)
	}
	if gotStars != 4 {
		t.Errorf("Upsert() called with stars = %d, want 4", gotStars)
	}
}

func TestCardService_RemoveCardRating_NotFound(t *testing.T) {
	ratingRepo := &mockCardRatingRepository{
		deleteFn: func(ctx context.Context, userID, cardID uuid.UUID) error {
			return models.ErrCardRatingNotFound
		},
	}
	svc := setupTestCardServiceFull(nil, nil, nil, nil, ratingRepo, nil, nil, nil, nil, nil, nil, nil)

	err := svc.RemoveCardRating(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrCardRatingNotFound) {
		t.Fatalf("RemoveCardRating() error = %v, want ErrCardRatingNotFound", err)
	}
}

// ==================== Шеринг ====================

func TestCardService_ShareTask_ForbiddenForNonOwner(t *testing.T) {
	owner := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: id, UserID: owner, Status: models.CardTaskDone}, nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.ShareTask(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ShareTask() error = %v, want ErrForbidden", err)
	}
}

func TestCardService_ShareTask_GeneratesNewToken(t *testing.T) {
	owner := uuid.New()
	taskID := uuid.New()
	var gotToken string
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: taskID, UserID: owner, Status: models.CardTaskDone}, nil
		},
		setShareTokenFn: func(ctx context.Context, tID uuid.UUID, token string) error {
			gotToken = token
			return nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	resp, err := svc.ShareTask(context.Background(), owner, taskID)
	if err != nil {
		t.Fatalf("ShareTask() error = %v", err)
	}
	if resp.ShareToken == "" || resp.ShareToken != gotToken {
		t.Errorf("ShareTask() token = %q, SetShareToken() got %q", resp.ShareToken, gotToken)
	}
}

func TestCardService_ShareTask_Idempotent_ReturnsExistingToken(t *testing.T) {
	owner := uuid.New()
	taskID := uuid.New()
	existing := "already-shared-token"
	setCalled := false
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: taskID, UserID: owner, Status: models.CardTaskDone, ShareToken: &existing}, nil
		},
		setShareTokenFn: func(ctx context.Context, tID uuid.UUID, token string) error {
			setCalled = true
			return nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	resp, err := svc.ShareTask(context.Background(), owner, taskID)
	if err != nil {
		t.Fatalf("ShareTask() error = %v", err)
	}
	if resp.ShareToken != existing {
		t.Errorf("ShareTask() token = %q, want existing %q (идемпотентно)", resp.ShareToken, existing)
	}
	if setCalled {
		t.Error("SetShareToken() should not be called when a token already exists")
	}
}

func TestCardService_UnshareTask_ForbiddenForNonOwner(t *testing.T) {
	owner := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: id, UserID: owner, Status: models.CardTaskDone}, nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	err := svc.UnshareTask(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("UnshareTask() error = %v, want ErrForbidden", err)
	}
}

func TestCardService_GetSharedTask_NotDone_ReturnsNotFound(t *testing.T) {
	taskRepo := &mockCardTaskRepository{
		findByShareTokenFn: func(ctx context.Context, token string) (*models.CardTask, error) {
			return &models.CardTask{ID: uuid.New(), Status: models.CardTaskProcessing}, nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetSharedTask(context.Background(), "some-token")
	if !errors.Is(err, ErrCardTaskNotFound) {
		t.Fatalf("GetSharedTask() error = %v, want ErrCardTaskNotFound (промежуточный статус не должен раскрываться)", err)
	}
}

func TestCardService_GetSharedTask_Done_ReturnsCards(t *testing.T) {
	taskID := uuid.New()
	topic := "anatomy"
	taskRepo := &mockCardTaskRepository{
		findByShareTokenFn: func(ctx context.Context, token string) (*models.CardTask, error) {
			return &models.CardTask{ID: taskID, Status: models.CardTaskDone, Topic: &topic, Difficulty: models.DifficultyMedium}, nil
		},
	}
	cardRepo := &mockCardRepository{
		listByTaskFn: func(ctx context.Context, tID uuid.UUID, page, limit int) ([]models.Card, int, error) {
			return []models.Card{{ID: uuid.New(), TaskID: taskID, Question: "q", Answer: "a"}}, 1, nil
		},
	}
	svc := setupTestCardServiceFull(taskRepo, cardRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	resp, err := svc.GetSharedTask(context.Background(), "some-token")
	if err != nil {
		t.Fatalf("GetSharedTask() error = %v", err)
	}
	if len(resp.Cards) != 1 {
		t.Fatalf("len(resp.Cards) = %d, want 1", len(resp.Cards))
	}
	if resp.Topic == nil || *resp.Topic != topic {
		t.Errorf("Topic = %v, want %q", resp.Topic, topic)
	}
}

// ==================== Лента каталога ====================

func TestCardService_ListCatalogFeed_InvalidTextbookID(t *testing.T) {
	svc := setupTestCardService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	badID := "not-a-uuid"
	_, _, err := svc.ListCatalogFeed(context.Background(), nil, &badID, nil, 1, 20)
	if !errors.Is(err, ErrTextbookNotFound) {
		t.Fatalf("ListCatalogFeed() error = %v, want ErrTextbookNotFound", err)
	}
}

func TestCardService_ListCatalogFeed_Success(t *testing.T) {
	taskRepo := &mockCardTaskRepository{
		listCatalogFeedFn: func(ctx context.Context, f models.CardCatalogFeedFilter) ([]models.CardCatalogEntry, int, error) {
			return []models.CardCatalogEntry{{TaskID: uuid.New(), TextbookTitle: "Anatomy 101"}}, 1, nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, items, err := svc.ListCatalogFeed(context.Background(), nil, nil, nil, 1, 20)
	if err != nil {
		t.Fatalf("ListCatalogFeed() error = %v", err)
	}
	if len(items) != 1 || items[0].TextbookTitle != "Anatomy 101" {
		t.Fatalf("items = %+v, want single entry Anatomy 101", items)
	}
}

// ==================== streak ====================

func TestComputeStreak(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	day := func(offset int) time.Time { return now.AddDate(0, 0, offset) }

	tests := []struct {
		name string
		days []time.Time
		want int
	}{
		{"no days", nil, 0},
		{"today only", []time.Time{day(0)}, 1},
		{"three consecutive ending today", []time.Time{day(0), day(-1), day(-2)}, 3},
		{"gap breaks streak", []time.Time{day(0), day(-3)}, 1},
		{"no review today, starts from yesterday", []time.Time{day(-1), day(-2)}, 2},
		{"only two days ago - no active streak", []time.Time{day(-2)}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeStreak(tt.days, now)
			if got != tt.want {
				t.Errorf("computeStreak(%v) = %d, want %d", tt.days, got, tt.want)
			}
		})
	}
}

// ==================== ProcessTask ====================

func TestCardService_ProcessTask_Success_CatalogTextbookReusesChunks(t *testing.T) {
	textbookID := uuid.New()
	topic := "Строение сердца"
	cardsCount := 2
	task := &models.CardTask{
		ID: uuid.New(), UserID: uuid.New(), TextbookID: &textbookID, SourceType: models.SourceCatalogTextbook,
		Topic: &topic, Difficulty: models.DifficultyMedium, CardsCount: &cardsCount, Status: models.CardTaskPending,
	}

	var processingMarked, doneMarked bool
	var doneCount int
	taskRepo := &mockCardTaskRepository{
		findByIDFn:       func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) { return task, nil },
		markProcessingFn: func(ctx context.Context, id uuid.UUID) error { processingMarked = true; return nil },
		markDoneFn: func(ctx context.Context, id uuid.UUID, cardsCount int) error {
			doneMarked = true
			doneCount = cardsCount
			return nil
		},
	}
	chunkRepo := &mockTextbookChunkRepository{
		existsForTextbookFn: func(ctx context.Context, id uuid.UUID) (bool, error) { return true, nil }, // уже есть - парсить PDF не нужно
		searchNearestFn: func(ctx context.Context, textbookID, taskID *uuid.UUID, embedding []float32, topK int) ([]models.TextbookChunk, error) {
			return []models.TextbookChunk{{Content: "фрагмент про сердце", PageNumber: intPtr(3)}}, nil
		},
	}
	llmProvider := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return `[{"question":"Сколько камер в сердце?","answer":"Четыре","page_approx":3},{"question":"Q2","answer":"A2"}]`, nil
		},
	}
	var savedCards []models.Card
	cardRepo := &mockCardRepository{
		createBatchFn: func(ctx context.Context, cards []models.Card) ([]models.Card, error) {
			out := make([]models.Card, len(cards))
			for i, c := range cards {
				c.ID = uuid.New()
				out[i] = c
			}
			savedCards = out
			return out, nil
		},
	}
	var progressForUser uuid.UUID
	progressRepo := &mockCardProgressRepository{
		createBatchDefaultFn: func(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) error {
			progressForUser = userID
			return nil
		},
	}

	svc := setupTestCardService(taskRepo, cardRepo, progressRepo, chunkRepo, nil, nil, nil, nil, llmProvider, nil)

	if err := svc.ProcessTask(context.Background(), task.ID); err != nil {
		t.Fatalf("ProcessTask() error = %v", err)
	}
	if !processingMarked {
		t.Error("expected MarkProcessing to be called")
	}
	if !doneMarked || doneCount != 2 {
		t.Errorf("MarkDone called=%v count=%d, want true/2", doneMarked, doneCount)
	}
	if len(savedCards) != 2 {
		t.Fatalf("saved %d cards, want 2", len(savedCards))
	}
	if savedCards[0].Difficulty != models.DifficultyMedium {
		t.Errorf("card difficulty = %v, want task difficulty medium (not LLM-provided)", savedCards[0].Difficulty)
	}
	if progressForUser != task.UserID {
		t.Errorf("progress created for user %v, want %v", progressForUser, task.UserID)
	}
}

func TestCardService_ProcessTask_LLMFailure_MarksFailedAndReturnsNil(t *testing.T) {
	textbookID := uuid.New()
	topic := "тема"
	task := &models.CardTask{
		ID: uuid.New(), UserID: uuid.New(), TextbookID: &textbookID, SourceType: models.SourceCatalogTextbook,
		Topic: &topic, Difficulty: models.DifficultyMedium, Status: models.CardTaskPending,
	}

	var failedMsg string
	taskRepo := &mockCardTaskRepository{
		findByIDFn:       func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) { return task, nil },
		markProcessingFn: func(ctx context.Context, id uuid.UUID) error { return nil },
		markFailedFn: func(ctx context.Context, id uuid.UUID, errMsg string) error {
			failedMsg = errMsg
			return nil
		},
	}
	chunkRepo := &mockTextbookChunkRepository{
		existsForTextbookFn: func(ctx context.Context, id uuid.UUID) (bool, error) { return true, nil },
		searchNearestFn: func(ctx context.Context, textbookID, taskID *uuid.UUID, embedding []float32, topK int) ([]models.TextbookChunk, error) {
			return []models.TextbookChunk{{Content: "фрагмент"}}, nil
		},
	}
	llmProvider := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "", fmt.Errorf("provider unavailable")
		},
	}

	svc := setupTestCardService(taskRepo, nil, nil, chunkRepo, nil, nil, nil, nil, llmProvider, nil)

	err := svc.ProcessTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("ProcessTask() error = %v, want nil (failure recorded on the task, not propagated to asynq for retry)", err)
	}
	if failedMsg == "" {
		t.Error("expected MarkFailed to be called with a non-empty error message")
	}
}

func TestCardService_ProcessTask_LLMFailure_NotifiesCardTaskFailed(t *testing.T) {
	textbookID := uuid.New()
	topic := "тема"
	userID := uuid.New()
	task := &models.CardTask{
		ID: uuid.New(), UserID: userID, TextbookID: &textbookID, SourceType: models.SourceCatalogTextbook,
		Topic: &topic, Difficulty: models.DifficultyMedium, Status: models.CardTaskPending,
	}

	taskRepo := &mockCardTaskRepository{
		findByIDFn:       func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) { return task, nil },
		markProcessingFn: func(ctx context.Context, id uuid.UUID) error { return nil },
		markFailedFn:     func(ctx context.Context, id uuid.UUID, errMsg string) error { return nil },
	}
	chunkRepo := &mockTextbookChunkRepository{
		existsForTextbookFn: func(ctx context.Context, id uuid.UUID) (bool, error) { return true, nil },
		searchNearestFn: func(ctx context.Context, textbookID, taskID *uuid.UUID, embedding []float32, topK int) ([]models.TextbookChunk, error) {
			return []models.TextbookChunk{{Content: "фрагмент"}}, nil
		},
	}
	llmProvider := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "", fmt.Errorf("provider unavailable")
		},
	}
	notifier := &mockPushNotifier{}
	svc := NewCardService(taskRepo, &mockCardRepository{}, &mockCardProgressRepository{}, &mockCardFavoriteRepository{}, &mockCardRatingRepository{}, chunkRepo, &mockTextbookRepository{},
		&mockUploadRepository{findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Upload, error) {
			return &models.Upload{ID: id, UploadType: "pdf", S3Key: "uploads/pdf/x.pdf"}, nil
		}}, &mockReportRepository{}, &mockObjectStorage{}, llmProvider, llmProvider, &mockTaskEnqueuer{}, notifier)

	if err := svc.ProcessTask(context.Background(), task.ID); err != nil {
		t.Fatalf("ProcessTask() error = %v", err)
	}
	if len(notifier.calls) != 1 || notifier.calls[0] != models.NotificationCardTaskFailed {
		t.Fatalf("notifier.calls = %v, want [card_task_failed]", notifier.calls)
	}
}

func TestCardService_ProcessTask_UserUpload_ParsesAndEmbedsPDF(t *testing.T) {
	topic := "тема"
	cardsCount := 1
	pdfKey := "uploads/pdf/source.pdf"
	task := &models.CardTask{
		ID: uuid.New(), UserID: uuid.New(), SourceType: models.SourceUserUpload,
		Topic: &topic, Difficulty: models.DifficultyEasy, CardsCount: &cardsCount,
		TempS3Key: &pdfKey, Status: models.CardTaskPending,
	}

	taskRepo := &mockCardTaskRepository{
		findByIDFn:       func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) { return task, nil },
		markProcessingFn: func(ctx context.Context, id uuid.UUID) error { return nil },
		markDoneFn:       func(ctx context.Context, id uuid.UUID, cardsCount int) error { return nil },
	}
	var chunksStored []models.TextbookChunk
	chunkRepo := &mockTextbookChunkRepository{
		createBatchFn: func(ctx context.Context, chunks []models.TextbookChunk) error {
			chunksStored = chunks
			return nil
		},
		searchNearestFn: func(ctx context.Context, textbookID, taskID *uuid.UUID, embedding []float32, topK int) ([]models.TextbookChunk, error) {
			return []models.TextbookChunk{{Content: "любой фрагмент"}}, nil
		},
	}
	var removedKey string
	storage := &mockObjectStorage{
		getFn: func(ctx context.Context, key string) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(buildMinimalPDF(t, "Sample text"))), nil
		},
		removeFn: func(ctx context.Context, key string) error { removedKey = key; return nil },
	}
	llmProvider := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return `[{"question":"Q","answer":"A"}]`, nil
		},
	}
	cardRepo := &mockCardRepository{}
	progressRepo := &mockCardProgressRepository{}

	svc := setupTestCardService(taskRepo, cardRepo, progressRepo, chunkRepo, nil, nil, nil, storage, llmProvider, nil)

	if err := svc.ProcessTask(context.Background(), task.ID); err != nil {
		t.Fatalf("ProcessTask() error = %v", err)
	}
	if len(chunksStored) == 0 {
		t.Fatal("expected chunks to be parsed from PDF and stored")
	}
	if chunksStored[0].TaskID == nil || *chunksStored[0].TaskID != task.ID {
		t.Errorf("chunk TaskID = %v, want %v (user_upload chunks scoped by task, not textbook)", chunksStored[0].TaskID, task.ID)
	}
	if removedKey != pdfKey {
		t.Errorf("removed key = %q, want %q (temp file must be deleted after processing)", removedKey, pdfKey)
	}
}

func intPtr(n int) *int { return &n }

// buildMinimalPDF - минимальный однострочный PDF с текстом, для проверки
// ensureChunks (Get -> ExtractPages -> Split -> Embed -> CreateBatch).
func buildMinimalPDF(t *testing.T, text string) []byte {
	t.Helper()
	var buf bytes.Buffer
	var offsets []int
	writeObj := func(body string) {
		offsets = append(offsets, buf.Len())
		buf.WriteString(body)
	}
	buf.WriteString("%PDF-1.4\n")
	writeObj("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	writeObj("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	writeObj("3 0 obj\n<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 4 0 R >> >> /MediaBox [0 0 300 300] /Contents 5 0 R >>\nendobj\n")
	writeObj("4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")
	content := fmt.Sprintf("BT /F1 18 Tf 10 100 Td (%s) Tj ET", text)
	writeObj(fmt.Sprintf("5 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(content), content))
	xrefStart := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(offsets)+1))
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(offsets)+1, xrefStart))
	return buf.Bytes()
}
