package appinit

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/leszko11/google-play-console-cli/internal/gpc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
	"gopkg.in/yaml.v3"
)

var exportHTTPClient = http.DefaultClient

var exportImageTypes = []string{
	"featureGraphic",
	"icon",
	"phoneScreenshots",
	"promoGraphic",
	"sevenInchScreenshots",
	"tenInchScreenshots",
	"tvBanner",
	"tvScreenshots",
	"wearScreenshots",
}

var gppImageTypeDirs = map[string]string{
	"featureGraphic":       "feature-graphic",
	"icon":                 "icon",
	"phoneScreenshots":     "phone-screenshots",
	"promoGraphic":         "promo-graphic",
	"sevenInchScreenshots": "seven-inch-screenshots",
	"tenInchScreenshots":   "ten-inch-screenshots",
	"tvBanner":             "tv-banner",
	"tvScreenshots":        "tv-screenshots",
	"wearScreenshots":      "wear-screenshots",
}

type exportOptions struct {
	PackageName        string
	Dir                string
	Include            string
	Tracks             string
	Layout             string
	SkipImages         bool
	WriteProjectConfig bool
}

type exportResult struct {
	PackageName string            `json:"packageName"`
	Dir         string            `json:"dir"`
	Layout      string            `json:"layout"`
	Sections    []string          `json:"sections"`
	Counts      map[string]int    `json:"counts,omitempty"`
	Files       map[string]string `json:"files,omitempty"`
}

type exportPaths struct {
	ListingDir       string
	ChangelogDir     string
	ProductsDir      string
	SubscriptionsDir string
	GPPPlayDir       string
}

type exportedSubscription struct {
	Subscription   *androidpublisher.Subscription `json:"subscription"`
	RegionsVersion string                         `json:"regionsVersion,omitempty"`
}

func newExportCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var opts exportOptions
	fs.StringVar(&opts.PackageName, "package-name", "", "Package name")
	fs.StringVar(&opts.Dir, "dir", "", "Export directory")
	fs.StringVar(&opts.Include, "include", "", "Comma-separated sections: app-details,listing,changelog,products,subscriptions")
	fs.StringVar(&opts.Tracks, "tracks", "", "Comma-separated tracks to export changelogs from")
	fs.StringVar(&opts.Layout, "layout", "gpc", "Export layout: gpc or gpp")
	fs.BoolVar(&opts.SkipImages, "skip-images", false, "Skip downloading listing images")
	fs.BoolVar(&opts.WriteProjectConfig, "write-project-config", false, "Write .gpc.yaml with project-local defaults")

	return &ffcli.Command{
		Name:      "export",
		ShortHelp: "Export existing Play store state into local files",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			opts, sections, tracks, err := validateExportOptions(opts)
			if err != nil {
				return err
			}
			client, requestCtx, cancel, err := shared.BuildClient[Client](ctx, shared.BuildClientDeps[Client]{
				LoadConfig: deps.LoadConfig,
				LookupEnv:  deps.LookupEnv,
				NewClient:  deps.NewClient,
			})
			if err != nil {
				return err
			}
			defer cancel()
			return runExport(ctx, requestCtx, client, deps.Stdout, opts, sections, tracks)
		},
	}
}

func validateExportOptions(opts exportOptions) (exportOptions, map[string]struct{}, map[string]struct{}, error) {
	pkg, err := shared.ResolvePackageName(opts.PackageName)
	if err != nil {
		return exportOptions{}, nil, nil, err
	}
	opts.PackageName = pkg
	opts.Dir = strings.TrimSpace(opts.Dir)
	if opts.Dir == "" {
		return exportOptions{}, nil, nil, shared.UsageErrorf("--dir is required")
	}
	opts.Layout = strings.ToLower(strings.TrimSpace(opts.Layout))
	if opts.Layout == "" {
		opts.Layout = "gpc"
	}
	if opts.Layout != "gpc" && opts.Layout != "gpp" {
		return exportOptions{}, nil, nil, shared.UsageErrorf("--layout must be one of: gpc, gpp")
	}
	if opts.Layout == "gpp" && opts.WriteProjectConfig {
		return exportOptions{}, nil, nil, shared.UsageErrorf("--write-project-config is only supported with --layout gpc")
	}

	sections := map[string]struct{}{
		"app-details":   {},
		"listing":       {},
		"changelog":     {},
		"products":      {},
		"subscriptions": {},
	}
	if raw := strings.TrimSpace(opts.Include); raw != "" {
		sections = make(map[string]struct{})
		for _, part := range strings.Split(raw, ",") {
			name := strings.ToLower(strings.TrimSpace(part))
			switch name {
			case "app-details", "listing", "changelog", "products", "subscriptions":
				sections[name] = struct{}{}
			default:
				return exportOptions{}, nil, nil, shared.UsageErrorf("unsupported --include section %q", name)
			}
		}
	}

	tracks := map[string]struct{}{}
	if raw := strings.TrimSpace(opts.Tracks); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			track := strings.TrimSpace(part)
			if track != "" {
				tracks[track] = struct{}{}
			}
		}
	}
	return opts, sections, tracks, nil
}

