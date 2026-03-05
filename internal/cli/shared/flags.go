package shared

import (
	"flag"
	"time"
)

const EnvServiceAccountPath = "GPC_SERVICE_ACCOUNT_PATH"
const EnvDefaultOutput = "GPC_DEFAULT_OUTPUT"
const EnvStrictAuth = "GPC_STRICT_AUTH"

type GlobalFlags struct {
	PackageName     string
	ServiceAccount  string
	Profile         string
	Output          string
	Timeout         time.Duration
	UploadTimeout   time.Duration
	Debug           string
	Pretty          bool
	Paginate        bool
	StrictAuth      bool
	BootstrapAssist bool
}

var boundGlobalFlags *GlobalFlags

func BindGlobalFlags(fs *flag.FlagSet, cfg *GlobalFlags) {
	if cfg == nil {
		cfg = &GlobalFlags{}
	}
	boundGlobalFlags = cfg

	fs.StringVar(&cfg.PackageName, "package-name", "", "App package name")
	fs.StringVar(&cfg.ServiceAccount, "service-account", "", "Path to service account JSON")
	fs.StringVar(&cfg.Profile, "profile", "", "Authentication profile override")
	fs.StringVar(&cfg.Output, "output", "", "Output format override: json, table, markdown")
	fs.DurationVar(&cfg.Timeout, "timeout", 0, "Request timeout")
	fs.DurationVar(&cfg.UploadTimeout, "upload-timeout", 0, "Upload request timeout")
	fs.StringVar(&cfg.Debug, "debug", "", "Enable debug logging")
	fs.BoolVar(&cfg.Pretty, "pretty", false, "Pretty print JSON output")
	fs.BoolVar(&cfg.Paginate, "paginate", false, "Fetch all paginated API responses")
	fs.BoolVar(&cfg.StrictAuth, "strict-auth", false, "Fail when credentials are resolved from multiple sources")
	fs.BoolVar(&cfg.BootstrapAssist, "bootstrap-assist", false, "Enable interactive bootstrap build assistance")
}

func ActiveGlobalFlags() GlobalFlags {
	if boundGlobalFlags == nil {
		return GlobalFlags{}
	}
	return *boundGlobalFlags
}
