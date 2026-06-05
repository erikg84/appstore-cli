package asc

import "github.com/dallaslabs/appctl/core/store"

func (c *Client) ListUsers() ([]store.AppUser, error) {
    var payload map[string]any
    if err := c.getJSON("/v1/users", nil, &payload); err != nil {
        return nil, err
    }

    users := make([]store.AppUser, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        attributes := attrs(item)
        users = append(users, store.AppUser{
            ID:             str(item, "id"),
            Email:          firstNonEmpty(str(attributes, "username"), str(attributes, "email")),
            Roles:          stringList(attributes["roles"]),
            AllAppsVisible: boolVal(attributes, "allAppsVisible"),
        })
    }
    return users, nil
}

func stringList(v any) []string {
    switch t := v.(type) {
    case []string:
        return append([]string(nil), t...)
    case []any:
        items := make([]string, 0, len(t))
        for _, item := range t {
            if s := stringFrom(item); s != "" {
                items = append(items, s)
            }
        }
        return items
    default:
        if s := stringFrom(v); s != "" {
            return []string{s}
        }
    }
    return nil
}