func runExport(parentCtx, requestCtx context.Context, client Client, out io.Writer, opts exportOptions, sections map[string]struct{}, tracks map[string]struct{}) error {
	edit, err := client.CreateEdit(requestCtx, opts.PackageName)
	if err != nil {
		return fmt.Errorf("failed to create edit for export: %w", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := shared.ContextWithTimeout(parentCtx, shared.ActiveGlobalFlags().Timeout)
		_ = client.DeleteEdit(cleanupCtx, opts.PackageName, edit.ID)
		cleanupCancel()
	}()

	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}

	paths := resolveExportPaths(opts.Dir, opts.Layout)
	result := exportResult{
		PackageName: opts.PackageName,
		Dir:         opts.Dir,
		Layout:      opts.Layout,
		Sections:    make([]string, 0, len(sections)),
		Counts:      map[string]int{},
		Files:       map[string]string{},
	}

	var details gpc.AppDetailsInfo
	if _, ok := sections["app-details"]; ok {
		details, err = client.GetAppDetails(requestCtx, opts.PackageName, edit.ID)
		if err != nil {
			return fmt.Errorf("failed to read app details: %w", err)
		}
		if err := exportAppDetails(paths, opts.Layout, details); err != nil {
			return err
		}
		result.Sections = append(result.Sections, "app-details")
		result.Files["appinitManifest"] = filepath.Join(opts.Dir, "appinit.yaml")
	}

	if _, ok := sections["listing"]; ok {
		listings, err := client.ListListings(requestCtx, opts.PackageName, edit.ID)
		if err != nil {
			return fmt.Errorf("failed to list listings: %w", err)
		}
		if err := exportListings(requestCtx, client, opts.PackageName, edit.ID, paths, opts.Layout, opts.SkipImages, listings); err != nil {
			return err
		}
		result.Sections = append(result.Sections, "listing")
		result.Counts["listingLocales"] = len(listings)
	}

	if _, ok := sections["changelog"]; ok {
		count, err := exportChangelogs(requestCtx, client, opts.PackageName, edit.ID, paths, opts.Layout, tracks)
		if err != nil {
			return err
		}
		result.Sections = append(result.Sections, "changelog")
		result.Counts["changelogTracks"] = count
	}

	if _, ok := sections["products"]; ok {
		count, err := exportProducts(requestCtx, client, opts.PackageName, paths)
		if err != nil {
			return err
		}
		result.Sections = append(result.Sections, "products")
		result.Counts["products"] = count
	}

	if _, ok := sections["subscriptions"]; ok {
		count, err := exportSubscriptions(requestCtx, client, opts.PackageName, paths)
		if err != nil {
			return err
		}
		result.Sections = append(result.Sections, "subscriptions")
		result.Counts["subscriptions"] = count
	}

	if err := writeExportAppInitManifest(filepath.Join(opts.Dir, "appinit.yaml"), opts.Layout, paths, details, sections); err != nil {
		return err
	}
	if opts.WriteProjectConfig {
		if err := writeProjectConfig(filepath.Join(opts.Dir, ".gpc.yaml"), opts.PackageName, paths); err != nil {
			return err
		}
		result.Files["projectConfig"] = filepath.Join(opts.Dir, ".gpc.yaml")
	}

	sort.Strings(result.Sections)
	return shared.WriteJSON(out, result)
}

