package asc

import (
    "fmt"
    "net/url"

    "github.com/dallaslabs/appctl/core/store"
)

func (c *Client) ListSubscriptionGroups(appID string) ([]store.SubscriptionGroup, error) {
    var payload map[string]any
    if err := c.getJSON(fmt.Sprintf("/v1/apps/%s/subscriptionGroups", url.PathEscape(appID)), nil, &payload); err != nil {
        return nil, err
    }

    groups := make([]store.SubscriptionGroup, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        attributes := attrs(item)
        groups = append(groups, store.SubscriptionGroup{
            ID:   str(item, "id"),
            Name: firstNonEmpty(str(attributes, "name"), str(attributes, "referenceName"), str(item, "id")),
        })
    }
    return groups, nil
}

func (c *Client) ListSubscriptions(groupID string) ([]store.Subscription, error) {
    var payload map[string]any
    if err := c.getJSON(fmt.Sprintf("/v1/subscriptionGroups/%s/subscriptions", url.PathEscape(groupID)), nil, &payload); err != nil {
        return nil, err
    }

    subs := make([]store.Subscription, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        attributes := attrs(item)
        subs = append(subs, store.Subscription{
            ID:         str(item, "id"),
            ProductID:  firstNonEmpty(str(attributes, "productId"), str(attributes, "subscriptionId"), str(item, "id")),
            GroupID:    groupID,
            Name:       firstNonEmpty(str(attributes, "name"), str(attributes, "referenceName"), str(attributes, "productId")),
            State:      firstNonEmpty(str(attributes, "state"), str(attributes, "subscriptionState"), str(attributes, "reviewState")),
            ReviewNote: firstNonEmpty(str(attributes, "reviewNote"), str(attributes, "notes")),
        })
    }
    return subs, nil
}
