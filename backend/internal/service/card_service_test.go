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
	if taskRepo == nil {
		taskRepo = &mockCardTaskRepository{}
	}
	if cardRepo == nil {
		cardRepo = &mockCardRepository{}
	}
	if progressRepo == nil {
		progressRepo = &mockCardProgressRepository{}
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
	return NewCardService(taskRepo, cardRepo, progressRepo, chunkRepo, textbookRepo, uploadRepo, reportRepo, storage, llmProvider, llmProvider, enqueuer, &mockPushNotifier{})
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

func TestCardService_GetTask_ForbiddenForNonOwner(t *testing.T) {
	owner := uuid.New()
	taskRepo := &mockCardTaskRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
			return &models.CardTask{ID: id, UserID: owner, Status: models.CardTaskDone}, nil
		},
	}
	svc := setupTestCardService(taskRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetTask(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("GetTask() error = %v, want ErrForbidden", err)
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
	svc := NewCardService(taskRepo, &mockCardRepository{}, &mockCardProgressRepository{}, chunkRepo, &mockTextbookRepository{},
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
