package surugaya

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// --- Response envelope ---

// apiResponse is the common envelope for all Surugaya API responses.
type apiResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	LID  int64           `json:"lid"`
	Msg  string          `json:"msg"`
}

// --- Search types ---

type searchData struct {
	Item   []searchItem `json:"item"`
	MaxNum int          `json:"maxNum"`
	Total  int64        `json:"total"`
}

type searchItem struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Brand       *string     `json:"brand"`   // nullable
	Category    string      `json:"category"`
	Condition   string      `json:"condition"`
	Link        string      `json:"link"`
	Pic         string      `json:"pic"`
	ReleaseDate string      `json:"releaseDate"`
	Sale        []saleEntry `json:"sale"`
	State       *string     `json:"state"` // "品切れ" or null
	StoreTag    string      `json:"storeTag"`
}

type saleEntry struct {
	Price float64 `json:"price"`
	Type  string  `json:"type"` // "中古", "新品", "定価", "予約"
}

// --- Detail types ---

type detailData struct {
	BuyState       bool              `json:"buyState"`
	Title          string            `json:"title"`
	Desc           string            `json:"desc"`
	StateDetail    string            `json:"stateDetail"`
	ImgList        []string          `json:"imgList"`
	Tags           []string          `json:"tags"`
	Category       categoryTree      `json:"category"`
	Detail         map[string]string `json:"detail"`
	ShopSimpleInfo shopInfo          `json:"shopSimpleInfo"`
	Types          []productType     `json:"types"`
	OtherBranch    []otherBranch     `json:"otherBranch"`
	Nyuka          nyukaInfo         `json:"nyuka"`
	Platform       int               `json:"platform"`
}

type categoryTree struct {
	Classify   string        `json:"classify"`
	ClassifyID int           `json:"classifyId"`
	Category   *categoryTree `json:"category,omitempty"` // recursive child
}

type shopInfo struct {
	ShopName string `json:"shopName"`
	ShopPic  string `json:"shopPic"`
	Score    string `json:"score"`
	GoodsNum int    `json:"goodsNum"`
	ShopInfo string `json:"shopInfo"`
}

type productType struct {
	BranchNumber  string  `json:"branchNumber"`
	CartID        string  `json:"cartId"`
	LimitPurchase int     `json:"limitPurchase"` // -1 = cannot buy
	Price         float64 `json:"price"`
	State         string  `json:"state"`
	StateCode     int     `json:"stateCode"`
	Stock         int     `json:"stock"`
	TenpoCD       string  `json:"tenpoCD"`
	Kubun         string  `json:"kubun"`
}

type otherBranch struct {
	ID           string `json:"id"`
	State        string `json:"state"`
	Price        string `json:"price"`
	BranchNumber string `json:"branch_number"`
	TenpoCD      string `json:"tenpo_cd"`
}

type nyukaInfo struct {
	Rem          string      `json:"rem"`
	TenpoCD      string      `json:"tenpoCD"`
	BranchNumber string      `json:"branchNumber"`
	InStock      interface{} `json:"inStock"` // null or number
}

// --- Extend types (comments + similar) ---

// ExtendData holds comments and similar products for a product.
type ExtendData struct {
	Comments  []Comment     `json:"comments"`
	OtherList []SimilarItem `json:"otherList"`
}

// Comment represents a product review/comment.
type Comment struct {
	Star     string `json:"star"`
	Title    string `json:"title"`
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
	Comment  string `json:"comment"`
	Other    string `json:"other"`
}

