package service

import (
	"strings"
	"testing"
)

func TestParseCardsJSON_PlainArray(t *testing.T) {
	text := `[{"question":"Q1","answer":"A1"},{"question":"Q2","answer":"A2"}]`
	cards, err := parseCardsJSON(text)
	if err != nil {
		t.Fatalf("parseCardsJSON() error = %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(cards))
	}
	if cards[0].Question != "Q1" || cards[0].Answer != "A1" {
		t.Errorf("cards[0] = %+v, unexpected", cards[0])
	}
}

func TestParseCardsJSON_MarkdownFencedWithLanguage(t *testing.T) {
	text := "```json\n[{\"question\":\"Q1\",\"answer\":\"A1\"}]\n```"
	cards, err := parseCardsJSON(text)
	if err != nil {
		t.Fatalf("parseCardsJSON() error = %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("len(cards) = %d, want 1", len(cards))
	}
}

func TestParseCardsJSON_MarkdownFencedNoLanguage(t *testing.T) {
	text := "```\n[{\"question\":\"Q1\",\"answer\":\"A1\"}]\n```"
	cards, err := parseCardsJSON(text)
	if err != nil {
		t.Fatalf("parseCardsJSON() error = %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("len(cards) = %d, want 1", len(cards))
	}
}

func TestParseCardsJSON_InvalidJSON(t *testing.T) {
	_, err := parseCardsJSON("не JSON вообще")
	if err == nil {
		t.Fatal("parseCardsJSON() error = nil, want non-nil")
	}
}

func TestParseCardsJSON_DropsCardsMissingQuestionOrAnswer(t *testing.T) {
	text := `[{"question":"Q1","answer":"A1"},{"question":"","answer":"A2"},{"question":"Q3","answer":""}]`
	cards, err := parseCardsJSON(text)
	if err != nil {
		t.Fatalf("parseCardsJSON() error = %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("len(cards) = %d, want 1 (only fully-populated card)", len(cards))
	}
}

func TestParseCardsJSON_AllInvalidReturnsError(t *testing.T) {
	text := `[{"question":"","answer":""}]`
	_, err := parseCardsJSON(text)
	if err == nil {
		t.Fatal("parseCardsJSON() error = nil, want error when no valid cards remain")
	}
}

func TestBuildCardsPrompt_ContainsTopicAndChunks(t *testing.T) {
	prompt := buildCardsPrompt("Строение сердца", "medium", 10, nil)
	if !strings.Contains(prompt, "Строение сердца") {
		t.Error("prompt does not mention the topic")
	}
	if !strings.Contains(prompt, "10") {
		t.Error("prompt does not mention the requested count")
	}
}
