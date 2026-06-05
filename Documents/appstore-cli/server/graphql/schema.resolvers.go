package graphql

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dallaslabs/appctl/core/config"
	"github.com/dallaslabs/appctl/core/store"
)

func (r *Resolver) Apps(ctx context.Context) ([]store.AppSummary, error) {
	return graphqlApps(), nil
}

func (r *Resolver) App(ctx context.Context, alias string) (*store.AppSummary, error) {
	app, ok := config.Apps[alias]
	if !ok {
		return nil, nil
	}
	summary := store.AppSummary{
		Alias:       alias,
		Name:        app.Name,
		ASCID:       app.ASCAppID,
		PlayPackage: app.PlayPackage,
	}
	return &summary, nil
}

func (r *Resolver) Versions(ctx context.Context, alias string) ([]store.Version, error) {
	app, err := graphqlApp(alias)
	if err != nil {
		return nil, err
	}
	return r.ascClient().ListVersions(app.ASCAppID)
}

func (r *Resolver) Builds(ctx context.Context, alias string) ([]store.Build, error) {
	app, err := graphqlApp(alias)
	if err != nil {
		return nil, err
	}
	return r.ascClient().ListBuilds(app.ASCAppID)
}

func (r *Resolver) Tracks(ctx context.Context, alias string) ([]store.Track, error) {
	app, err := graphqlApp(alias)
	if err != nil {
		return nil, err
	}
	if app.PlayPackage == "" {
		return nil, fmt.Errorf("app %q has no Google Play package", alias)
	}
	pc, err := r.playClient()
	if err != nil {
		return nil, err
	}
	return pc.ListTracks(app.PlayPackage)
}

func (r *Resolver) Reviews(ctx context.Context, alias, storeName string) ([]store.Review, error) {
	app, err := graphqlApp(alias)
	if err != nil {
		return nil, err
	}
	storeName, err = graphqlStore(storeName)
	if err != nil {
		return nil, err
	}
	var reviews []store.Review
	if storeName == "ios" || storeName == "both" {
		iosReviews, err := r.ascClient().ListReviews(app.ASCAppID)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, iosReviews...)
	}
	if storeName == "android" || storeName == "both" {
		if app.PlayPackage != "" {
			pc, err := r.playClient()
			if err != nil && storeName == "android" {
				return nil, err
			}
			if err == nil {
				androidReviews, err := pc.ListReviews(app.PlayPackage)
				if err != nil {
					return nil, err
				}
				reviews = append(reviews, androidReviews...)
			}
		} else if storeName == "android" {
			return nil, fmt.Errorf("app %q has no Google Play package", alias)
		}
	}
	return reviews, nil
}

func (r *Resolver) IAP(ctx context.Context, alias, storeName string) ([]store.IAP, error) {
	app, err := graphqlApp(alias)
	if err != nil {
		return nil, err
	}
	storeName, err = graphqlStore(storeName)
	if err != nil {
		return nil, err
	}
	var items []store.IAP
	if storeName == "ios" || storeName == "both" {
		iosItems, err := r.ascClient().ListIAPs(app.ASCAppID)
		if err != nil {
			return nil, err
		}
		items = append(items, iosItems...)
	}
	if storeName == "android" || storeName == "both" {
		if app.PlayPackage != "" {
			pc, err := r.playClient()
			if err != nil && storeName == "android" {
				return nil, err
			}
			if err == nil {
				androidItems, err := pc.ListIAPs(app.PlayPackage)
				if err != nil {
					return nil, err
				}
				items = append(items, androidItems...)
			}
		} else if storeName == "android" {
			return nil, fmt.Errorf("app %q has no Google Play package", alias)
		}
	}
	return items, nil
}

func (r *Resolver) Subscriptions(ctx context.Context, alias, storeName string) ([]store.Subscription, error) {
	app, err := graphqlApp(alias)
	if err != nil {
		return nil, err
	}
	storeName, err = graphqlStore(storeName)
	if err != nil {
		return nil, err
	}
	var items []store.Subscription
	if storeName == "ios" || storeName == "both" {
		asc := r.ascClient()
		groups, err := asc.ListSubscriptionGroups(app.ASCAppID)
		if err != nil {
			return nil, err
		}
		for _, group := range groups {
			subs, err := asc.ListSubscriptions(group.ID)
			if err != nil {
				return nil, err
			}
			items = append(items, subs...)
		}
	}
	if storeName == "android" || storeName == "both" {
		if app.PlayPackage != "" {
			pc, err := r.playClient()
			if err != nil && storeName == "android" {
				return nil, err
			}
			if err == nil {
				androidItems, err := pc.ListSubscriptions(app.PlayPackage)
				if err != nil {
					return nil, err
				}
				items = append(items, androidItems...)
			}
		} else if storeName == "android" {
			return nil, fmt.Errorf("app %q has no Google Play package", alias)
		}
	}
	return items, nil
}

func (r *Resolver) TestFlightGroups(ctx context.Context, alias string) ([]store.BetaGroup, error) {
	app, err := graphqlApp(alias)
	if err != nil {
		return nil, err
	}
	return r.ascClient().ListBetaGroups(app.ASCAppID)
}

func (r *Resolver) TestFlightTesters(ctx context.Context, alias string) ([]store.BetaTester, error) {
	app, err := graphqlApp(alias)
	if err != nil {
		return nil, err
	}
	return r.ascClient().ListBetaTesters(app.ASCAppID)
}

func (r *Resolver) Users(ctx context.Context) ([]store.AppUser, error) {
	users, err := r.ascClient().ListUsers()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(os.Getenv("PLAY_DEVELOPER_ACCOUNT")) != "" {
		if pc, err := r.playClient(); err == nil {
			playUsers, err := pc.ListUsers("")
			if err != nil {
				return nil, err
			}
			users = append(users, playUsers...)
		}
	}
	return users, nil
}

func graphqlApp(alias string) (config.App, error) {
	app, ok := config.Apps[alias]
	if !ok {
		return config.App{}, fmt.Errorf("unknown app %q", alias)
	}
	return app, nil
}

func graphqlApps() []store.AppSummary {
	aliases := make([]string, 0, len(config.Apps))
	for alias := range config.Apps {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	apps := make([]store.AppSummary, 0, len(aliases))
	for _, alias := range aliases {
		app := config.Apps[alias]
		apps = append(apps, store.AppSummary{
			Alias:       alias,
			Name:        app.Name,
			ASCID:       app.ASCAppID,
			PlayPackage: app.PlayPackage,
		})
	}
	return apps
}

func graphqlStore(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "both", nil
	}
	switch value {
	case "ios", "android", "both":
		return value, nil
	default:
		return "", fmt.Errorf("invalid store %q", value)
	}
}