// SimilarItem represents a similar product recommendation.
type SimilarItem struct {
	Img   string  `json:"img"`
	MID   string  `json:"mid"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	URL   string  `json:"url"`
}

// --- Store types ---

// StoreData holds other available stores for a product.
type StoreData struct {
	Stores []StoreEntry `json:"stores"`
}

// StoreEntry represents a single store offering.
type StoreEntry struct {
	BranchNumber string  `json:"branchNumber"`
	BuyState     bool    `json:"buyState"`
	CartID       string  `json:"cartId"`
	GoodsLink    string  `json:"goodsLink"`
	MailOrder    float64 `json:"mailOrder"`
	Price        float64 `json:"price"`
	PSTime       string  `json:"pstime"`
	Score        string  `json:"score"`
	ScoreDesc    float64 `json:"scoreDesc"`
	State        string  `json:"state"`
	Stock        int     `json:"stock"`
	StoreLink    string  `json:"storeLink"`
	StoreName    string  `json:"storeName"`
	TenpoCD      string  `json:"tenpoCD"`
	Kubun        string  `json:"kubun"`
}

// --- Discount types ---

// DiscountData holds current discount/sale information.
type DiscountData struct {
	TimeSale    []TimeSaleEntry    `json:"timeSale"`
	UnifiedSale []UnifiedSaleEntry `json:"unifiedSale"`
}

// TimeSaleEntry represents a time-limited sale.
type TimeSaleEntry struct {
	Content string `json:"content"`
	PicLink string `json:"picLink"`
	Title   string `json:"title"`
}

// UnifiedSaleEntry represents a platform-wide sale.
type UnifiedSaleEntry struct {
	ActiveTime string              `json:"activeTime"`
	Conditions []DiscountCondition `json:"conditions"`
	Content    string              `json:"content"`
	Title      string              `json:"title"`
}

// DiscountCondition represents a discount tier.
type DiscountCondition struct {
	Count    string `json:"count"`
	Discount string `json:"discount"`
}

// --- User comments types ---

// UserCommentsData holds a user's comment history.
type UserCommentsData struct {
	Comments []UserComment `json:"comments"`
	Platform int           `json:"platform"`
}

// UserComment represents a single user comment.
type UserComment struct {
	Comment  string `json:"comment"`
	Other    string `json:"other"`
	Star     string `json:"star"`
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
}

// --- Campaign types ---

// CampaignData holds the campaign list.
type CampaignData struct {
	Campaigns []Campaign `json:"campaigns"`
}

// Campaign represents a single campaign.
type Campaign struct {
	Conditions []CampaignCondition `json:"conditions"`
	Detail     string              `json:"detail"`
	Image      string              `json:"image"`
	Title      string              `json:"title"`
	URLList    []CampaignURL       `json:"urlList"`
}

// CampaignCondition represents a campaign discount tier.
type CampaignCondition struct {
	Discount string `json:"discount"`
	Num      string `json:"num"`
}

// CampaignURL represents a campaign link.
type CampaignURL struct {
	Alt string `json:"alt"`
	URL string `json:"url"`
}

// --- Campaign detail types ---

// CampaignDetailData holds campaign detail information.
type CampaignDetailData struct {
	Campaigns []CampaignDetailEntry `json:"campaigns"`
	Groups    []CampaignGroup       `json:"groups"`
}

// CampaignDetailEntry represents a campaign detail entry.
type CampaignDetailEntry struct {
	Condition string `json:"condition"`
	Desc      string `json:"desc"`
	Image     string `json:"image"`
	Title     string `json:"title"`
}

// CampaignGroup represents a group of items in a campaign.
type CampaignGroup struct {
	Title             string                 `json:"title"`
	Items             []CampaignGroupItem    `json:"items"`
	OtherSearchParams map[string]interface{} `json:"otherSearchParams"`
}

// CampaignGroupItem represents a single item in a campaign group.
type CampaignGroupItem struct {
	CanBuy       bool                   `json:"canBuy"`
	DetailParams map[string]interface{} `json:"detailParams"`
	ID           string                 `json:"id"`
	Image        string                 `json:"image"`
	Name         string                 `json:"name"`
	URL          string                 `json:"url"`
	PriceList    []CampaignPrice        `json:"priceList"`
}

// CampaignPrice represents a price entry in a campaign item.
type CampaignPrice struct {
	IsRange  bool    `json:"isRange"`
	Price    float64 `json:"price"`
	Status   string  `json:"status"`
	MinPrice string  `json:"minPrice"`
	MaxPrice string  `json:"maxPrice"`
}

// --- Search parameters ---

// SearchParams holds query parameters for the search endpoint.
type SearchParams struct {
	SearchWord       string
	Page             int
	Category         string
	SafeSearchEnable string // "1" = on (default)
	InStock          string // "On" or "Off"
	IsMarketplace    string // "1"=seller priority, "2"=surugaya priority
	RankBy           string // "relavancy(int)", "price:ascending", etc.
	SaleClassified   []string // "中古", "予約", "新品"
	PriceMin         int64
	PriceMax         int64
	PriceRateMin     int64
	PriceRateMax     int64
	ReleaseYearMin   int
	ReleaseYearMax   int
	Hendou           string // "人気上昇中", "値下げ", "新入荷"
	TenpoCode        string
}

// UserCommentsParams holds query parameters for the user comments endpoint.
type UserCommentsParams struct {
	UserID   string
	PageNum  int
	PageSize int
	SortType string // "-rate", "-created_at", "-useful"
}

// --- Client ---

// Client handles HTTP communication with the Surugaya domestic API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a Client for the Surugaya domestic API.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{baseURL: baseURL, http: httpClient}
}

// doGet performs a GET request and decodes the API response envelope.
func (c *Client) doGet(ctx context.Context, path string, query url.Values) (*apiResponse, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("surugaya: parse URL: %w", err)
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("surugaya: create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("surugaya: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("surugaya: unexpected HTTP status %d", resp.StatusCode)
	}

	var envelope apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("surugaya: decode response: %w", err)
	}
	if envelope.Code != 200 {
		return nil, fmt.Errorf("surugaya: API error code=%d msg=%s", envelope.Code, envelope.Msg)
	}
	return &envelope, nil
}

// Search calls the search endpoint.
func (c *Client) Search(ctx context.Context, params SearchParams) (*searchData, error) {
	q := url.Values{}
	q.Set("search_word", params.SearchWord)
	q.Set("page", strconv.Itoa(params.Page))

	if params.SafeSearchEnable != "" {
		q.Set("safe_search_enable", params.SafeSearchEnable)
	}
	if params.Category != "" {
		q.Set("category", params.Category)
	}
	if params.InStock != "" {
		q.Set("inStock", params.InStock)
	}
	if params.IsMarketplace != "" {
		q.Set("is_marketplace", params.IsMarketplace)
	}
	if params.RankBy != "" {
		q.Set("rankBy", params.RankBy)
	}
	if len(params.SaleClassified) > 0 {
		for _, sc := range params.SaleClassified {
			q.Add("sale_classified[]", sc)
		}
	}
	if params.PriceMin > 0 || params.PriceMax > 0 {
		min := "*"
		max := "*"
		if params.PriceMin > 0 {
			min = strconv.FormatInt(params.PriceMin, 10)
		}
		if params.PriceMax > 0 {
			max = strconv.FormatInt(params.PriceMax, 10)
		}
		q.Set("price", fmt.Sprintf("[%s,%s]", min, max))
	}
	if params.PriceRateMin > 0 || params.PriceRateMax > 0 {
		min := "*"
		max := "*"
		if params.PriceRateMin > 0 {
			min = strconv.FormatInt(params.PriceRateMin, 10)
		}
		if params.PriceRateMax > 0 {
			max = strconv.FormatInt(params.PriceRateMax, 10)
		}
		q.Set("price_rate", fmt.Sprintf("[%s,%s]", min, max))
	}
	if params.ReleaseYearMin > 0 || params.ReleaseYearMax > 0 {
		min := "*"
		max := "*"
		if params.ReleaseYearMin > 0 {
			min = strconv.Itoa(params.ReleaseYearMin)
		}
		if params.ReleaseYearMax > 0 {
			max = strconv.Itoa(params.ReleaseYearMax)
		}
		q.Set("release_year", fmt.Sprintf("[%s,%s]", min, max))
	}
	if params.Hendou != "" {
		q.Set("hendou", params.Hendou)
	}
	if params.TenpoCode != "" {
		q.Set("tenpo_code", params.TenpoCode)
	}

	envelope, err := c.doGet(ctx, "/suruga/product/search", q)
	if err != nil {
		return nil, err
	}

	var data searchData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("surugaya: decode search data: %w", err)
	}
	return &data, nil
}

// GetProductDetail fetches product detail by goods ID.
func (c *Client) GetProductDetail(ctx context.Context, goodsID string) (*detailData, error) {
	envelope, err := c.doGet(ctx, "/suruga/product/detail/"+goodsID, nil)
	if err != nil {
		return nil, err
	}

	var data detailData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("surugaya: decode detail data: %w", err)
	}
	return &data, nil
}

// GetProductExtend fetches comments and similar products.
func (c *Client) GetProductExtend(ctx context.Context, goodsID string) (*ExtendData, error) {
	envelope, err := c.doGet(ctx, "/suruga/product/extend/"+goodsID, nil)
	if err != nil {
		return nil, err
	}

	var data ExtendData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("surugaya: decode extend data: %w", err)
	}
	return &data, nil
}

// GetProductStores fetches other available stores for a product.
func (c *Client) GetProductStores(ctx context.Context, goodsID string) (*StoreData, error) {
	envelope, err := c.doGet(ctx, "/suruga/product/extend/store/"+goodsID, nil)
	if err != nil {
		return nil, err
	}

	var data StoreData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("surugaya: decode store data: %w", err)
	}
	return &data, nil
}

// GetDiscount fetches current discount/sale information.
func (c *Client) GetDiscount(ctx context.Context) (*DiscountData, error) {
	envelope, err := c.doGet(ctx, "/suruga/product/discount", nil)
	if err != nil {
		return nil, err
	}

	var data DiscountData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("surugaya: decode discount data: %w", err)
	}
	return &data, nil
}

// GetUserComments fetches a user's comment history.
func (c *Client) GetUserComments(ctx context.Context, params UserCommentsParams) (*UserCommentsData, error) {
	q := url.Values{}
	q.Set("user_id", params.UserID)
	q.Set("page_num", strconv.Itoa(params.PageNum))
	q.Set("page_size", strconv.Itoa(params.PageSize))
	q.Set("sort_type", params.SortType)

	envelope, err := c.doGet(ctx, "/suruga/product/user/comments", q)
	if err != nil {
		return nil, err
	}

	var data UserCommentsData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("surugaya: decode user comments data: %w", err)
	}
	return &data, nil
}

// GetCampaigns fetches the campaign list.
func (c *Client) GetCampaigns(ctx context.Context) (*CampaignData, error) {
	envelope, err := c.doGet(ctx, "/suruga/product/campaign", nil)
	if err != nil {
		return nil, err
	}

	var data CampaignData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("surugaya: decode campaign data: %w", err)
	}
	return &data, nil
}

// GetCampaignDetail fetches campaign detail by URL.
func (c *Client) GetCampaignDetail(ctx context.Context, detailURL string) (*CampaignDetailData, error) {
	q := url.Values{}
	q.Set("url", detailURL)

	envelope, err := c.doGet(ctx, "/suruga/product/campaign/detail", q)
	if err != nil {
		return nil, err
	}

	var data CampaignDetailData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("surugaya: decode campaign detail data: %w", err)
	}
	return &data, nil
}

// Health calls the base URL to verify connectivity.
func (c *Client) Health(ctx context.Context) error {
	// Use search with a minimal query as a health probe since
	// the upstream API does not have a dedicated health endpoint.
	q := url.Values{}
	q.Set("search_word", "test")
	q.Set("page", "1")

	u, err := url.Parse(c.baseURL + "/suruga/product/search")
	if err != nil {
		return fmt.Errorf("surugaya: parse URL: %w", err)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("surugaya: create health request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("surugaya: health request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("surugaya: health returned status %d", resp.StatusCode)
	}
	return nil
}

