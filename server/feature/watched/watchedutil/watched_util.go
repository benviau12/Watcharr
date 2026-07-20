package watchedutil

import (
	"strconv"

	"github.com/sbondCo/Watcharr/database/entity"
)

// Get the "next up" episode for a show - the episode the user should
// watch next, given what they've marked watched so far.
func GetNextUpInTv(
	ws []entity.WatchedSeason,
	we []entity.WatchedEpisode,
) string {
	if len(ws) <= 0 && len(we) <= 0 {
		return ""
	}

	seasonNum, seasonStatus, foundSeason := getCurrentSeasonInTv(ws)

	if foundSeason {
		if seasonStatus == entity.FINISHED || seasonStatus == entity.DROPPED {
			// The current season is done, so next up is the first episode
			// of the next season.
			//
			// NOTE: We don't have this show's total season count here, so
			// if this was the final season, we will point at a season that
			// doesn't exist yet. Threading per-season episode counts through
			// to here would mean a TMDB/cache lookup per list item on every
			// page load, which isn't worth it for this cosmetic detail.
			return SeasonAndEpToReadable(seasonNum+1, 1)
		}

		// Season is still in progress (WATCHING) - find how far into it
		// we've gotten.
		biggest := getBiggestEpisodeInSeason(we, seasonNum)
		if biggest == nil {
			return SeasonAndEpToReadable(seasonNum, 1)
		}
		if biggest.Status == entity.FINISHED || biggest.Status == entity.DROPPED {
			return SeasonAndEpToReadable(biggest.SeasonNumber, biggest.EpisodeNumber+1)
		}
		return SeasonAndEpToReadable(biggest.SeasonNumber, biggest.EpisodeNumber)
	}

	// No season data at all - fall back to episode data only.
	biggest := getBiggestEpisodeOverall(we)
	if biggest == nil {
		return ""
	}
	if biggest.Status == entity.FINISHED || biggest.Status == entity.DROPPED {
		return SeasonAndEpToReadable(biggest.SeasonNumber, biggest.EpisodeNumber+1)
	}
	return SeasonAndEpToReadable(biggest.SeasonNumber, biggest.EpisodeNumber)
}

// Find the season the user is currently on: the biggest WATCHING season,
// or otherwise the biggest FINISHED/DROPPED season.
func getCurrentSeasonInTv(ws []entity.WatchedSeason) (seasonNum int, status entity.WatchedStatus, found bool) {
	watchingSeason := -1
	doneSeason := -1
	var doneStatus entity.WatchedStatus

	for i := range ws {
		v := &ws[i]
		switch v.Status {
		case entity.WATCHING:
			if v.SeasonNumber > watchingSeason {
				watchingSeason = v.SeasonNumber
			}
		case entity.FINISHED, entity.DROPPED:
			if v.SeasonNumber > doneSeason {
				doneSeason = v.SeasonNumber
				doneStatus = v.Status
			}
		}
	}

	if watchingSeason >= 0 {
		return watchingSeason, entity.WATCHING, true
	}
	if doneSeason >= 0 {
		return doneSeason, doneStatus, true
	}
	return -1, "", false
}

// Find the episode with the biggest episode number within a specific season.
func getBiggestEpisodeInSeason(we []entity.WatchedEpisode, seasonNum int) *entity.WatchedEpisode {
	var biggest *entity.WatchedEpisode
	for i := range we {
		v := &we[i]
		if v.SeasonNumber != seasonNum {
			continue
		}
		if biggest == nil || v.EpisodeNumber > biggest.EpisodeNumber {
			biggest = v
		}
	}
	return biggest
}

// Find the episode with the biggest season/episode number across all
// seasons (season takes priority over episode number).
func getBiggestEpisodeOverall(we []entity.WatchedEpisode) *entity.WatchedEpisode {
	var biggest *entity.WatchedEpisode
	for i := range we {
		v := &we[i]
		if biggest == nil ||
			v.SeasonNumber > biggest.SeasonNumber ||
			(v.SeasonNumber == biggest.SeasonNumber && v.EpisodeNumber > biggest.EpisodeNumber) {
			biggest = v
		}
	}
	return biggest
}

func SeasonAndEpToReadable(
	seasonNum int,
	episodeNum int,
) string {
	return "S" + strconv.Itoa(seasonNum) + "E" + strconv.Itoa(episodeNum)
}
