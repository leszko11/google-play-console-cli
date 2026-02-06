package shared

import "github.com/peterbourgon/ff/v3/ffcli"

func DefaultUsageFunc(c *ffcli.Command) string {
	return ffcli.DefaultUsageFunc(c)
}
