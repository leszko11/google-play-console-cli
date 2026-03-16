package games

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/config"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakeClient struct {
	achievements gpc.GamesAchievementsListInfo
	events       gpc.GamesEventsListInfo
	leaderboards gpc.GamesLeaderboardsListInfo
	leaderboard  gpc.GamesLeaderboardInfo
}

func (f *fakeClient) ListAchievementDefinitions(_ context.Context, _ string, _ int64, _ string) (gpc.GamesAchievementsListInfo, error) {
	return f.achievements, nil
}

func (f *fakeClient) ListEventDefinitions(_ context.Context, _ string, _ int64, _ string) (gpc.GamesEventsListInfo, error) {
	return f.events, nil
}

func (f *fakeClient) ListLeaderboards(_ context.Context, _ string, _ int64, _ string) (gpc.GamesLeaderboardsListInfo, error) {
	return f.leaderboards, nil
}

func (f *fakeClient) GetLeaderboard(_ context.Context, _, _ string) (gpc.GamesLeaderboardInfo, error) {
	return f.leaderboard, nil
}

func defaultConfig() config.Config {
	return config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {ServiceAccountPath: "/tmp/sa.json"},
		},
	}
}

func runGames(t *testing.T, deps Deps, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = &bytes.Buffer{}
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) string {
			if key == "GPC_BYPASS_KEYCHAIN" {
				return "1"
			}
			return ""
		}
	}
	cmd := NewCommand(deps)
	err := cmd.ParseAndRun(context.Background(), args)
	return out.String(), err
}

func TestAchievementsListJSON(t *testing.T) {
	client := &fakeClient{
		achievements: gpc.GamesAchievementsListInfo{
			Achievements:  []gpc.GamesAchievementInfo{{ID: "achievement-1", Name: "Win", AchievementType: "STANDARD", InitialState: "REVEALED", ExperiencePoints: 25}},
			NextPageToken: "next-achievements",
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runGames(t, deps, "achievements", "list", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"id":"achievement-1"`, `"nextPageToken":"next-achievements"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestEventsListMarkdown(t *testing.T) {
	client := &fakeClient{
		events: gpc.GamesEventsListInfo{
			Events: []gpc.GamesEventInfo{{ID: "event-1", DisplayName: "Match Start", Visibility: "REVEALED", ChildEventCount: 2}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runGames(t, deps, "events", "list", "--output", "markdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"| id | displayName | visibility | childEvents |",
		"| event-1 | Match Start | REVEALED | 2 |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestLeaderboardsListTable(t *testing.T) {
	client := &fakeClient{
		leaderboards: gpc.GamesLeaderboardsListInfo{
			Leaderboards: []gpc.GamesLeaderboardInfo{{ID: "leaderboard-1", Name: "Top Score", Order: "LARGER_IS_BETTER"}},
		},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runGames(t, deps, "leaderboards", "list", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "ID\tNAME\tORDER") || !strings.Contains(out, "leaderboard-1\tTop Score\tLARGER_IS_BETTER") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestLeaderboardsGetRequiresID(t *testing.T) {
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return &fakeClient{}, nil },
	}

	_, err := runGames(t, deps, "leaderboards", "get")
	if err == nil || !strings.Contains(err.Error(), "--leaderboard-id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLeaderboardsGetMarkdown(t *testing.T) {
	client := &fakeClient{
		leaderboard: gpc.GamesLeaderboardInfo{ID: "leaderboard-1", Name: "Top Score", Order: "LARGER_IS_BETTER"},
	}
	deps := Deps{
		LoadConfig: func() (config.Config, error) { return defaultConfig(), nil },
		NewClient:  func(context.Context, gpc.CredentialInput) (Client, error) { return client, nil },
	}

	out, err := runGames(t, deps, "leaderboards", "get", "--leaderboard-id", "leaderboard-1", "--output", "markdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"| field | value |",
		"| id | leaderboard-1 |",
		"| order | LARGER_IS_BETTER |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}
