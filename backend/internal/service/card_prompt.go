package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/medflow/backend/internal/models"
)

// llmCard - ожидаемая форма одной карточки в JSON-ответе LLM. difficulty
// сюда намеренно не входит - вся генерация задачи получает единый
// difficulty из запроса пользователя (см. runPipeline), самооценке модели
// не доверяем.
type llmCard struct {
	Chapter         string `json:"chapter"`
	Topic           string `json:"topic"`
	Subtopic        string `json:"subtopic"`
	Question        string `json:"question"`
	Answer          string `json:"answer"`
	PageApprox      int    `json:"page_approx"`
	SourceReference string `json:"source_reference"`
}

func difficultyLabel(d models.CardDifficulty) string {
	switch d {
	case models.DifficultyEasy:
		return "лёгкая"
	case models.DifficultyHard:
		return "сложная"
	default:
		return "средняя"
	}
}

// buildCardsPrompt формирует промпт с RAG-контекстом: строгая инструкция на
// JSON-массив + релевантные фрагменты учебника, найденные векторным поиском.
func buildCardsPrompt(topic string, difficulty models.CardDifficulty, count int, chunks []models.TextbookChunk) string {
	var sb strings.Builder
	sb.WriteString("Ты — ассистент, который готовит учебные карточки для студентов-медиков на основе приведённого материала.\n")
	fmt.Fprintf(&sb, "Составь ровно %d карточек по теме «%s», уровень сложности вопросов: %s.\n", count, topic, difficultyLabel(difficulty))
	sb.WriteString("Используй только факты из приведённых ниже фрагментов учебника. Каждая карточка - один чёткий вопрос и точный ответ.\n")
	sb.WriteString("Ответь СТРОГО валидным JSON-массивом объектов, без markdown-разметки, без пояснений до или после, в формате:\n")
	sb.WriteString(`[{"chapter":"","topic":"","subtopic":"","question":"","answer":"","page_approx":0,"source_reference":""}]` + "\n")
	sb.WriteString("\nФрагменты учебника:\n")
	for i, c := range chunks {
		fmt.Fprintf(&sb, "\n--- Фрагмент %d (стр. %d) ---\n%s\n", i+1, c.PageNumber, c.Content)
	}
	return sb.String()
}

// parseCardsJSON разбирает ответ LLM в список карточек, снимая markdown-код-
// заборы (```json ... ```), которыми модели нередко оборачивают JSON вопреки
// инструкции. Карточки без вопроса/ответа отбрасываются как невалидные.
func parseCardsJSON(text string) ([]llmCard, error) {
	text = stripMarkdownFence(text)

	var cards []llmCard
	if err := json.Unmarshal([]byte(text), &cards); err != nil {
		return nil, fmt.Errorf("unmarshal llm response: %w", err)
	}

	valid := make([]llmCard, 0, len(cards))
	for _, c := range cards {
		if strings.TrimSpace(c.Question) == "" || strings.TrimSpace(c.Answer) == "" {
			continue
		}
		valid = append(valid, c)
	}
	if len(valid) == 0 {
		return nil, errors.New("llm response contained no valid cards")
	}
	return valid, nil
}

func stripMarkdownFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	if nl := strings.IndexByte(s, '\n'); nl != -1 && nl < 10 {
		// первая строка после открывающего забора - язык (json/JSON) либо пусто
		s = s[nl+1:]
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}
