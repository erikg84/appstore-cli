package play

import "github.com/dallaslabs/appctl/core/store"

func (c *Client) ListUsers(packageName string) ([]store.AppUser, error) {
    parent, err := c.developerParent(packageName)
    if err != nil {
        return nil, err
    }

    resp, err := c.service.Users.List(parent).PageSize(100).Do()
    if err != nil {
        return nil, err
    }

    users := make([]store.AppUser, 0, len(resp.Users))
    for _, user := range resp.Users {
        users = append(users, store.AppUser{
            ID:             user.Name,
            Email:          user.Email,
            Roles:          append([]string(nil), user.DeveloperAccountPermissions...),
            AllAppsVisible: hasPermission(user.DeveloperAccountPermissions, "CAN_SEE_ALL_APPS") || hasPermission(user.DeveloperAccountPermissions, "CAN_VIEW_NON_FINANCIAL_DATA_GLOBAL"),
        })
    }
    for token := resp.NextPageToken; token != ""; {
        resp, err = c.service.Users.List(parent).PageSize(100).PageToken(token).Do()
        if err != nil {
            return nil, err
        }
        for _, user := range resp.Users {
            users = append(users, store.AppUser{
                ID:             user.Name,
                Email:          user.Email,
                Roles:          append([]string(nil), user.DeveloperAccountPermissions...),
                AllAppsVisible: hasPermission(user.DeveloperAccountPermissions, "CAN_SEE_ALL_APPS") || hasPermission(user.DeveloperAccountPermissions, "CAN_VIEW_NON_FINANCIAL_DATA_GLOBAL"),
            })
        }
        token = resp.NextPageToken
    }
    return users, nil
}

func hasPermission(roles []string, want string) bool {
    for _, role := range roles {
        if role == want {
            return true
        }
    }
    return false
}
