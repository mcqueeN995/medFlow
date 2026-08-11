package models

import (
	"time"

	"github.com/google/uuid"
)

type ThreadTag string

const (
	TagStudy        ThreadTag = "study"
	TagDepartment   ThreadTag = "department"
	TagHumor        ThreadTag = "humor"
	TagMarketplace  ThreadTag = "marketplace"
	TagClinicalBase ThreadTag = "clinical_base"
	TagNews         ThreadTag = "news"
	TagHelp         ThreadTag = "help"
	TagOther        ThreadTag = "other"
)

type ReactionTargetType string

const (
	ReactionTargetThread  ReactionTargetType = "thread"
	ReactionTargetComment ReactionTargetType = "comment"
)

// ReactionKind различает обычную эмодзи-реакцию (лайк) от голоса
// up/down за комментарий - обе живут в одной таблице reactions, но
// независимо друг от друга на одной и той же цели (см. миграцию
// 000016_reactions_kind).
type ReactionKind string

const (
	ReactionKindEmoji ReactionKind = "emoji"
	ReactionKindVote  ReactionKind = "vote"
)

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusReviewed  ReportStatus = "reviewed"
	ReportStatusDismissed ReportStatus = "dismissed"
)

// PublicUser - проекция автора треда/комментария, собирается JOIN'ом в
// репозитории (не отдельная таблица), поэтому не имеет своего репозитория.
type PublicUser struct {
	ID           uuid.UUID
	Nickname     string
	University   *University
	Course       *int
	Faculty      *string
	CreatedAt    time.Time
	ThreadsCount int
}

type Thread struct {
	ID            uuid.UUID
	Author        PublicUser
	Title         string
	Content       string
	Tags          []ThreadTag
	ViewsCount    int
	LikesCount    int
	CommentsCount int
	HiddenAt      *time.Time
	HiddenBy      *uuid.UUID
	HiddenReason  *string
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Comment struct {
	ID       uuid.UUID
	ThreadID uuid.UUID
	ParentID *uuid.UUID
	// ReplyToID - конкретный комментарий, которому реально отвечали, если он
	// отличается от ParentID (см. ForumService.CreateComment: дерево
	// ограничено 2 уровнями, ответ на ответ схлопывается к ParentID
	// верхнего уровня, а ReplyToID сохраняет исходный адресат).
	ReplyToID    *uuid.UUID
	Author       PublicUser
	Content      string
	Depth        int
	LikesCount   int
	HiddenAt     *time.Time
	HiddenBy     *uuid.UUID
	HiddenReason *string
	DeletedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Replies      []Comment
}

type Reaction struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TargetType ReactionTargetType
	TargetID   uuid.UUID
	Emoji      string
	Kind       ReactionKind
	CreatedAt  time.Time
}

// VoteSummary - агрегат голосов up/down за одну цель, см.
// ReactionRepo.VoteSummaries. Score = (кол-во up) - (кол-во down);
// MyVote - направление голоса viewerID, если он голосовал ("up"/"down"),
// иначе nil.
type VoteSummary struct {
	Score  int
	MyVote *string
}

type ThreadListFilter struct {
	Tag      *ThreadTag
	AuthorID *uuid.UUID
	// Q - поиск по заголовку/содержимому треда (ILIKE '%q%').
	Q     *string
	Sort  string
	Page  int
	Limit int
}

type Report struct {
	ID             uuid.UUID
	ReporterID     uuid.UUID
	TargetType     string
	TargetID       uuid.UUID
	Reason         string
	Status         ReportStatus
	ReviewedBy     *uuid.UUID
	ReviewedAt     *time.Time
	ResolutionNote *string
	CreatedAt      time.Time
}

type ReportListFilter struct {
	Status     *ReportStatus
	TargetType *string
	Page       int
	Limit      int
}
