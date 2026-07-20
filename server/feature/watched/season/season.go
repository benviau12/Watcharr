package season

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WatchedSeasonAddRequest struct {
	WatchedID       uint                 `json:"watchedId"`
	SeasonNumber    int                  `json:"seasonNumber"`
	Status          entity.WatchedStatus `json:"status"`
	Rating          int8                 `json:"rating" binding:"max=10"`
	AddActivity     entity.ActivityType  `json:"-"`
	AddActivityDate time.Time            `json:"-"`
	// Data to add to activity if the season is created.
	// Combined with data we already add.
	AddActivityData map[string]interface{} `json:"-"`
	// If true, and Status is FINISHED or DROPPED, cascade that status down
	// to every episode in the season. Only set by the season router for
	// direct user requests - sync/import/hook callers must leave this
	// false so they don't trigger a redundant TMDB lookup + cascade.
	CascadeStatus bool `json:"-"`
}

type WatchedSeasonAddResponse struct {
	WatchedSeasons []entity.WatchedSeason `json:"watchedSeasons"`
	AddedActivity  entity.Activity        `json:"addedActivity"`
	// Response from the season->episodes status cascade hook, if it ran.
	SeasonStatusChangedHookResponse SeasonStatusChangedHookResponse `json:"seasonStatusChangedHookResponse,omitempty"`
}

// Response from hookSeasonStatusChanged.
type SeasonStatusChangedHookResponse struct {
	// Full refreshed watched-episodes list for the watched item, if any
	// episodes were changed by the cascade.
	WatchedEpisodes []entity.WatchedEpisode `json:"watchedEpisodes,omitempty"`
	// All activities we have added.
	AddedActivities []entity.Activity `json:"addedActivities,omitempty"`
	// All errors (non-fatal, the season update itself already succeeded)
	// encountered while cascading.
	Errors []string `json:"errors,omitempty"`
}

type ContentProvider interface {
	SeasonDetails(tvId string, seasonNumber string) (tmdb.TMDBSeasonDetails, error)
}

type Service struct {
	db               *gorm.DB
	cp               ContentProvider
	activityProvider domain.ActivityAddProvider
}

func NewService(db *gorm.DB, cp ContentProvider, activityProvider domain.ActivityAddProvider) *Service {
	return &Service{
		db,
		cp,
		activityProvider,
	}
}

