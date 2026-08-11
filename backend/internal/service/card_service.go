package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/pkg/chunker"
	"github.com/medflow/backend/internal/pkg/llm"
	"github.com/medflow/backend/internal/pkg/pdf"
	"github.com/medflow/backend/internal/pkg/sm2"
)

// TaskTypeGenerateCards - имя asynq-задачи, разделяемое CardService.CreateTask
// (энкодит) и internal/worker.CardTaskHandler (обрабатывает).
const TaskTypeGenerateCards = "cards:generate"

type GenerateCardsPayload struct {
	TaskID string `json:"task_id"`
}

const (
	maxActiveTasksPerUser  = 3  // совпадает с MAX_ACTIVE_TASKS в frontend-моке
	estimatedSecondsPerJob = 20 // грубая оценка на одну задачу впереди в очереди
	defaultCardsCount      = 10
	topKChunks             = 8
)

var (
	ErrCardTaskNotFound     = errors.New("card task not found")
	ErrCardNotFound         = errors.New("card not found")
	ErrTooManyActiveTasks   = errors.New("too many active card tasks")
	ErrCardFavoriteNotFound = errors.New("card favorite not found")
	ErrCardRatingNotFound   = errors.New("card rating not found")
)

type CardService struct {
	taskRepo     CardTaskRepository
	cardRepo     CardRepository
	progressRepo CardProgressRepository
	favoriteRepo CardFavoriteRepository
	ratingRepo   CardRatingRepository
	chunkRepo    TextbookChunkRepository
	textbookRepo TextbookRepository
	uploadRepo   UploadRepository
	reportRepo   ReportRepository
	storage      ObjectStorage
	llm          llm.Provider
	embed        llm.Provider
	enqueuer     TaskEnqueuer
	pushNotifier PushNotifier
}

// NewCardService. llmProvider генерирует текст карточек (cfg.LLM.Provider -
// облачный DeepSeek/Qwen/OpenRouter или локальная Ollama). embedProvider
// считает эмбеддинги и должен быть построен через llm.NewOllamaProvider
// независимо от llmProvider - см. llm.ErrEmbedNotSupported: у облачных
// провайдеров нет единого контракта на эмбеддинги, а размерность вектора в
// БД (vector(1024)) зафиксирована под bge-m3 через Ollama.
func NewCardService(
	taskRepo CardTaskRepository,
	cardRepo CardRepository,
	progressRepo CardProgressRepository,
	favoriteRepo CardFavoriteRepository,
	ratingRepo CardRatingRepository,
	chunkRepo TextbookChunkRepository,
	textbookRepo TextbookRepository,
	uploadRepo UploadRepository,
	reportRepo ReportRepository,
	storage ObjectStorage,
	llmProvider llm.Provider,
	embedProvider llm.Provider,
	enqueuer TaskEnqueuer,
	pushNotifier PushNotifier,
) *CardService {
	return &CardService{
		taskRepo: taskRepo, cardRepo: cardRepo, progressRepo: progressRepo,
		favoriteRepo: favoriteRepo, ratingRepo: ratingRepo, chunkRepo: chunkRepo,
		textbookRepo: textbookRepo, uploadRepo: uploadRepo, reportRepo: reportRepo,
		storage: storage, llm: llmProvider, embed: embedProvider, enqueuer: enqueuer, pushNotifier: pushNotifier,
	}
}

// ==================== HTTP-facing ====================