func resolveExportPaths(root, layout string) exportPaths {
	if layout == "gpp" {
		playDir := filepath.Join(root, "play")
		return exportPaths{
			ListingDir:       filepath.Join(playDir, "listings"),
			ChangelogDir:     filepath.Join(playDir, "release-notes"),
			ProductsDir:      filepath.Join(playDir, "products"),
			SubscriptionsDir: filepath.Join(playDir, "subscriptions"),
			GPPPlayDir:       playDir,
		}
	}
	return exportPaths{
		ListingDir:       filepath.Join(root, "listing"),
		ChangelogDir:     filepath.Join(root, "changelog"),
		ProductsDir:      filepath.Join(root, "products"),
		SubscriptionsDir: filepath.Join(root, "subscriptions"),
	}
}

func exportAppDetails(paths exportPaths, layout string, details gpc.AppDetailsInfo) error {
	if layout != "gpp" {
		return nil
	}
	type fileValue struct {
		name  string
		value string
	}
	for _, item := range []fileValue{
		{name: "default-language.txt", value: details.DefaultLanguage},
		{name: "contact-email.txt", value: details.ContactEmail},
		{name: "contact-phone.txt", value: details.ContactPhone},
		{name: "contact-website.txt", value: details.ContactWebsite},
	} {
		if strings.TrimSpace(item.value) == "" {
			continue
		}
		if err := writeTextFile(filepath.Join(paths.GPPPlayDir, item.name), item.value); err != nil {
			return err
		}
	}
	return nil
}

