package asc

import (
    "net/url"

    "github.com/dallaslabs/appctl/core/store"
)

func (c *Client) ListBetaGroups(appID string) ([]store.BetaGroup, error) {
    query := url.Values{}
    query.Set("filter[app]", appID)

    var payload map[string]any
    if err := c.getJSON("/v1/betaGroups", query, &payload); err != nil {
        return nil, err
    }

    groups := make([]store.BetaGroup, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        attributes := attrs(item)
        groups = append(groups, store.BetaGroup{
            ID:          str(item, "id"),
            Name:        firstNonEmpty(str(attributes, "name"), str(attributes, "publicLinkName"), str(item, "id")),
            IsInternal:  boolVal(attributes, "isInternalGroup"),
            TesterCount: intVal(attributes, "testerCount"),
        })
    }
    return groups, nil
}

func (c *Client) ListBetaTesters(appID string) ([]store.BetaTester, error) {
    query := url.Values{}
    query.Set("filter[apps]", appID)

    var payload map[string]any
    if err := c.getJSON("/v1/betaTesters", query, &payload); err != nil {
        return nil, err
    }

    testers := make([]store.BetaTester, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        attributes := attrs(item)
        testers = append(testers, store.BetaTester{
            ID:        str(item, "id"),
            Email:     firstNonEmpty(str(attributes, "email"), str(attributes, "inviteEmail")),
            FirstName: str(attributes, "firstName"),
            LastName:  str(attributes, "lastName"),
            State:     firstNonEmpty(str(attributes, "betaTesterState"), str(attributes, "inviteType"), "ACTIVE"),
        })
    }
    return testers, nil
}