// CreateTask. file_id обязателен всегда (см. openapi.yaml) и указывает на
// PDF, загруженный через POST /upload - именно он парсится и чанкуется,
// независимо от того, задан ли textbook_id. textbook_id, если задан,
// переводит задачу в источник catalog_textbook: карточки и чанки
// привязываются к учебнику (чанки переиспользуются другими задачами по
// этому же учебнику), и включается межпользовательский кэш по
// учебник+тема+сложность+количество (см. computeCacheKey) - у обычной
// user_upload-загрузки кэша нет, она одноразовая и её файл удаляется из S3
// сразу после обработки (см. cleanupTempFile).
func (s *CardService) CreateTask(ctx context.Context, userID uuid.UUID, req dto.CreateCardTaskRequest) (*dto.CardTask, error) {
	active, err := s.taskRepo.CountActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active >= maxActiveTasksPerUser {
		return nil, ErrTooManyActiveTasks
	}

	fileID, err := uuid.Parse(req.FileID)
	if err != nil {
		return nil, ErrPDFUploadNotFound
	}
	upload, err := s.uploadRepo.FindByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, models.ErrUploadNotFound) {
			return nil, ErrPDFUploadNotFound
		}
		return nil, err
	}
	if upload.UploadType != "pdf" {
		return nil, ErrPDFUploadWrongType
	}

	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = models.DifficultyMedium
	}
	cardsCount := req.CardsCount
	if cardsCount == 0 {
		cardsCount = defaultCardsCount
	}

	task := &models.CardTask{
		UserID: userID, SourceType: models.SourceUserUpload, Topic: &req.Topic, Difficulty: difficulty,
		CardsCount: &cardsCount, TempS3Key: &upload.S3Key, TempFileSize: &upload.SizeBytes,
	}

	var cacheKey *string
	if req.TextbookID != nil {
		textbookID, err := uuid.Parse(*req.TextbookID)
		if err != nil {
			return nil, ErrTextbookNotFound
		}
		if _, err := s.textbookRepo.FindByID(ctx, textbookID); err != nil {
			return nil, s.mapTextbookErr(err)
		}
		task.TextbookID = &textbookID
		task.SourceType = models.SourceCatalogTextbook
		key := computeCacheKey(textbookID, req.Topic, difficulty, cardsCount)
		cacheKey = &key
		task.CacheKey = cacheKey
	}

	created, err := s.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, err
	}

	if cacheKey != nil {
		if hit, err := s.tryCacheHit(ctx, userID, *cacheKey, created); err != nil {
			return nil, err
		} else if hit != nil {
			out := dto.ToCardTask(hit)
			return &out, nil
		}
	}

	payload, err := json.Marshal(GenerateCardsPayload{TaskID: created.ID.String()})
	if err != nil {
		return nil, err
	}
	if err := s.enqueuer.Enqueue(TaskTypeGenerateCards, payload); err != nil {
		_ = s.taskRepo.MarkFailed(ctx, created.ID, "failed to enqueue for processing")
		return nil, err
	}

	s.enrichQueueInfo(ctx, created)
	out := dto.ToCardTask(created)
	return &out, nil
}

