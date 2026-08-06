package dto

import (
	"time"

	"github.com/medflow/backend/internal/models"
)

type Pagination struct {
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	Total   int  `json:"total"`
	HasNext bool `json:"has_next"`
}

func NewPagination(page, limit, total int) Pagination {
	return Pagination{
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasNext: page*limit < total,
	}
}

type PublicUser struct {
	ID           string             `json:"id"`
	Nickname     string             `json:"nickname"`
	University   *models.University `json:"university,omitempty"`
	Course       *int               `json:"course,omitempty"`
	Faculty      *string            `json:"faculty,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	ThreadsCount int                `json:"threads_count"`
}

func ToPublicUser(u models.PublicUser) PublicUser {
	return PublicUser{
		ID:           u.ID.String(),
		Nickname:     u.Nickname,
		University:   u.University,
		Course:       u.Course,
		Faculty:      u.Faculty,
		CreatedAt:    u.CreatedAt,
		ThreadsCount: u.ThreadsCount,
	}
}

type Thread struct {
	ID            string             `json:"id"`
	Author        PublicUser         `json:"author"`
	Title         string             `json:"title"`
	Content       string             `json:"content"`
	Tags          []models.ThreadTag `json:"tags"`
	ViewsCount    int                `json:"views_count"`
	LikesCount    int                `json:"likes_count"`
	CommentsCount int                `json:"comments_count"`
	HiddenAt      *time.Time         `json:"hidden_at,omitempty"`
	DeletedAt     *time.Time         `json:"deleted_at,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
}

func ToThread(t *models.Thread) Thread {
	tags := t.Tags
	if tags == nil {
		tags = []models.ThreadTag{}
	}
	return Thread{
		ID:            t.ID.String(),
		Author:        ToPublicUser(t.Author),
		Title:         t.Title,
		Content:       t.Content,
		Tags:          tags,
		ViewsCount:    t.ViewsCount,
		LikesCount:    t.LikesCount,
		CommentsCount: t.CommentsCount,
		HiddenAt:      t.HiddenAt,
		DeletedAt:     t.DeletedAt,
		CreatedAt:     t.CreatedAt,
	}
}

// ThreadListItem - Thread без content, см. описание в openapi.yaml.
type ThreadListItem struct {
	ID            string             `json:"id"`
	Author        PublicUser         `json:"author"`
	Title         string             `json:"title"`
	Tags          []models.ThreadTag `json:"tags"`
	ViewsCount    int                `json:"views_count"`
	LikesCount    int                `json:"likes_count"`
	CommentsCount int                `json:"comments_count"`
	CreatedAt     time.Time          `json:"created_at"`
}

func ToThreadListItem(t *models.Thread) ThreadListItem {
	tags := t.Tags
	if tags == nil {
		tags = []models.ThreadTag{}
	}
	return ThreadListItem{
		ID:            t.ID.String(),
		Author:        ToPublicUser(t.Author),
		Title:         t.Title,
		Tags:          tags,
		ViewsCount:    t.ViewsCount,
		LikesCount:    t.LikesCount,
		CommentsCount: t.CommentsCount,
		CreatedAt:     t.CreatedAt,
	}
}

type CreateThreadRequest struct {
	Title   string             `json:"title" binding:"required,max=500"`
	Content string             `json:"content" binding:"required,max=50000"`
	Tags    []models.ThreadTag `json:"tags,omitempty" binding:"omitempty,max=5"`
}

type UpdateThreadRequest struct {
	Title   *string             `json:"title,omitempty" binding:"omitempty,max=500"`
	Content *string             `json:"content,omitempty" binding:"omitempty,max=50000"`
	Tags    *[]models.ThreadTag `json:"tags,omitempty"`
}

type Comment struct {
	ID         string     `json:"id"`
	Author     PublicUser `json:"author"`
	Content    string     `json:"content"`
	Depth      int        `json:"depth"`
	LikesCount int        `json:"likes_count"`
	HiddenAt   *time.Time `json:"hidden_at,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func ToComment(c *models.Comment) Comment {
	return Comment{
		ID:         c.ID.String(),
		Author:     ToPublicUser(c.Author),
		Content:    c.Content,
		Depth:      c.Depth,
		LikesCount: c.LikesCount,
		HiddenAt:   c.HiddenAt,
		DeletedAt:  c.DeletedAt,
		CreatedAt:  c.CreatedAt,
	}
}

type CommentTree struct {
	Comment
	Replies []Comment `json:"replies"`
}

func ToCommentTree(c *models.Comment) CommentTree {
	replies := make([]Comment, len(c.Replies))
	for i, reply := range c.Replies {
		replies[i] = ToComment(&reply)
	}
	return CommentTree{
		Comment: ToComment(c),
		Replies: replies,
	}
}

type CreateCommentRequest struct {
	Content  string  `json:"content" binding:"required,min=1,max=5000"`
	ParentID *string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=5000"`
}

type ReactionRequest struct {
	Emoji string `json:"emoji" binding:"required,max=10"`
}

type Reaction struct {
	ID         string                    `json:"id"`
	Emoji      string                    `json:"emoji"`
	TargetType models.ReactionTargetType `json:"target_type"`
	TargetID   string                    `json:"target_id"`
	CreatedAt  time.Time                 `json:"created_at"`
}

func ToReaction(r *models.Reaction) Reaction {
	return Reaction{
		ID:         r.ID.String(),
		Emoji:      r.Emoji,
		TargetType: r.TargetType,
		TargetID:   r.TargetID.String(),
		CreatedAt:  r.CreatedAt,
	}
}

type ReportRequest struct {
	Reason string `json:"reason" binding:"required,max=2000"`
}

type Report struct {
	ID             string              `json:"id"`
	ReporterID     string              `json:"reporter_id"`
	TargetType     string              `json:"target_type"`
	TargetID       string              `json:"target_id"`
	Reason         string              `json:"reason"`
	Status         models.ReportStatus `json:"status"`
	ReviewedBy     *string             `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time          `json:"reviewed_at,omitempty"`
	ResolutionNote *string             `json:"resolution_note,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
}

func ToReport(r *models.Report) Report {
	var reviewedBy *string
	if r.ReviewedBy != nil {
		s := r.ReviewedBy.String()
		reviewedBy = &s
	}
	return Report{
		ID:             r.ID.String(),
		ReporterID:     r.ReporterID.String(),
		TargetType:     r.TargetType,
		TargetID:       r.TargetID.String(),
		Reason:         r.Reason,
		Status:         r.Status,
		ReviewedBy:     reviewedBy,
		ReviewedAt:     r.ReviewedAt,
		ResolutionNote: r.ResolutionNote,
		CreatedAt:      r.CreatedAt,
	}
}
