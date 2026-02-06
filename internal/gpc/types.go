package gpc

type CredentialInput struct {
	ServiceAccountPath string
	ServiceAccountJSON []byte
}

type AppInfo struct {
	PackageName string `json:"packageName"`
}