// tryCacheHit ищет уже готовую задачу с тем же cache_key и, если находит,
// клонирует её карточки под newTask вместо повторного похода в LLM.
// Возвращает nil без ошибки, если кэш-хита нет (обычный путь генерации).
func (s *CardService) tryCacheHit(ctx context.Context, userID uuid.UUID, cacheKey string, newTask *models.CardTask) (*models.CardTask, error) {
	existing, err := s.taskRepo.FindDoneByCacheKey(ctx, cacheKey)
	if err != nil {
		if errors.Is(err, models.ErrCardTaskNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if existing.ID == newTask.ID {
		return nil, nil
	}

	cloned, err := s.cardRepo.CloneForTask(ctx, existing.ID, newTask.ID)
	if err != nil {
		return nil, err
	}
	cardIDs := make([]uuid.UUID, len(cloned))
	for i, c := range cloned {
		cardIDs[i] = c.ID
	}
	if err := s.progressRepo.CreateBatchDefault(ctx, userID, cardIDs); err != nil {
		return nil, err
	}
	if err := s.taskRepo.MarkDone(ctx, newTask.ID, len(cloned)); err != nil {
		return nil, err
	}
	return s.taskRepo.FindByID(ctx, newTask.ID)
}

func (s *CardService) ListTasks(ctx context.Context, userID uuid.UUID, status *models.CardTaskStatus, page, limit int) (*dto.Pagination, []dto.CardTask, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	tasks, total, err := s.taskRepo.List(ctx, models.CardTaskListFilter{UserID: userID, Status: status, Page: page, Limit: limit})
	if err != nil {
		return nil, nil, err
	}

	items := make([]dto.CardTask, len(tasks))
	for i := range tasks {
		s.enrichQueueInfo(ctx, &tasks[i])
		items[i] = dto.ToCardTask(&tasks[i])
	}
	pagination := dto.NewPagination(page, limit, total)
	return &pagination, items, nil
}

func (s *CardService) GetTask(ctx context.Context, userID, taskID uuid.UUID) (*dto.CardTask, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, s.mapTaskErr(err)
	}
	if err := s.authorizeTaskAccess(task, userID); err != nil {
		return nil, err
	}
	s.enrichQueueInfo(ctx, task)
	out := dto.ToCardTask(task)
	return &out, nil
}

func (s *CardService) ListTaskCards(ctx context.Context, userID, taskID uuid.UUID, page, limit int) (*dto.Pagination, []dto.Card, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, nil, s.mapTaskErr(err)
	}
	if err := s.authorizeTaskAccess(task, userID); err != nil {
		return nil, nil, err
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	cards, total, err := s.cardRepo.ListByTask(ctx, taskID, page, limit)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.enrichCards(ctx, userID, cards)
	if err != nil {
		return nil, nil, err
	}
	pagination := dto.NewPagination(page, limit, total)
	return &pagination, items, nil
}

// authorizeTaskAccess - catalog_textbook-задачи (и их карточки) открыты
// любому авторизованному пользователю, т.к. переиспользуются между всеми
// (см. cache_key/CloneForTask); user_upload остаётся приватной, только
// автору - исходный PDF всё равно удаляется после обработки, делиться
// содержимым можно только явным шерингом (см. ShareTask).
func (s *CardService) authorizeTaskAccess(task *models.CardTask, userID uuid.UUID) error {
	if task.SourceType == models.SourceUserUpload && task.UserID != userID {
		return ErrForbidden
	}
	return nil
}

// authorizeCardAccess - то же правило, что и authorizeTaskAccess, но для
// эндпоинтов избранного/рейтинга, которые получают cardID напрямую (не
// taskID) - избранное/рейтинг catalog_textbook-карточки доступны любому,
// user_upload - только автору задачи.
func (s *CardService) authorizeCardAccess(ctx context.Context, userID, cardID uuid.UUID) (*models.Card, error) {
	card, err := s.cardRepo.FindByID(ctx, cardID)
	if err != nil {
		if errors.Is(err, models.ErrCardNotFound) {
			return nil, ErrCardNotFound
		}
		return nil, err
	}
	task, err := s.taskRepo.FindByID(ctx, card.TaskID)
	if err != nil {
		return nil, s.mapTaskErr(err)
	}
	if err := s.authorizeTaskAccess(task, userID); err != nil {
		return nil, err
	}
	return card, nil
}

// enrichCards - батч-обогащение списка карточек избранным/рейтингом одного
// viewerID, три запроса вместо 3*N (см. IsFavoritedBatch/AggregateForCardsBatch/
// MyRatingsBatch) - тот же принцип, что и VoteSummaries в форуме.
func (s *CardService) enrichCards(ctx context.Context, userID uuid.UUID, cards []models.Card) ([]dto.Card, error) {
	cardIDs := make([]uuid.UUID, len(cards))
	for i := range cards {
		cardIDs[i] = cards[i].ID
	}

	favorites, err := s.favoriteRepo.IsFavoritedBatch(ctx, userID, cardIDs)
	if err != nil {
		return nil, err
	}
	ratings, err := s.ratingRepo.AggregateForCardsBatch(ctx, cardIDs)
	if err != nil {
		return nil, err
	}
	myRatings, err := s.ratingRepo.MyRatingsBatch(ctx, userID, cardIDs)
	if err != nil {
		return nil, err
	}

	items := make([]dto.Card, len(cards))
	for i := range cards {
		item := dto.ToCard(&cards[i])
		if favorites[cards[i].ID] {
			v := true
			item.IsFavorite = &v
		}
		if agg, ok := ratings[cards[i].ID]; ok {
			avg, cnt := agg.AverageStars, agg.RatingsCount
			item.AverageStars = &avg
			item.RatingsCount = &cnt
		}
		if my, ok := myRatings[cards[i].ID]; ok {
			item.MyStars = &my
		}
		items[i] = item
	}
	return items, nil
}

func (s *CardService) Review(ctx context.Context, userID uuid.UUID, limit int) ([]dto.ReviewCard, int, error) {
	if limit < 1 || limit > 50 {
		limit = 20
	}
	due, err := s.progressRepo.ListDueForUser(ctx, userID, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.progressRepo.CountDueForUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	items := make([]dto.ReviewCard, len(due))
	for i := range due {
		items[i] = dto.ToReviewCard(&due[i])
	}
	return items, total, nil
}

func (s *CardService) RateCard(ctx context.Context, userID, cardID uuid.UUID, grade int) (*dto.CardProgressInfo, error) {
	progress, err := s.progressRepo.FindByUserAndCard(ctx, userID, cardID)
	if err != nil {
		if errors.Is(err, models.ErrCardProgressNotFound) {
			return nil, ErrCardNotFound
		}
		return nil, err
	}

	updated := sm2.Apply(sm2.Progress{
		EaseFactor: progress.EaseFactor, IntervalDays: progress.IntervalDays, Repetitions: progress.Repetitions,
	}, grade)

	now := time.Now()
	progress.EaseFactor = updated.EaseFactor
	progress.IntervalDays = updated.IntervalDays
	progress.Repetitions = updated.Repetitions
	progress.NextReviewAt = sm2.NextReviewDate(updated.IntervalDays, now)
	progress.LastReviewAt = &now
	progress.LastGrade = &grade

	if err := s.progressRepo.Update(ctx, progress); err != nil {
		return nil, err
	}
	out := dto.ToCardProgressInfo(progress)
	return &out, nil
}

func (s *CardService) ReportCard(ctx context.Context, userID, cardID uuid.UUID, reason string) (*dto.Report, error) {
	if _, err := s.cardRepo.FindByID(ctx, cardID); err != nil {
		if errors.Is(err, models.ErrCardNotFound) {
			return nil, ErrCardNotFound
		}
		return nil, err
	}
	if err := s.cardRepo.IncrementReportCount(ctx, cardID); err != nil {
		return nil, err
	}
	report, err := s.reportRepo.Create(ctx, userID, "card", cardID, reason)
	if err != nil {
		return nil, err
	}
	out := dto.ToReport(report, nil)
	return &out, nil
}

func (s *CardService) Stats(ctx context.Context, userID uuid.UUID) (*dto.CardsStats, error) {
	stats, err := s.progressRepo.StatsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	days, err := s.progressRepo.DistinctReviewDaysForUser(ctx, userID, 366)
	if err != nil {
		return nil, err
	}
	stats.StreakDays = computeStreak(days, time.Now())
	out := dto.ToCardsStats(stats)
	return &out, nil
}

// ==================== Лента каталога ====================

func (s *CardService) ListCatalogFeed(ctx context.Context, q, textbookID *string, difficulty *models.CardDifficulty, page, limit int) (*dto.Pagination, []dto.CardCatalogEntry, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := models.CardCatalogFeedFilter{Q: q, Difficulty: difficulty, Page: page, Limit: limit}
	if textbookID != nil {
		id, err := uuid.Parse(*textbookID)
		if err != nil {
			return nil, nil, ErrTextbookNotFound
		}
		filter.TextbookID = &id
	}

	entries, total, err := s.taskRepo.ListCatalogFeed(ctx, filter)
	if err != nil {
		return nil, nil, err
	}
	items := make([]dto.CardCatalogEntry, len(entries))
	for i := range entries {
		items[i] = dto.ToCardCatalogEntry(&entries[i])
	}
	pagination := dto.NewPagination(page, limit, total)
	return &pagination, items, nil
}

// ==================== Избранное ====================

func (s *CardService) FavoriteCard(ctx context.Context, userID, cardID uuid.UUID) error {
	if _, err := s.authorizeCardAccess(ctx, userID, cardID); err != nil {
		return err
	}
	return s.favoriteRepo.Add(ctx, userID, cardID)
}

func (s *CardService) UnfavoriteCard(ctx context.Context, userID, cardID uuid.UUID) error {
	if err := s.favoriteRepo.Remove(ctx, userID, cardID); err != nil {
		if errors.Is(err, models.ErrCardFavoriteNotFound) {
			return ErrCardFavoriteNotFound
		}
		return err
	}
	return nil
}

func (s *CardService) ListFavorites(ctx context.Context, userID uuid.UUID, page, limit int) (*dto.Pagination, []dto.Card, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	cards, total, err := s.favoriteRepo.ListForUser(ctx, userID, page, limit)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.enrichCards(ctx, userID, cards)
	if err != nil {
		return nil, nil, err
	}
	pagination := dto.NewPagination(page, limit, total)
	return &pagination, items, nil
}

// ReviewFavorites - тот же батч-повтор SM-2, что и Review, но ограничен
// избранными карточками - отдельная колода поверх общего due-списка.
func (s *CardService) ReviewFavorites(ctx context.Context, userID uuid.UUID, limit int) ([]dto.ReviewCard, int, error) {
	if limit < 1 || limit > 50 {
		limit = 20
	}
	due, err := s.progressRepo.ListDueFavoritesForUser(ctx, userID, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.progressRepo.CountDueFavoritesForUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	items := make([]dto.ReviewCard, len(due))
	for i := range due {
		items[i] = dto.ToReviewCard(&due[i])
	}
	return items, total, nil
}

// ==================== Рейтинг звёзд ====================

func (s *CardService) RateCardStars(ctx context.Context, userID, cardID uuid.UUID, stars int) error {
	if _, err := s.authorizeCardAccess(ctx, userID, cardID); err != nil {
		return err
	}
	return s.ratingRepo.Upsert(ctx, userID, cardID, stars)
}

func (s *CardService) RemoveCardRating(ctx context.Context, userID, cardID uuid.UUID) error {
	if err := s.ratingRepo.Delete(ctx, userID, cardID); err != nil {
		if errors.Is(err, models.ErrCardRatingNotFound) {
			return ErrCardRatingNotFound
		}
		return err
	}
	return nil
}

func (s *CardService) ListRatedCards(ctx context.Context, userID uuid.UUID, page, limit int) (*dto.Pagination, []dto.Card, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	cards, total, err := s.ratingRepo.ListRatedByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.enrichCards(ctx, userID, cards)
	if err != nil {
		return nil, nil, err
	}
	pagination := dto.NewPagination(page, limit, total)
	return &pagination, items, nil
}

// ==================== Шеринг ====================

// ShareTask - идемпотентно: повторный вызов на уже расшаренной задаче
// возвращает тот же токен, а не генерирует новый (иначе старые
// разошедшиеся ссылки молча ломались бы). Доступно только владельцу задачи,
// независимо от source_type - в отличие от избранного/рейтинга, шеринг
// личных user_upload-наборов тоже разрешён (это единственный способ ими
// поделиться, раз они не попадают в общую ленту каталога).
func (s *CardService) ShareTask(ctx context.Context, userID, taskID uuid.UUID) (*dto.ShareTaskResponse, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, s.mapTaskErr(err)
	}
	if task.UserID != userID {
		return nil, ErrForbidden
	}
	if task.ShareToken != nil {
		return &dto.ShareTaskResponse{ShareToken: *task.ShareToken}, nil
	}

	token, err := GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}
	if err := s.taskRepo.SetShareToken(ctx, taskID, token); err != nil {
		return nil, err
	}
	return &dto.ShareTaskResponse{ShareToken: token}, nil
}

func (s *CardService) UnshareTask(ctx context.Context, userID, taskID uuid.UUID) error {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return s.mapTaskErr(err)
	}
	if task.UserID != userID {
		return ErrForbidden
	}
	return s.taskRepo.ClearShareToken(ctx, taskID)
}

