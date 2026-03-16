package gpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	games "google.golang.org/api/games/v1"
)

func TestNewGamesClient_RejectsMissingCredentials(t *testing.T) {
	_, err := NewGamesClient(context.Background(), CredentialInput{})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestGamesClientListAchievementDefinitions_RequiresService(t *testing.T) {
	client := &GamesClient{}
	_, err := client.ListAchievementDefinitions(context.Background(), "", 0, "")
	if err == nil || !strings.Contains(err.Error(), "service is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGamesClientGetLeaderboard_RequiresID(t *testing.T) {
	client := &GamesClient{games: &games.Service{}}
	_, err := client.GetLeaderboard(context.Background(), "  ", "")
	if err == nil || !strings.Contains(err.Error(), "leaderboard id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGamesInfoMappers(t *testing.T) {
	achievement := gamesAchievementInfoFromDefinition(&games.AchievementDefinition{
		Id:               "achievement-1",
		Name:             "Win",
		Description:      "Win once",
		AchievementType:  "STANDARD",
		InitialState:     "REVEALED",
		ExperiencePoints: 25,
		TotalSteps:       0,
	})
	if achievement.ID != "achievement-1" || achievement.Name != "Win" || achievement.ExperiencePoints != 25 {
		t.Fatalf("unexpected achievement mapping: %+v", achievement)
	}

	event := gamesEventInfoFromDefinition(&games.EventDefinition{
		Id:          "event-1",
		DisplayName: "Match Start",
		Description: "Starts a match",
		Visibility:  "REVEALED",
		ChildEvents: []*games.EventChild{{}, {}},
	})
	if event.ID != "event-1" || event.ChildEventCount != 2 {
		t.Fatalf("unexpected event mapping: %+v", event)
	}

	leaderboard := gamesLeaderboardInfoFromResource(&games.Leaderboard{
		Id:    "leaderboard-1",
		Name:  "Top Score",
		Order: "LARGER_IS_BETTER",
	})
	if leaderboard.ID != "leaderboard-1" || leaderboard.Order != "LARGER_IS_BETTER" {
		t.Fatalf("unexpected leaderboard mapping: %+v", leaderboard)
	}
}
