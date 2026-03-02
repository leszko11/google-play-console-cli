package gpc

type CredentialInput struct {
	ServiceAccountPath string
	ServiceAccountJSON []byte
}

type AppInfo struct {
	PackageName string `json:"packageName"`
}

type EditInfo struct {
	ID                string `json:"id"`
	ExpiryTimeSeconds string `json:"expiryTimeSeconds,omitempty"`
}

type ListingInfo struct {
	Language         string `json:"language"`
	Title            string `json:"title,omitempty"`
	ShortDescription string `json:"shortDescription,omitempty"`
	FullDescription  string `json:"fullDescription,omitempty"`
}

type ListingUpdate struct {
	Title            string
	ShortDescription string
	FullDescription  string
}

type AppDetailsInfo struct {
	DefaultLanguage string `json:"defaultLanguage,omitempty"`
	ContactEmail    string `json:"contactEmail,omitempty"`
	ContactPhone    string `json:"contactPhone,omitempty"`
	ContactWebsite  string `json:"contactWebsite,omitempty"`
}

type AppDetailsUpdate struct {
	DefaultLanguage string
	ContactEmail    string
	ContactPhone    string
	ContactWebsite  string
}

type ImageInfo struct {
	ID     string `json:"id,omitempty"`
	SHA1   string `json:"sha1,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	URL    string `json:"url,omitempty"`
}

type TestersInfo struct {
	Track        string   `json:"track,omitempty"`
	GoogleGroups []string `json:"googleGroups,omitempty"`
}

type CountryTargetedInfo struct {
	CountryCode string `json:"countryCode,omitempty"`
}

type CountryAvailabilityInfo struct {
	Track              string                `json:"track,omitempty"`
	Countries          []CountryTargetedInfo `json:"countries,omitempty"`
	RestOfWorld        bool                  `json:"restOfWorld,omitempty"`
	SyncWithProduction bool                  `json:"syncWithProduction,omitempty"`
}

type ReviewInfo struct {
	ReviewID   string `json:"reviewId,omitempty"`
	AuthorName string `json:"authorName,omitempty"`
}

type ReviewsListInfo struct {
	Reviews   []ReviewInfo `json:"reviews,omitempty"`
	NextToken string       `json:"nextToken,omitempty"`
}

type ReviewReplyInfo struct {
	ReplyText         string `json:"replyText,omitempty"`
	LastEditedSeconds int64  `json:"lastEditedSeconds,omitempty"`
	LastEditedNanos   int64  `json:"lastEditedNanos,omitempty"`
}

type UserInfo struct {
	Name                        string   `json:"name,omitempty"`
	Email                       string   `json:"email,omitempty"`
	AccessState                 string   `json:"accessState,omitempty"`
	ExpirationTime              string   `json:"expirationTime,omitempty"`
	Partial                     bool     `json:"partial,omitempty"`
	DeveloperAccountPermissions []string `json:"developerAccountPermissions,omitempty"`
	GrantCount                  int      `json:"grantCount,omitempty"`
}

type UsersListInfo struct {
	Users         []UserInfo `json:"users,omitempty"`
	NextPageToken string     `json:"nextPageToken,omitempty"`
}

type GrantInfo struct {
	Name                string   `json:"name,omitempty"`
	PackageName         string   `json:"packageName,omitempty"`
	AppLevelPermissions []string `json:"appLevelPermissions,omitempty"`
	PermissionCount     int      `json:"permissionCount,omitempty"`
}

type OneTimeProductInfo struct {
	PackageName         string `json:"packageName,omitempty"`
	ProductID           string `json:"productId,omitempty"`
	ListingCount        int    `json:"listingCount,omitempty"`
	PurchaseOptionCount int    `json:"purchaseOptionCount,omitempty"`
	OfferTagCount       int    `json:"offerTagCount,omitempty"`
}

type OneTimeProductsListInfo struct {
	Products      []OneTimeProductInfo `json:"products,omitempty"`
	NextPageToken string               `json:"nextPageToken,omitempty"`
}

type IAPInfo struct {
	PackageName  string `json:"packageName,omitempty"`
	SKU          string `json:"sku,omitempty"`
	Status       string `json:"status,omitempty"`
	PurchaseType string `json:"purchaseType,omitempty"`
	ListingCount int    `json:"listingCount,omitempty"`
	PriceCount   int    `json:"priceCount,omitempty"`
}

type IAPsListInfo struct {
	Products      []IAPInfo `json:"products,omitempty"`
	NextPageToken string    `json:"nextPageToken,omitempty"`
}

type SubscriptionInfo struct {
	PackageName   string `json:"packageName,omitempty"`
	ProductID     string `json:"productId,omitempty"`
	Archived      bool   `json:"archived,omitempty"`
	BasePlanCount int    `json:"basePlanCount,omitempty"`
	ListingCount  int    `json:"listingCount,omitempty"`
}

type SubscriptionsListInfo struct {
	Subscriptions []SubscriptionInfo `json:"subscriptions,omitempty"`
	NextPageToken string             `json:"nextPageToken,omitempty"`
}

type SubscriptionOfferInfo struct {
	PackageName string `json:"packageName,omitempty"`
	ProductID   string `json:"productId,omitempty"`
	BasePlanID  string `json:"basePlanId,omitempty"`
	OfferID     string `json:"offerId,omitempty"`
	State       string `json:"state,omitempty"`
	PhaseCount  int    `json:"phaseCount,omitempty"`
	TagCount    int    `json:"tagCount,omitempty"`
}

type SubscriptionOffersListInfo struct {
	Offers        []SubscriptionOfferInfo `json:"offers,omitempty"`
	NextPageToken string                  `json:"nextPageToken,omitempty"`
}

type ProductPurchaseInfo struct {
	OrderID              string `json:"orderId,omitempty"`
	ProductID            string `json:"productId,omitempty"`
	PurchaseToken        string `json:"purchaseToken,omitempty"`
	PurchaseState        int64  `json:"purchaseState,omitempty"`
	AcknowledgementState int64  `json:"acknowledgementState,omitempty"`
	ConsumptionState     int64  `json:"consumptionState,omitempty"`
	PurchaseTimeMillis   int64  `json:"purchaseTimeMillis,omitempty"`
	RegionCode           string `json:"regionCode,omitempty"`
}

type SubscriptionPurchaseInfo struct {
	Kind                 string `json:"kind,omitempty"`
	LatestOrderID        string `json:"latestOrderId,omitempty"`
	SubscriptionState    string `json:"subscriptionState,omitempty"`
	AcknowledgementState string `json:"acknowledgementState,omitempty"`
	RegionCode           string `json:"regionCode,omitempty"`
	StartTime            string `json:"startTime,omitempty"`
	LineItemCount        int    `json:"lineItemCount,omitempty"`
}

type VoidedPurchaseInfo struct {
	OrderID            string `json:"orderId,omitempty"`
	PurchaseToken      string `json:"purchaseToken,omitempty"`
	PurchaseTimeMillis int64  `json:"purchaseTimeMillis,omitempty"`
	VoidedTimeMillis   int64  `json:"voidedTimeMillis,omitempty"`
	VoidedReason       int64  `json:"voidedReason,omitempty"`
	VoidedSource       int64  `json:"voidedSource,omitempty"`
	VoidedQuantity     int64  `json:"voidedQuantity,omitempty"`
}

type VoidedPurchasesQuery struct {
	MaxResults                        int64
	StartIndex                        int64
	Token                             string
	StartTime                         int64
	EndTime                           int64
	Type                              int64
	IncludeQuantityBasedPartialRefund bool
	Paginate                          bool
}

type VoidedPurchasesListInfo struct {
	VoidedPurchases []VoidedPurchaseInfo `json:"voidedPurchases,omitempty"`
	NextToken       string               `json:"nextToken,omitempty"`
}

type TrackInfo struct {
	Name     string             `json:"name"`
	Releases []TrackReleaseInfo `json:"releases,omitempty"`
}

type TrackReleaseInfo struct {
	Name           string  `json:"name,omitempty"`
	Status         string  `json:"status,omitempty"`
	UserFraction   float64 `json:"userFraction,omitempty"`
	VersionCodes   []int64 `json:"versionCodes,omitempty"`
	UpdatePriority int64   `json:"updatePriority,omitempty"`
}

type TrackUpdate struct {
	Status         string
	ReleaseName    string
	UserFraction   float64
	VersionCodes   []int64
	UpdatePriority int64
}

type BundleInfo struct {
	VersionCode int64  `json:"versionCode,omitempty"`
	SHA1        string `json:"sha1,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

type APKInfo struct {
	VersionCode int64  `json:"versionCode,omitempty"`
	SHA1        string `json:"sha1,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

type InternalSharingArtifactInfo struct {
	DownloadURL            string `json:"downloadUrl,omitempty"`
	CertificateFingerprint string `json:"certificateFingerprint,omitempty"`
	SHA256                 string `json:"sha256,omitempty"`
}

type DeobfuscationFileInfo struct {
	SymbolType string `json:"symbolType,omitempty"`
}