// GetSharedTask - публичный, без авторизации (см. router.go: вне группы
// cards.Use(AuthRequired)). Отдаёт только задачи в статусе done - угаданный
// или протухший (сгенерённый заново после Unshare/повторного Share) токен
// не должен раскрывать промежуточное состояние pending/processing/failed.
func (s *CardService) GetSharedTask(ctx context.Context, token string) (*dto.SharedCardTask, error) {
	task, err := s.taskRepo.FindByShareToken(ctx, token)
	if err != nil {
		return nil, s.mapTaskErr(err)
	}
	if task.Status != models.CardTaskDone {
		return nil, ErrCardTaskNotFound
	}

	cards, _, err := s.cardRepo.ListByTask(ctx, task.ID, 1, 500)
	if err != nil {
		return nil, err
	}
	cardDTOs := make([]dto.Card, len(cards))
	for i := range cards {
		cardDTOs[i] = dto.ToCard(&cards[i])
	}
	out := &dto.SharedCardTask{Topic: task.Topic, Difficulty: task.Difficulty, Cards: cardDTOs}
	return out, nil
}

// enrichQueueInfo вычисляет position_in_queue/estimated_wait_seconds на
// лету только для pending-задач - см. models.CardTask и план модуля.
func (s *CardService) enrichQueueInfo(ctx context.Context, t *models.CardTask) {
	if t.Status != models.CardTaskPending {
		return
	}
	n, err := s.taskRepo.CountPendingBefore(ctx, t.CreatedAt)
	if err != nil {
		return
	}
	pos := n + 1
	eta := pos * estimatedSecondsPerJob
	t.PositionInQueue = &pos
	t.EstimatedWaitSeconds = &eta
}

