package shared

import (
	"flag"
	"time"
)

type GlobalFlags struct {
	PackageName    string
	ServiceAccount string
	Output         string
	Timeout        time.Duration
	UploadTimeout  time.Duration
	Debug          string
	Pretty         bool
	Paginate       bool
}

func BindGlobalFlags(fs *flag.FlagSet, cfg *GlobalFlags) {
	fs.StringVar(&cfg.PackageName, "package-name", "", "App package name")
	fs.StringVar(&cfg.ServiceAccount, "service-account", "", "Path to service account JSON")
	fs.StringVar(&cfg.Output, "output", "json", "Output format: json, table, markdown")
	fs.DurationVar(&cfg.Timeout, "timeout", 0, "Request timeout")
	fs.DurationVar(&cfg.UploadTimeout, "upload-timeout", 0, "Upload request timeout")
	fs.StringVar(&cfg.Debug, "debug", "", "Enable debug logging")
	fs.BoolVar(&cfg.Pretty, "pretty", false, "Pretty print JSON output")
	fs.BoolVar(&cfg.Paginate, "paginate", false, "Fetch all paginated API responses")
}
