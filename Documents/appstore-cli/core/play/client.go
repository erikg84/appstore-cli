package play

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	androidpublisher "google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
	reporting "google.golang.org/api/playdeveloperreporting/v1beta1"

	"github.com/dallaslabs/appctl/core/store"
)

type Client struct {
	service          *androidpublisher.Service
	reporting        *reporting.Service
	tokenSource      oauth2.TokenSource
	developerAccount string
}

func NewClient(serviceAccountFile string) (*Client, error) {
	ctx := context.Background()

	keyData, err := os.ReadFile(serviceAccountFile)
	if err != nil {
		return nil, fmt.Errorf("read play service account key: %w", err)
	}

	// Build a single token source for all services
	creds, err := google.CredentialsFromJSON(
		ctx,
		keyData,
		androidpublisher.AndroidpublisherScope,
		"https://www.googleapis.com/auth/playdeveloperreporting",
		"https://www.googleapis.com/auth/devstorage.read_only",
	)
	if err != nil {
		return nil, err
	}
	ts := creds.TokenSource

	svc, err := androidpublisher.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, err
	}
	rptSvc, err := reporting.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, err
	}
	return &Client{
		service:          svc,
		reporting:        rptSvc,
		tokenSource:      ts,
		developerAccount: strings.TrimSpace(os.Getenv("PLAY_DEVELOPER_ACCOUNT")),
	}, nil
}

// ListApps returns all apps accessible to this service account via the Reporting API.
func (c *Client) ListApps() ([]store.AppSummary, error) {
	resp, err := c.reporting.Apps.Search().Do()
	if err != nil {
		return nil, err
	}
	apps := make([]store.AppSummary, 0, len(resp.Apps))
	for _, a := range resp.Apps {
		apps = append(apps, store.AppSummary{
			Alias:       a.PackageName,
			Name:        a.DisplayName,
			PlayPackage: a.PackageName,
		})
	}
	return apps, nil
}

func (c *Client) ListTracks(packageName string) ([]store.Track, error) {
	edit, err := c.service.Edits.Insert(packageName, &androidpublisher.AppEdit{}).Do()
	if err != nil {
		return nil, err
	}

	resp, err := c.service.Edits.Tracks.List(packageName, edit.Id).Do()
	if err != nil {
		return nil, err
	}

	tracks := make([]store.Track, 0)
	for _, track := range resp.Tracks {
		if len(track.Releases) == 0 {
			tracks = append(tracks, store.Track{Name: track.Track})
			continue
		}
		for _, release := range track.Releases {
			versionCodes := make([]int64, len(release.VersionCodes))
			copy(versionCodes, release.VersionCodes)
			tracks = append(tracks, store.Track{
				Name:         track.Track,
				Status:       release.Status,
				VersionCodes: versionCodes,
				VersionName:  release.Name,
			})
		}
	}
	return tracks, nil
}

func (c *Client) ListReviews(packageName string) ([]store.Review, error) {
	resp, err := c.service.Reviews.List(packageName).MaxResults(100).Do()
	if err != nil {
		return nil, err
	}

	reviews := make([]store.Review, 0, len(resp.Reviews))
	for _, review := range resp.Reviews {
		reviews = append(reviews, mapReview(review))
	}
	for resp != nil && resp.TokenPagination != nil && resp.TokenPagination.NextPageToken != "" {
		token := resp.TokenPagination.NextPageToken
		resp, err = c.service.Reviews.List(packageName).MaxResults(100).Token(token).Do()
		if err != nil {
			return nil, err
		}
		if resp == nil {
			break
		}
		for _, review := range resp.Reviews {
			reviews = append(reviews, mapReview(review))
		}
	}
	return reviews, nil
}

func (c *Client) GetDetails(packageName string) (*androidpublisher.AppDetails, error) {
	edit, err := c.service.Edits.Insert(packageName, &androidpublisher.AppEdit{}).Do()
	if err != nil {
		return nil, err
	}
	return c.service.Edits.Details.Get(packageName, edit.Id).Do()
}

func (c *Client) developerParent(packageName string) (string, error) {
	if strings.HasPrefix(packageName, "developers/") {
		return packageName, nil
	}
	if strings.HasPrefix(c.developerAccount, "developers/") {
		return c.developerAccount, nil
	}
	if c.developerAccount != "" {
		return "developers/" + c.developerAccount, nil
	}
	return "", fmt.Errorf("PLAY_DEVELOPER_ACCOUNT is required for Google Play users")
}

func mapReview(review *androidpublisher.Review) store.Review {
	if review == nil {
		return store.Review{Store: "android"}
	}
	var latest *androidpublisher.UserComment
	for _, comment := range review.Comments {
		if comment != nil && comment.UserComment != nil {
			latest = comment.UserComment
		}
	}
	result := store.Review{
		Store:        "android",
		ReviewerName: review.AuthorName,
	}
	if latest != nil {
		result.Rating = int(latest.StarRating)
		result.AppVersion = latest.AppVersionName
		result.Date = formatTimestamp(latest.LastModified)
		parts := strings.SplitN(latest.Text, "\t", 2)
		result.Title = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			result.Body = strings.TrimSpace(parts[1])
		} else {
			result.Body = strings.TrimSpace(latest.Text)
		}
		if result.Body == "" {
			result.Body = strings.TrimSpace(latest.OriginalText)
		}
	}
	return result
}

func formatTimestamp(ts *androidpublisher.Timestamp) string {
	if ts == nil {
		return ""
	}
	return time.Unix(ts.Seconds, ts.Nanos).UTC().Format(time.RFC3339)
}