// computeStreak - число подряд идущих календарных дней с повторением,
// считая от сегодня (или от вчера, если сегодня ещё не повторяли ни разу) -
// days должны быть отсортированы по убыванию (см. DistinctReviewDaysForUser).
func computeStreak(days []time.Time, now time.Time) int {
	if len(days) == 0 {
		return 0
	}
	cursor := now
	if !sameDay(days[0], cursor) {
		cursor = cursor.AddDate(0, 0, -1)
		if !sameDay(days[0], cursor) {
			return 0
		}
	}
	streak := 0
	for _, d := range days {
		if !sameDay(d, cursor) {
			break
		}
		streak++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return streak
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// computeCacheKey - см. план модуля: одинаковый учебник+тема+сложность+
// количество на разных пользователей должны переиспользовать одну генерацию.
func computeCacheKey(textbookID uuid.UUID, topic string, difficulty models.CardDifficulty, cardsCount int) string {
	norm := strings.ToLower(strings.TrimSpace(topic))
	raw := fmt.Sprintf("%s|%s|%s|%d", textbookID, norm, difficulty, cardsCount)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *CardService) mapTaskErr(err error) error {
	if errors.Is(err, models.ErrCardTaskNotFound) {
		return ErrCardTaskNotFound
	}
	return err
}

func (s *CardService) mapTextbookErr(err error) error {
	if errors.Is(err, models.ErrTextbookNotFound) {
		return ErrTextbookNotFound
	}
	return err
}

// ==================== RAG-конвейер (вызывается из internal/worker) ====================

// ProcessTask выполняет полный конвейер генерации: получить/подготовить
// чанки → найти релевантные по теме → сгенерировать карточки через LLM →
// сохранить. Вызывается asynq-обработчиком, не HTTP-хендлером.
func (s *CardService) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return err
	}

	if err := s.taskRepo.MarkProcessing(ctx, taskID); err != nil {
		return err
	}

	cards, err := s.runPipeline(ctx, task)
	if err != nil {
		if markErr := s.taskRepo.MarkFailed(ctx, taskID, err.Error()); markErr != nil {
			return markErr
		}
		s.notifyTaskOutcome(ctx, task.UserID, false)
		s.cleanupTempFile(ctx, task)
		return nil
	}

	saved, err := s.cardRepo.CreateBatch(ctx, cards)
	if err != nil {
		if markErr := s.taskRepo.MarkFailed(ctx, taskID, err.Error()); markErr != nil {
			return markErr
		}
		s.notifyTaskOutcome(ctx, task.UserID, false)
		s.cleanupTempFile(ctx, task)
		return nil
	}

	cardIDs := make([]uuid.UUID, len(saved))
	for i, c := range saved {
		cardIDs[i] = c.ID
	}
	if err := s.progressRepo.CreateBatchDefault(ctx, task.UserID, cardIDs); err != nil {
		return err
	}
	if err := s.taskRepo.MarkDone(ctx, taskID, len(saved)); err != nil {
		return err
	}
	s.notifyTaskOutcome(ctx, task.UserID, true)

	s.cleanupTempFile(ctx, task)
	return nil
}

