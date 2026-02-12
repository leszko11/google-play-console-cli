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
