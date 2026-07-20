package content

import (
	"log/slog"
	"sort"

	"github.com/sbondCo/Watcharr/media/tmdb"
)

// Getting only region needed from api is not a feature yet
// https://trello.com/c/75tR4cpF/106-add-watch-provider-region-filtering
// When it is, this can be removed for that instead.
func transformProviders(c *any, country string) tmdb.WatchProviders {
	slog.Debug("transformProviders called", "country", country)
	resp := tmdb.WatchProviders{}

	cmap, ok := (*c).(map[string]any)
	if !ok {
		slog.Error("transformProviders: Assertion failed")
		return tmdb.WatchProviders{}
	}

	rmap, ok := cmap["results"].(map[string]any)
	if !ok {
		slog.Warn("transformProviders: Couldn't find results property..")
		return tmdb.WatchProviders{}
	}

	val, ok := rmap[country]
	if !ok {
		slog.Warn("transformProviders: Couldn't find country..",
			"country", country)
		return tmdb.WatchProviders{}
	}
	slog.Debug("transformProviders: Found country..", "obj", val)

	rvmap, ok := val.(map[string]any)
	if !ok {
		slog.Warn("transformProviders: Couldn't assert country obj")
		return tmdb.WatchProviders{}
	}

	// Turning any into a type safe object we can use later.
	// Here we are just getting the flatrate items and manually
	// mapping them to a WatchProvider struct.
	resp.Flatrate = transformProvidersType("flatrate", rvmap, resp.Flatrate)
	resp.Free = transformProvidersType("free", rvmap, resp.Free)
	resp.Rent = transformProvidersType("rent", rvmap, resp.Rent)
	resp.Buy = transformProvidersType("buy", rvmap, resp.Buy)

	tmdbLink, ok := rvmap["link"].(string)
	if ok {
		resp.Link = tmdbLink
	}

	return resp
}

// Transform the type of provider requested.
func transformProvidersType(
	ptype string,
	rvmap map[string]any,
	providers []tmdb.WatchProvider,
) []tmdb.WatchProvider {
	tm, ok := rvmap[ptype].([]any)
	if !ok {
		slog.Warn("transformProvidersType: Assertion failed")
		return providers
	}
	start := len(providers)
	for i := range tm {
		v2, ok := tm[i].(map[string]any)
		if !ok {
			continue
		}
		providerName, ok := v2["provider_name"].(string)
		if !ok {
			continue
		}
		logoPath, _ := v2["logo_path"].(string)
		// display_priority comes through as a float64 from the JSON decoder.
		displayPriority, _ := v2["display_priority"].(float64)
		providers = append(providers,
			tmdb.WatchProvider{
				ProviderName:    providerName,
				LogoPath:        logoPath,
				DisplayPriority: int(displayPriority),
			})
	}
	// Sort just the entries we added, so the most relevant providers
	// (as ranked by tmdb) show up first.
	added := providers[start:]
	sort.Slice(added, func(i, j int) bool {
		return added[i].DisplayPriority < added[j].DisplayPriority
	})
	return providers
}