// notifyTaskOutcome - лучшее старание, тот же паттерн, что ForumService.notifyReply:
// push-уведомление о завершении/провале генерации карточек не должно влиять
// на исход самой обработки задачи.
func (s *CardService) notifyTaskOutcome(ctx context.Context, userID uuid.UUID, done bool) {
	if done {
		_ = s.pushNotifier.Notify(ctx, userID, models.NotificationCardTaskDone, "Карточки готовы", "Генерация карточек завершена — можно приступать к повторению")
		return
	}
	_ = s.pushNotifier.Notify(ctx, userID, models.NotificationCardTaskFailed, "Не удалось создать карточки", "Генерация карточек завершилась ошибкой, попробуйте ещё раз")
}

func (s *CardService) runPipeline(ctx context.Context, task *models.CardTask) ([]models.Card, error) {
	if err := s.ensureChunks(ctx, task); err != nil {
		return nil, err
	}

	topicEmbedding, err := s.embed.Embed(ctx, *task.Topic)
	if err != nil {
		return nil, fmt.Errorf("embed topic: %w", err)
	}

	nearest, err := s.chunkRepo.SearchNearest(ctx, task.TextbookID, sourceTaskIDFor(task), topicEmbedding, topKChunks)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	if len(nearest) == 0 {
		return nil, errors.New("no relevant material found for this topic")
	}

	cardsWanted := defaultCardsCount
	if task.CardsCount != nil {
		cardsWanted = *task.CardsCount
	}

	prompt := buildCardsPrompt(*task.Topic, task.Difficulty, cardsWanted, nearest)
	parsed, err := s.generateAndParse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	cards := make([]models.Card, len(parsed))
	for i, p := range parsed {
		p := p
		cards[i] = models.Card{
			TaskID: task.ID, TextbookID: task.TextbookID,
			Chapter: nonEmptyPtr(p.Chapter), Topic: nonEmptyPtr(p.Topic), Subtopic: nonEmptyPtr(p.Subtopic),
			Question: p.Question, Answer: p.Answer,
			PageApprox: nonZeroIntPtr(p.PageApprox), SourceReference: nonEmptyPtr(p.SourceReference),
			Difficulty: task.Difficulty,
		}
	}
	return cards, nil
}

