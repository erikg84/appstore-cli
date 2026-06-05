package play

import (
    androidpublisher "google.golang.org/api/androidpublisher/v3"

    "github.com/dallaslabs/appctl/core/store"
)

func (c *Client) ListSubscriptions(packageName string) ([]store.Subscription, error) {
    resp, err := c.service.Monetization.Subscriptions.List(packageName).PageSize(100).Do()
    if err != nil {
        return nil, err
    }

    items := make([]store.Subscription, 0, len(resp.Subscriptions))
    for _, sub := range resp.Subscriptions {
        items = append(items, mapSubscription(sub))
    }
    for token := resp.NextPageToken; token != ""; {
        resp, err = c.service.Monetization.Subscriptions.List(packageName).PageSize(100).PageToken(token).Do()
        if err != nil {
            return nil, err
        }
        for _, sub := range resp.Subscriptions {
            items = append(items, mapSubscription(sub))
        }
        token = resp.NextPageToken
    }
    return items, nil
}

func mapSubscription(sub *androidpublisher.Subscription) store.Subscription {
    if sub == nil {
        return store.Subscription{}
    }
    state := "ACTIVE"
    if sub.Archived {
        state = "ARCHIVED"
    }
    return store.Subscription{
        ID:        sub.ProductId,
        ProductID: sub.ProductId,
        Name:      pickSubscriptionName(sub.Listings),
        State:     state,
    }
}

func pickSubscriptionName(listings []*androidpublisher.SubscriptionListing) string {
    for _, listing := range listings {
        if listing != nil && listing.Title != "" {
            return listing.Title
        }
    }
    return ""
}