func exportListings(ctx context.Context, client Client, packageName, editID string, paths exportPaths, layout string, skipImages bool, listings []gpc.ListingInfo) error {
	for _, item := range listings {
		localeDir := filepath.Join(paths.ListingDir, item.Language)
		if err := writeTextFile(filepath.Join(localeDir, "title.txt"), item.Title); err != nil {
			return err
		}
		if err := writeTextFile(filepath.Join(localeDir, "short-description.txt"), item.ShortDescription); err != nil {
			return err
		}
		if err := writeTextFile(filepath.Join(localeDir, "full-description.txt"), item.FullDescription); err != nil {
			return err
		}
		if skipImages {
			continue
		}
		for _, imageType := range exportImageTypes {
			images, err := client.ListImages(ctx, packageName, editID, item.Language, imageType)
			if err != nil {
				return fmt.Errorf("failed to list images for %s/%s: %w", item.Language, imageType, err)
			}
			for index, image := range images {
				if err := downloadImage(image.URL, exportImagePath(paths, layout, item.Language, imageType, index)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func exportChangelogs(ctx context.Context, client Client, packageName, editID string, paths exportPaths, layout string, tracks map[string]struct{}) (int, error) {
	remoteTracks, err := client.ListTracks(ctx, packageName, editID)
	if err != nil {
		return 0, fmt.Errorf("failed to list tracks: %w", err)
	}
	exported := 0
	for _, track := range remoteTracks {
		if len(tracks) > 0 {
			if _, ok := tracks[track.Name]; !ok {
				continue
			}
		}
		release := selectTrackForExport(track)
		if len(release.ReleaseNotes) == 0 {
			continue
		}
		for _, note := range release.ReleaseNotes {
			target := filepath.Join(paths.ChangelogDir, track.Name, note.Language+".txt")
			if layout == "gpp" {
				target = filepath.Join(paths.ChangelogDir, note.Language, track.Name+".txt")
			}
			if err := writeTextFile(target, note.Text); err != nil {
				return 0, err
			}
		}
		exported++
	}
	return exported, nil
}

func selectTrackForExport(track gpc.TrackInfo) gpc.TrackReleaseInfo {
	for _, release := range track.Releases {
		if len(release.ReleaseNotes) > 0 {
			return release
		}
	}
	if len(track.Releases) == 0 {
		return gpc.TrackReleaseInfo{}
	}
	return track.Releases[0]
}

func exportProducts(ctx context.Context, client Client, packageName string, paths exportPaths) (int, error) {
	result, err := client.ListOneTimeProducts(ctx, packageName, 0, "", true)
	if err != nil {
		return 0, fmt.Errorf("failed to list products: %w", err)
	}
	sort.Slice(result.Products, func(i, j int) bool { return result.Products[i].ProductID < result.Products[j].ProductID })
	for _, item := range result.Products {
		product, err := client.GetOneTimeProductResource(ctx, packageName, item.ProductID)
		if err != nil {
			return 0, fmt.Errorf("failed to read product %q: %w", item.ProductID, err)
		}
		if err := writeJSONFile(filepath.Join(paths.ProductsDir, item.ProductID+".json"), product); err != nil {
			return 0, err
		}
	}
	return len(result.Products), nil
}

func exportSubscriptions(ctx context.Context, client Client, packageName string, paths exportPaths) (int, error) {
	result, err := client.ListSubscriptions(ctx, packageName, 0, "", true)
	if err != nil {
		return 0, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	regionsVersion, err := client.GetLatestRegionsVersion(ctx, packageName)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve regions version: %w", err)
	}
	sort.Slice(result.Subscriptions, func(i, j int) bool { return result.Subscriptions[i].ProductID < result.Subscriptions[j].ProductID })
	for _, item := range result.Subscriptions {
		subscription, err := client.GetSubscriptionResource(ctx, packageName, item.ProductID)
		if err != nil {
			return 0, fmt.Errorf("failed to read subscription %q: %w", item.ProductID, err)
		}
		if err := writeJSONFile(filepath.Join(paths.SubscriptionsDir, item.ProductID+".json"), exportedSubscription{
			Subscription:   subscription,
			RegionsVersion: regionsVersion,
		}); err != nil {
			return 0, err
		}
	}
	return len(result.Subscriptions), nil
}

func writeExportAppInitManifest(path, layout string, paths exportPaths, details gpc.AppDetailsInfo, sections map[string]struct{}) error {
	manifest := appInitManifest{}
	if _, ok := sections["app-details"]; ok {
		manifest.AppDetails = &appDetailsSection{
			DefaultLanguage: details.DefaultLanguage,
			ContactEmail:    details.ContactEmail,
			ContactPhone:    details.ContactPhone,
			ContactWebsite:  details.ContactWebsite,
		}
	}
	if _, ok := sections["listing"]; ok {
		dir := "./listing"
		if layout == "gpp" {
			dir = "./play/listings"
		}
		manifest.Listing = &listingSection{Dir: dir}
	}
	raw, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal appinit manifest: %w", err)
	}
	return writeBytes(path, raw)
}

func writeProjectConfig(path, packageName string, paths exportPaths) error {
	cfg := map[string]any{
		"package-name":     packageName,
		"listing-dir":      "./listing",
		"changelog-dir":    "./changelog",
		"appinit-manifest": "./appinit.yaml",
	}
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal project config: %w", err)
	}
	return writeBytes(path, raw)
}

func exportImagePath(paths exportPaths, layout, locale, imageType string, index int) string {
	if layout == "gpp" {
		name := gppImageTypeDirs[imageType]
		filename := fmt.Sprintf("%02d.png", index+1)
		if imageType == "icon" || imageType == "featureGraphic" || imageType == "promoGraphic" || imageType == "tvBanner" {
			filename = name + ".png"
			return filepath.Join(paths.ListingDir, locale, "graphics", filename)
		}
		return filepath.Join(paths.ListingDir, locale, "graphics", name, filename)
	}
	filename := fmt.Sprintf("%02d.png", index+1)
	if imageType == "icon" || imageType == "featureGraphic" || imageType == "promoGraphic" || imageType == "tvBanner" {
		return filepath.Join(paths.ListingDir, locale, "images", imageType+".png")
	}
	return filepath.Join(paths.ListingDir, locale, "images", imageType, filename)
}

func downloadImage(url, path string) error {
	if strings.TrimSpace(url) == "" {
		return nil
	}
	resp, err := exportHTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("download image %q: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download image %q: unexpected status %s", url, resp.Status)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("download image %q: %w", url, err)
	}
	return writeBytes(path, raw)
}

func writeTextFile(path, value string) error {
	return writeBytes(path, []byte(strings.TrimSpace(value)))
}

func writeJSONFile(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	raw = append(raw, '\n')
	return writeBytes(path, raw)
}

func writeBytes(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