// generateAndParse просит LLM строгий JSON и один раз перезапрашивает с
// более жёсткой инструкцией, если ответ не распарсился - модели нередко
// оборачивают JSON в markdown-код или добавляют пояснения.
func (s *CardService) generateAndParse(ctx context.Context, prompt string) ([]llmCard, error) {
	text, err := s.llm.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate cards: %w", err)
	}
	parsed, parseErr := parseCardsJSON(text)
	if parseErr == nil {
		return parsed, nil
	}

	retryPrompt := prompt + "\n\nВАЖНО: ответь СТРОГО валидным JSON-массивом объектов, без markdown-разметки и без пояснений."
	text2, err := s.llm.Generate(ctx, retryPrompt)
	if err != nil {
		return nil, fmt.Errorf("generate cards (retry): %w", err)
	}
	parsed, err = parseCardsJSON(text2)
	if err != nil {
		return nil, fmt.Errorf("llm returned invalid JSON after retry: %w", err)
	}
	return parsed, nil
}

// ensureChunks гарантирует наличие чанков с эмбеддингами для источника
// задачи: для catalog_textbook переиспользует уже посчитанные (если есть),
// для user_upload считает заново каждый раз (одноразовый файл).
func (s *CardService) ensureChunks(ctx context.Context, task *models.CardTask) error {
	if task.SourceType == models.SourceCatalogTextbook {
		exists, err := s.chunkRepo.ExistsForTextbook(ctx, *task.TextbookID)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
	}

	if task.TempS3Key == nil {
		return errors.New("card task has no source file")
	}
	reader, err := s.storage.Get(ctx, *task.TempS3Key)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	pages, err := pdf.ExtractPages(data)
	if err != nil {
		return fmt.Errorf("extract pdf text: %w", err)
	}

	rawChunks := chunker.Split(pages, chunker.DefaultSize, chunker.DefaultOverlap)
	if len(rawChunks) == 0 {
		return errors.New("no extractable text in source file")
	}

	chunks := make([]models.TextbookChunk, len(rawChunks))
	for i, rc := range rawChunks {
		embedding, err := s.embed.Embed(ctx, rc.Content)
		if err != nil {
			return fmt.Errorf("embed chunk %d: %w", i, err)
		}
		pageNumber := rc.PageNumber
		chunks[i] = models.TextbookChunk{
			TextbookID: task.TextbookID, TaskID: sourceTaskIDFor(task),
			ChunkIndex: i, Content: rc.Content, PageNumber: &pageNumber, Embedding: embedding,
		}
	}

	return s.chunkRepo.CreateBatch(ctx, chunks)
}

