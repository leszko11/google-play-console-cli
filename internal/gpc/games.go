package gpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/games/v1"
	"google.golang.org/api/option"
)

type GamesAchievementInfo struct {
	ID               string `json:"id"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	AchievementType  string `json:"achievementType,omitempty"`
	InitialState     string `json:"initialState,omitempty"`
	ExperiencePoints int64  `json:"experiencePoints,omitempty"`
	TotalSteps       int64  `json:"totalSteps,omitempty"`
}

type GamesAchievementsListInfo struct {
	Achievements  []GamesAchievementInfo `json:"achievements,omitempty"`
	NextPageToken string                 `json:"nextPageToken,omitempty"`
}

type GamesEventInfo struct {
	ID              string `json:"id"`
	DisplayName     string `json:"displayName,omitempty"`
	Description     string `json:"description,omitempty"`
	Visibility      string `json:"visibility,omitempty"`
	ChildEventCount int    `json:"childEventCount,omitempty"`
}

type GamesEventsListInfo struct {
	Events        []GamesEventInfo `json:"events,omitempty"`
	NextPageToken string           `json:"nextPageToken,omitempty"`
}

type GamesLeaderboardInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Order string `json:"order,omitempty"`
}

type GamesLeaderboardsListInfo struct {
	Leaderboards  []GamesLeaderboardInfo `json:"leaderboards,omitempty"`
	NextPageToken string                 `json:"nextPageToken,omitempty"`
}

type GamesClient struct {
	games *games.Service
}

func NewGamesClient(ctx context.Context, creds CredentialInput, opts ...option.ClientOption) (*GamesClient, error) {
	if strings.TrimSpace(creds.ServiceAccountPath) == "" && len(creds.ServiceAccountJSON) == 0 {
		return nil, ErrInvalidCredentials
	}

	clientOpts := make([]option.ClientOption, 0, 3+len(opts))
	if strings.TrimSpace(creds.ServiceAccountPath) != "" {
		clientOpts = append(clientOpts, option.WithCredentialsFile(creds.ServiceAccountPath))
	}
	if len(creds.ServiceAccountJSON) > 0 {
		clientOpts = append(clientOpts, option.WithCredentialsJSON(creds.ServiceAccountJSON))
	}
	clientOpts = append(clientOpts, option.WithScopes(games.AndroidpublisherScope))
	clientOpts = append(clientOpts, opts...)

	svc, err := games.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, err
	}

	return &GamesClient{games: svc}, nil
}

func (c *GamesClient) ListAchievementDefinitions(ctx context.Context, language string, pageSize int64, pageToken string) (GamesAchievementsListInfo, error) {
	if c == nil || c.games == nil {
		return GamesAchievementsListInfo{}, errors.New("games service is not configured")
	}
	if pageSize < 0 {
		return GamesAchievementsListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}

	call := c.games.AchievementDefinitions.List().Context(ctx)
	if pageSize > 0 {
		call.MaxResults(pageSize)
	}
	if token := strings.TrimSpace(pageToken); token != "" {
		call.PageToken(token)
	}
	if language = strings.TrimSpace(language); language != "" {
		call.Language(language)
	}

	resp, err := call.Do()
	if err != nil {
		return GamesAchievementsListInfo{}, mapGoogleAPIErrorWithService("games", err, false)
	}

	items := make([]GamesAchievementInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, gamesAchievementInfoFromDefinition(item))
	}
	return GamesAchievementsListInfo{Achievements: items, NextPageToken: strings.TrimSpace(resp.NextPageToken)}, nil
}

func (c *GamesClient) ListEventDefinitions(ctx context.Context, language string, pageSize int64, pageToken string) (GamesEventsListInfo, error) {
	if c == nil || c.games == nil {
		return GamesEventsListInfo{}, errors.New("games service is not configured")
	}
	if pageSize < 0 {
		return GamesEventsListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}

	call := c.games.Events.ListDefinitions().Context(ctx)
	if pageSize > 0 {
		call.MaxResults(pageSize)
	}
	if token := strings.TrimSpace(pageToken); token != "" {
		call.PageToken(token)
	}
	if language = strings.TrimSpace(language); language != "" {
		call.Language(language)
	}

	resp, err := call.Do()
	if err != nil {
		return GamesEventsListInfo{}, mapGoogleAPIErrorWithService("games", err, false)
	}

	items := make([]GamesEventInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, gamesEventInfoFromDefinition(item))
	}
	return GamesEventsListInfo{Events: items, NextPageToken: strings.TrimSpace(resp.NextPageToken)}, nil
}

func (c *GamesClient) ListLeaderboards(ctx context.Context, language string, pageSize int64, pageToken string) (GamesLeaderboardsListInfo, error) {
	if c == nil || c.games == nil {
		return GamesLeaderboardsListInfo{}, errors.New("games service is not configured")
	}
	if pageSize < 0 {
		return GamesLeaderboardsListInfo{}, fmt.Errorf("page size must be greater than or equal to zero")
	}

	call := c.games.Leaderboards.List().Context(ctx)
	if pageSize > 0 {
		call.MaxResults(pageSize)
	}
	if token := strings.TrimSpace(pageToken); token != "" {
		call.PageToken(token)
	}
	if language = strings.TrimSpace(language); language != "" {
		call.Language(language)
	}

	resp, err := call.Do()
	if err != nil {
		return GamesLeaderboardsListInfo{}, mapGoogleAPIErrorWithService("games", err, false)
	}

	items := make([]GamesLeaderboardInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, gamesLeaderboardInfoFromResource(item))
	}
	return GamesLeaderboardsListInfo{Leaderboards: items, NextPageToken: strings.TrimSpace(resp.NextPageToken)}, nil
}

func (c *GamesClient) GetLeaderboard(ctx context.Context, leaderboardID, language string) (GamesLeaderboardInfo, error) {
	if c == nil || c.games == nil {
		return GamesLeaderboardInfo{}, errors.New("games service is not configured")
	}
	leaderboardID = strings.TrimSpace(leaderboardID)
	if leaderboardID == "" {
		return GamesLeaderboardInfo{}, fmt.Errorf("leaderboard id is required")
	}

	call := c.games.Leaderboards.Get(leaderboardID).Context(ctx)
	if language = strings.TrimSpace(language); language != "" {
		call.Language(language)
	}

	resp, err := call.Do()
	if err != nil {
		return GamesLeaderboardInfo{}, mapGoogleAPIErrorWithService("games", err, false)
	}
	return gamesLeaderboardInfoFromResource(resp), nil
}

func gamesAchievementInfoFromDefinition(def *games.AchievementDefinition) GamesAchievementInfo {
	if def == nil {
		return GamesAchievementInfo{}
	}
	return GamesAchievementInfo{
		ID:               strings.TrimSpace(def.Id),
		Name:             strings.TrimSpace(def.Name),
		Description:      strings.TrimSpace(def.Description),
		AchievementType:  strings.TrimSpace(def.AchievementType),
		InitialState:     strings.TrimSpace(def.InitialState),
		ExperiencePoints: def.ExperiencePoints,
		TotalSteps:       def.TotalSteps,
	}
}

func gamesEventInfoFromDefinition(def *games.EventDefinition) GamesEventInfo {
	if def == nil {
		return GamesEventInfo{}
	}
	return GamesEventInfo{
		ID:              strings.TrimSpace(def.Id),
		DisplayName:     strings.TrimSpace(def.DisplayName),
		Description:     strings.TrimSpace(def.Description),
		Visibility:      strings.TrimSpace(def.Visibility),
		ChildEventCount: len(def.ChildEvents),
	}
}

func gamesLeaderboardInfoFromResource(lb *games.Leaderboard) GamesLeaderboardInfo {
	if lb == nil {
		return GamesLeaderboardInfo{}
	}
	return GamesLeaderboardInfo{
		ID:    strings.TrimSpace(lb.Id),
		Name:  strings.TrimSpace(lb.Name),
		Order: strings.TrimSpace(lb.Order),
	}
}
