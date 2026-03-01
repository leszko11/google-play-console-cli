package cmd

var (
	// Version is the semantic version, injected by ldflags at build time.
	Version = "dev"
	// Commit is the git commit hash, injected by ldflags at build time.
	Commit = "unknown"
	// Date is the build date, injected by ldflags at build time.
	Date = "unknown"
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func versionPayload() versionInfo {
	return versionInfo{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}
