package watchedutil_test

import (
	"testing"

	"github.com/sbondCo/Watcharr/database/dbmodel"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/feature/watched/watchedutil"
)

func TestGetNextUpInTv(t *testing.T) {
	tests := []struct {
		name string
		ws   []entity.WatchedSeason
		we   []entity.WatchedEpisode
		want string
	}{
		{
			name: "watching season, biggest episode finished -> next episode",
			ws: []entity.WatchedSeason{
				{SeasonNumber: 1, Status: entity.FINISHED},
				{SeasonNumber: 2, Status: entity.WATCHING},
				{SeasonNumber: 3, Status: entity.FINISHED},
			},
			we: []entity.WatchedEpisode{
				{GormModelNoDel: dbmodel.GormModelNoDel{ID: 60}, EpisodeNumber: 1, SeasonNumber: 1, Status: entity.FINISHED},
				{GormModelNoDel: dbmodel.GormModelNoDel{ID: 70}, EpisodeNumber: 5, SeasonNumber: 2, Status: entity.FINISHED},
				{GormModelNoDel: dbmodel.GormModelNoDel{ID: 72}, EpisodeNumber: 6, SeasonNumber: 3, Status: entity.DROPPED},
				{GormModelNoDel: dbmodel.GormModelNoDel{ID: 90}, EpisodeNumber: 2, SeasonNumber: 3, Status: entity.FINISHED},
			},
			want: "S2E6",
		},
		{
			name: "only a finished season -> next season, episode 1",
			ws: []entity.WatchedSeason{
				{SeasonNumber: 1, Status: entity.FINISHED},
			},
			want: "S2E1",
		},
		{
			name: "dropped season -> next season, episode 1",
			ws: []entity.WatchedSeason{
				{SeasonNumber: 1, Status: entity.DROPPED},
			},
			want: "S2E1",
		},
		{
			name: "watching season, biggest episode watching -> same episode",
			ws: []entity.WatchedSeason{
				{SeasonNumber: 1, Status: entity.WATCHING},
			},
			we: []entity.WatchedEpisode{
				{EpisodeNumber: 3, SeasonNumber: 1, Status: entity.WATCHING},
			},
			want: "S1E3",
		},
		{
			name: "watching season, biggest episode finished -> next episode in season",
			ws: []entity.WatchedSeason{
				{SeasonNumber: 1, Status: entity.WATCHING},
			},
			we: []entity.WatchedEpisode{
				{EpisodeNumber: 3, SeasonNumber: 1, Status: entity.FINISHED},
			},
			want: "S1E4",
		},
		{
			name: "watching season, no episode rows -> episode 1",
			ws: []entity.WatchedSeason{
				{SeasonNumber: 1, Status: entity.WATCHING},
			},
			want: "S1E1",
		},
		{
			name: "episodes only, no season rows, finished -> next episode",
			we: []entity.WatchedEpisode{
				{EpisodeNumber: 4, SeasonNumber: 2, Status: entity.FINISHED},
			},
			want: "S2E5",
		},
		{
			name: "episodes only, no season rows, watching -> same episode",
			we: []entity.WatchedEpisode{
				{EpisodeNumber: 4, SeasonNumber: 2, Status: entity.WATCHING},
			},
			want: "S2E4",
		},
		{
			name: "episodes only - season takes priority over episode number",
			we: []entity.WatchedEpisode{
				{EpisodeNumber: 10, SeasonNumber: 1, Status: entity.FINISHED},
				{EpisodeNumber: 1, SeasonNumber: 2, Status: entity.WATCHING},
			},
			want: "S2E1",
		},
		{
			name: "nothing watched -> empty",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := watchedutil.GetNextUpInTv(tt.ws, tt.we)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