// Add/edit a watched season.
func (s *Service) AddWatchedSeason(userId uint, ar WatchedSeasonAddRequest) (WatchedSeasonAddResponse, error) {
	slog.Debug("Adding watched season item", "userId", userId, "watchedID", ar.WatchedID, "season", ar.SeasonNumber)
	// 1. Make sure watched item exists and it is the correct type (TV)
	var w entity.Watched
	if resp := s.db.
		Where("id = ? AND user_id = ?", ar.WatchedID, userId).
		Preload("Content").
		Preload("WatchedSeasons").
		Find(&w); resp.Error != nil {
		slog.Error("Failed when adding a watched season", "error", "failed to get watched item from db")
		return WatchedSeasonAddResponse{}, errors.New("failed when retrieving watched item")
	}
	if w.ID == 0 {
		slog.Error("Failed when adding a watched season", "error", "watched item does not exist in db")
		return WatchedSeasonAddResponse{}, errors.New("can't add a watched season for a show that doesnt have a status itself")
	}
	if w.Content.Type != entity.SHOW {
		return WatchedSeasonAddResponse{}, errors.New("can't add watched season for non show content")
	}
	found := false
	updated := false
	for i, ws := range w.WatchedSeasons {
		if ws.SeasonNumber == ar.SeasonNumber {
			slog.Debug("Existing watched season item found, updating existing")
			found = true
			if ar.Status != "" && ar.Status != w.WatchedSeasons[i].Status {
				w.WatchedSeasons[i].Status = ar.Status
				updated = true
			}
			if ar.Rating != 0 && ar.Rating != w.WatchedSeasons[i].Rating {
				w.WatchedSeasons[i].Rating = ar.Rating
				updated = true
			}
			break
		}
	}
	var addedActivity entity.Activity
	if !found {
		slog.Debug("Existing watched season not found, adding as new entry")
		w.WatchedSeasons = append(w.WatchedSeasons, entity.WatchedSeason{
			UserID:       userId,
			WatchedID:    ar.WatchedID,
			SeasonNumber: ar.SeasonNumber,
			Status:       ar.Status,
			Rating:       ar.Rating,
		})
	}
	if resp := s.db.Save(&w.WatchedSeasons); resp.Error != nil {
		slog.Debug("Failed to save watched season item in db", "error", resp.Error)
		return WatchedSeasonAddResponse{}, errors.New("failed to save")
	}
	// Add activity
	if found {
		// Only add change activity if we actually updated a value
		// (changing value to same value doesn't count).
		if updated {
			if ar.Status != "" {
				json, _ := json.Marshal(map[string]interface{}{"season": ar.SeasonNumber, "status": ar.Status})
				addedActivity, _ = s.activityProvider.AddActivity(
					userId,
					domain.ActivityAddProps{
						WatchedID: w.ID,
						Type:      entity.SEASON_STATUS_CHANGED,
						Data:      string(json),
					},
					false,
				)
			}
			if ar.Rating != 0 {
				json, _ := json.Marshal(map[string]interface{}{"season": ar.SeasonNumber, "rating": ar.Rating})
				addedActivity, _ = s.activityProvider.AddActivity(
					userId,
					domain.ActivityAddProps{
						WatchedID: w.ID,
						Type:      entity.SEASON_RATING_CHANGED,
						Data:      string(json),
					},
					false,
				)
			}
		}
	} else {
		actData := map[string]interface{}{"season": ar.SeasonNumber, "status": ar.Status, "rating": ar.Rating}
		if len(ar.AddActivityData) > 0 {
			for k, v := range ar.AddActivityData {
				if _, ok := ar.AddActivityData[k]; ok {
					actData[k] = v
				}
			}
		}
		json, _ := json.Marshal(actData)
		act := domain.ActivityAddProps{WatchedID: w.ID, Type: entity.SEASON_ADDED, Data: string(json)}
		if ar.AddActivity != "" {
			act.Type = ar.AddActivity
		}
		if !ar.AddActivityDate.IsZero() {
			act.CustomDate = &ar.AddActivityDate
		}
		addedActivity, _ = s.activityProvider.AddActivity(userId, act, false)
	}
	resp := WatchedSeasonAddResponse{
		WatchedSeasons: w.WatchedSeasons,
		AddedActivity:  addedActivity,
	}
	if ar.CascadeStatus && (ar.Status == entity.FINISHED || ar.Status == entity.DROPPED) {
		slog.Debug("addWatchedSeason: Season status was changed and cascade requested, calling hook.")
		resp.SeasonStatusChangedHookResponse = s.hookSeasonStatusChanged(userId, w, ar.SeasonNumber, ar.Status)
	}
	return resp, nil
}

