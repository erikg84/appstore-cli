package asc

import (
    "fmt"
    "net/url"

    "github.com/dallaslabs/appctl/core/store"
)

func (c *Client) ListIAPs(appID string) ([]store.IAP, error) {
    var payload map[string]any
    if err := c.getJSON(fmt.Sprintf("/v1/apps/%s/inAppPurchasesV2", url.PathEscape(appID)), nil, &payload); err != nil {
        return nil, err
    }

    items := make([]store.IAP, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        items = append(items, mapIAP(item))
    }
    return items, nil
}

func (c *Client) GetIAP(iapID string) (*store.IAP, error) {
    var payload map[string]any
    if err := c.getJSON(fmt.Sprintf("/v1/inAppPurchasesV2/%s", url.PathEscape(iapID)), nil, &payload); err != nil {
        return nil, err
    }
    items := dataItems(payload)
    if len(items) == 0 {
        return nil, nil
    }
    iap := mapIAP(items[0])
    return &iap, nil
}

func mapIAP(item map[string]any) store.IAP {
    attributes := attrs(item)
    return store.IAP{
        ID:          str(item, "id"),
        ProductID:   firstNonEmpty(str(attributes, "productId"), str(attributes, "referenceName"), str(item, "id")),
        ProductType: firstNonEmpty(str(attributes, "inAppPurchaseType"), str(attributes, "productType"), str(attributes, "type")),
        Name:        firstNonEmpty(str(attributes, "name"), str(attributes, "referenceName"), str(attributes, "productId")),
        State:       firstNonEmpty(str(attributes, "state"), str(attributes, "reviewState"), str(attributes, "inAppPurchaseState")),
        Price:       firstNonEmpty(str(attributes, "price"), str(attributes, "displayPrice"), str(attributes, "customerPrice")),
    }
}
