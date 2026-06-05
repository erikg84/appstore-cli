package asc

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "sort"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/golang-jwt/jwt/v5"

    "github.com/dallaslabs/appctl/core/store"
)

const defaultBaseURL = "https://api.appstoreconnect.apple.com"

type SubscriptionGroup = store.SubscriptionGroup

type Client struct {
    keyID    string
    issuerID string
    keyFile  string
    baseURL  string
    client   *http.Client

    mu       sync.Mutex
    token    string
    tokenExp time.Time
}

func NewClient(keyID, issuerID, keyFile string) *Client {
    return &Client{
        keyID:    keyID,
        issuerID: issuerID,
        keyFile:  keyFile,
        baseURL:  defaultBaseURL,
        client:   &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *Client) ListApps() ([]store.AppSummary, error) {
    var payload map[string]any
    if err := c.getJSON("/v1/apps", nil, &payload); err != nil {
        return nil, err
    }

    apps := make([]store.AppSummary, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        attributes := attrs(item)
        apps = append(apps, store.AppSummary{
            Alias:       str(item, "id"),
            Name:        firstNonEmpty(str(attributes, "name"), str(attributes, "bundleId"), str(item, "id")),
            ASCID:       str(item, "id"),
            PlayPackage: str(attributes, "bundleId"),
        })
    }

    sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
    return apps, nil
}

func (c *Client) ListVersions(appID string) ([]store.Version, error) {
    var payload map[string]any
    if err := c.getJSON(fmt.Sprintf("/v1/apps/%s/appStoreVersions", url.PathEscape(appID)), nil, &payload); err != nil {
        return nil, err
    }

    versions := make([]store.Version, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        attributes := attrs(item)
        versions = append(versions, store.Version{
            Platform:      str(attributes, "platform"),
            VersionString: str(attributes, "versionString"),
            State:         firstNonEmpty(str(attributes, "appStoreState"), str(attributes, "appVersionState"), str(attributes, "state")),
            CreatedDate:   str(attributes, "createdDate"),
        })
    }
    return versions, nil
}

func (c *Client) ListBuilds(appID string) ([]store.Build, error) {
    query := url.Values{}
    query.Set("filter[app]", appID)

    var payload map[string]any
    if err := c.getJSON("/v1/builds", query, &payload); err != nil {
        return nil, err
    }

    builds := make([]store.Build, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        attributes := attrs(item)
        builds = append(builds, store.Build{
            Version:      firstNonEmpty(str(attributes, "version"), str(attributes, "buildVersion")),
            State:        firstNonEmpty(str(attributes, "processingState"), str(attributes, "betaReviewState"), str(attributes, "state")),
            UploadedDate: firstNonEmpty(str(attributes, "uploadedDate"), str(attributes, "expirationDate")),
            Platform:     str(attributes, "platform"),
        })
    }
    return builds, nil
}

func (c *Client) ListReviews(appID string) ([]store.Review, error) {
    var payload map[string]any
    if err := c.getJSON(fmt.Sprintf("/v1/apps/%s/customerReviews", url.PathEscape(appID)), nil, &payload); err != nil {
        return nil, err
    }

    reviews := make([]store.Review, 0, len(dataItems(payload)))
    for _, item := range dataItems(payload) {
        attributes := attrs(item)
        reviews = append(reviews, store.Review{
            Store:        "ios",
            Rating:       intVal(attributes, "rating"),
            Title:        firstNonEmpty(str(attributes, "title"), str(attributes, "headline")),
            Body:         firstNonEmpty(str(attributes, "body"), str(attributes, "review")),
            ReviewerName: firstNonEmpty(str(attributes, "reviewerName"), str(attributes, "reviewerNickname"), str(attributes, "nickname")),
            Date:         firstNonEmpty(str(attributes, "createdDate"), str(attributes, "lastModifiedDate")),
            AppVersion:   firstNonEmpty(str(attributes, "appVersionString"), str(attributes, "territory")),
        })
    }
    return reviews, nil
}

func (c *Client) getJSON(path string, query url.Values, target any) error {
    body, err := c.get(path, query)
    if err != nil {
        return err
    }
    return jsonUnmarshal(body, target)
}

func (c *Client) get(path string, query url.Values) ([]byte, error) {
    body, _, err := c.getWithHeaders(path, query)
    return body, err
}

func (c *Client) getWithHeaders(path string, query url.Values) ([]byte, http.Header, error) {
    return c.getWithAccept(path, query, "application/json")
}

func (c *Client) getWithAccept(path string, query url.Values, accept string) ([]byte, http.Header, error) {
    token, err := c.bearerToken()
    if err != nil {
        return nil, nil, err
    }

    u, err := url.Parse(c.baseURL + path)
    if err != nil {
        return nil, nil, err
    }
    if query != nil {
        u.RawQuery = query.Encode()
    }

    req, err := http.NewRequest(http.MethodGet, u.String(), nil)
    if err != nil {
        return nil, nil, err
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Accept", accept)

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, resp.Header, err
    }
    if resp.StatusCode >= 300 {
        return nil, resp.Header, fmt.Errorf("app store connect %s %s: %s", req.Method, u.Path, truncate(body))
    }
    return body, resp.Header, nil
}

func (c *Client) bearerToken() (string, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.token != "" && time.Until(c.tokenExp) > 2*time.Minute {
        return c.token, nil
    }

    keyData, err := os.ReadFile(c.keyFile)
    if err != nil {
        return "", err
    }

    privateKey, err := jwt.ParseECPrivateKeyFromPEM(keyData)
    if err != nil {
        return "", err
    }

    now := time.Now().UTC()
    claims := jwt.MapClaims{
        "iss": c.issuerID,
        "aud": "appstoreconnect-v1",
        "iat": now.Unix(),
        "exp": now.Add(20 * time.Minute).Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
    token.Header["kid"] = c.keyID
    token.Header["typ"] = "JWT"

    signed, err := token.SignedString(privateKey)
    if err != nil {
        return "", err
    }

    c.token = signed
    c.tokenExp = now.Add(20 * time.Minute)
    return signed, nil
}

func dataItems(payload map[string]any) []map[string]any {
    if payload == nil {
        return nil
    }
    if raw, ok := payload["data"].([]any); ok {
        items := make([]map[string]any, 0, len(raw))
        for _, item := range raw {
            if m, ok := item.(map[string]any); ok {
                items = append(items, m)
            }
        }
        return items
    }
    if raw, ok := payload["data"].(map[string]any); ok {
        return []map[string]any{raw}
    }
    return nil
}

func attrs(item map[string]any) map[string]any {
    if item == nil {
        return nil
    }
    if raw, ok := item["attributes"].(map[string]any); ok {
        return raw
    }
    return nil
}

func str(m map[string]any, key string) string {
    if m == nil {
        return ""
    }
    return stringFrom(m[key])
}

func stringFrom(v any) string {
    switch t := v.(type) {
    case nil:
        return ""
    case string:
        return t
    case fmt.Stringer:
        return t.String()
    case float64:
        return strconv.FormatFloat(t, 'f', -1, 64)
    case int:
        return strconv.Itoa(t)
    case int64:
        return strconv.FormatInt(t, 10)
    case bool:
        return strconv.FormatBool(t)
    case map[string]any:
        keys := make([]string, 0, len(t))
        for key := range t {
            keys = append(keys, key)
        }
        sort.Strings(keys)
        for _, key := range keys {
            if s := stringFrom(t[key]); s != "" {
                return s
            }
        }
    case []any:
        parts := make([]string, 0, len(t))
        for _, item := range t {
            if s := stringFrom(item); s != "" {
                parts = append(parts, s)
            }
        }
        return strings.Join(parts, ",")
    }
    return fmt.Sprint(v)
}

func intVal(m map[string]any, key string) int {
    if m == nil {
        return 0
    }
    switch t := m[key].(type) {
    case float64:
        return int(t)
    case float32:
        return int(t)
    case int:
        return t
    case int64:
        return int(t)
    case string:
        v, _ := strconv.Atoi(t)
        return v
    case json.Number:
        v, _ := t.Int64()
        return int(v)
    }
    return 0
}

func boolVal(m map[string]any, key string) bool {
    if m == nil {
        return false
    }
    switch t := m[key].(type) {
    case bool:
        return t
    case string:
        v, _ := strconv.ParseBool(t)
        return v
    }
    return false
}

func firstNonEmpty(values ...string) string {
    for _, value := range values {
        if strings.TrimSpace(value) != "" {
            return value
        }
    }
    return ""
}

func truncate(body []byte) string {
    text := strings.TrimSpace(string(body))
    if len(text) > 200 {
        return text[:200] + "..."
    }
    return text
}

func jsonUnmarshal(body []byte, target any) error {
    dec := json.NewDecoder(strings.NewReader(string(body)))
    dec.UseNumber()
    return dec.Decode(target)
}