// Called after a season's status has been set to FINISHED or DROPPED (with
// cascading requested). Stamps that same status onto every episode in the
// season, without touching any episode's existing rating.
func (s *Service) hookSeasonStatusChanged(userId uint, w entity.Watched, seasonNum int, newStatus entity.WatchedStatus) SeasonStatusChangedHookResponse {
	hookResponse := SeasonStatusChangedHookResponse{}

	seasonDetails, err := s.cp.SeasonDetails(strconv.Itoa(w.Content.TmdbID), strconv.Itoa(seasonNum))
	if err != nil {
		slog.Error("hookSeasonStatusChanged: Failed to get season details!", "error", err)
		hookResponse.Errors = append(hookResponse.Errors, "failed to get season details for show")
		return hookResponse
	}

	// Only load episodes for this season - AddWatchedSeason doesn't
	// preload WatchedEpisodes and there's no need to fetch the whole show's.
	var episodes []entity.WatchedEpisode
	if res := s.db.Where("user_id = ? AND watched_id = ? AND season_number = ?", userId, w.ID, seasonNum).Find(&episodes); res.Error != nil {
		slog.Error("hookSeasonStatusChanged: Failed to get existing watched episodes!", "error", res.Error)
		hookResponse.Errors = append(hookResponse.Errors, "failed to get existing watched episodes")
		return hookResponse
	}

	existingByEpisodeNum := make(map[int]int, len(episodes))
	for i, ep := range episodes {
		existingByEpisodeNum[ep.EpisodeNumber] = i
	}

	changed := 0
	for _, ep := range seasonDetails.Episodes {
		if idx, ok := existingByEpisodeNum[ep.EpisodeNumber]; ok {
			if episodes[idx].Status != newStatus {
				episodes[idx].Status = newStatus
				changed++
			}
			continue
		}
		episodes = append(episodes, entity.WatchedEpisode{
			UserID:        userId,
			WatchedID:     w.ID,
			SeasonNumber:  seasonNum,
			EpisodeNumber: ep.EpisodeNumber,
			Status:        newStatus,
		})
		changed++
	}

	if changed == 0 {
		return hookResponse
	}

	if res := s.db.Save(&episodes); res.Error != nil {
		slog.Error("hookSeasonStatusChanged: Failed to save cascaded episode statuses!", "error", res.Error)
		hookResponse.Errors = append(hookResponse.Errors, "failed to save cascaded episode statuses")
		return hookResponse
	}

	json, _ := json.Marshal(map[string]interface{}{
		"season":          seasonNum,
		"status":          newStatus,
		"episodesChanged": changed,
	})
	addedActivity, _ := s.activityProvider.AddActivity(
		userId,
		domain.ActivityAddProps{
			WatchedID: w.ID,
			Type:      entity.SEASON_EPISODES_STATUS_CHANGED_AUTO,
			Data:      string(json),
		},
		false,
	)
	hookResponse.AddedActivities = append(hookResponse.AddedActivities, addedActivity)

	// Re-query all watched episodes for the item so the client can
	// wholesale-replace its local state.
	var allEpisodes []entity.WatchedEpisode
	if res := s.db.Where("user_id = ? AND watched_id = ?", userId, w.ID).Find(&allEpisodes); res.Error != nil {
		slog.Error("hookSeasonStatusChanged: Failed to re-fetch all watched episodes for response!", "error", res.Error)
		hookResponse.Errors = append(hookResponse.Errors, "failed to fetch updated watched episodes for response")
		return hookResponse
	}
	hookResponse.WatchedEpisodes = allEpisodes

	return hookResponse
}

// Remove a watched season
func (s *Service) RmWatchedSeason(userId uint, seasonId uint) (entity.Activity, error) {
	slog.Debug("rmWatchedSeason called", "user_id", userId, "season_id", seasonId)
	var watchedSeason entity.WatchedSeason
	resp := s.db.
		Clauses(clause.Returning{}).
		Model(&entity.WatchedSeason{}).
		Unscoped().
		Where("id = ? AND user_id = ?", seasonId, userId).
		Delete(&watchedSeason)
	if resp.Error != nil {
		slog.Error("Failed when removing a watched season", "error", resp.Error)
		return entity.Activity{}, errors.New("failed when removing watched season")
	}
	if resp.RowsAffected == 0 {
		slog.Error("Failed when removing a watched season", "error", "zero rows affected")
		return entity.Activity{}, errors.New("wasn't removed from db.. may not exist")
	}
	slog.Debug("rmWatchedSeason, deleted row", "row", watchedSeason)
	if watchedSeason.ID != 0 {
		json, _ := json.Marshal(map[string]interface{}{
			"season": watchedSeason.SeasonNumber,
			"status": watchedSeason.Status,
			"rating": watchedSeason.Rating,
		})
		addedActivity, _ := s.activityProvider.AddActivity(
			userId,
			domain.ActivityAddProps{
				WatchedID: watchedSeason.WatchedID,
				Type:      entity.SEASON_REMOVED,
				Data:      string(json),
			},
			false,
		)
		return addedActivity, nil
	}
	return entity.Activity{}, errors.New("removed, but failed to add activity entry")
}

func (s *Service) GetWatchedSeason(userId uint, watchedId uint, seasonNumber int) (*entity.WatchedSeason, error) {
	var ws *entity.WatchedSeason
	if res := s.db.Model(&entity.WatchedSeason{}).Where("watched_id = ? AND season_number = ? AND user_id = ?", watchedId, seasonNumber, userId).Take(&ws); res.Error != nil {
		slog.Error("getWatchedSeason: Failed to get:", "error", res.Error.Error())
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return &entity.WatchedSeason{}, errors.New("failed to get watched season")
	}
	return ws, nil
}
