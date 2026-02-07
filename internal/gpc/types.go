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