// sourceTaskIDFor - taskID для textbook_chunks.task_id (только user_upload -
// у catalog_textbook чанки шарятся по textbook_id, task_id остаётся nil).
func sourceTaskIDFor(task *models.CardTask) *uuid.UUID {
	if task.SourceType == models.SourceCatalogTextbook {
		return nil
	}
	return &task.ID
}

// cleanupTempFile удаляет временный PDF user_upload-задачи из S3 сразу после
// обработки (успешной или нет) - обещание пользователю в UI ("файл
// удаляется с сервера сразу после обработки"). Лучшее старание: ошибка
// удаления не должна проваливать уже посчитанный результат задачи.
func (s *CardService) cleanupTempFile(ctx context.Context, task *models.CardTask) {
	if task.SourceType != models.SourceUserUpload || task.TempS3Key == nil {
		return
	}
	_ = s.storage.Remove(ctx, *task.TempS3Key)
}

func nonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// maxPlausiblePage - разумный потолок для "номера страницы" из ответа LLM.
// Локальные модели (в отличие от облачных) иногда галлюцинируют огромные
// или отрицательные значения в page_approx, которые не влезают в Postgres
// int4 (max ~2.1 млрд) и валят весь INSERT карточек - такие тоже трактуем
// как "не знаю", а не падаем на записи в БД.
const maxPlausiblePage = 100_000

func nonZeroIntPtr(n int) *int {
	if n <= 0 || n > maxPlausiblePage {
		return nil
	}
	return &n
}
