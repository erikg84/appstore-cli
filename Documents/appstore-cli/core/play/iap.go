package play

import (
    "fmt"

    androidpublisher "google.golang.org/api/androidpublisher/v3"

    "github.com/dallaslabs/appctl/core/store"
)

// ListIAPs lists one-time in-app products using the new Monetization API.
func (c *Client) ListIAPs(packageName string) ([]store.IAP, error) {
    resp, err := c.service.Monetization.Onetimeproducts.List(packageName).Do()
    if err != nil {
        return nil, err
    }
    items := make([]store.IAP, 0, len(resp.OneTimeProducts))
    for _, p := range resp.OneTimeProducts {
        items = append(items, mapOneTimeProduct(p))
    }
    return items, nil
}

// GetIAP retrieves a single one-time in-app product.
func (c *Client) GetIAP(packageName, productID string) (*store.IAP, error) {
    p, err := c.service.Monetization.Onetimeproducts.Get(packageName, productID).Do()
    if err != nil {
        return nil, err
    }
    iap := mapOneTimeProduct(p)
    return &iap, nil
}

func mapOneTimeProduct(p *androidpublisher.OneTimeProduct) store.IAP {
    if p == nil {
        return store.IAP{}
    }
    name := ""
    for _, l := range p.Listings {
        if l.Title != "" {
            name = l.Title
            break
        }
    }
    // Best-effort price from the first purchase option's first regional config
    price := ""
    if len(p.PurchaseOptions) > 0 {
        configs := p.PurchaseOptions[0].RegionalPricingAndAvailabilityConfigs
        for _, rc := range configs {
            if rc.Price != nil {
                price = fmt.Sprintf("%s %s", rc.Price.CurrencyCode,
                    formatMicros(rc.Price.Units, int32(rc.Price.Nanos)))
                break
            }
        }
    }
    return store.IAP{
        ID:          p.ProductId,
        ProductID:   p.ProductId,
        ProductType: "one_time",
        Name:        name,
        Price:       price,
    }
}

func formatMicros(units int64, nanos int32) string {
    return fmt.Sprintf("%.2f", float64(units)+float64(nanos)/1e9)
}

