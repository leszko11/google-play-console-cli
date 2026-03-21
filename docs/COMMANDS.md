# Command Reference Guide

This file is generated from live CLI help output.
For authoritative command behavior, use:

```bash
gpc --help
gpc <command> --help
gpc <command> <subcommand> --help
```

To regenerate:

```bash
make generate-command-docs
```

## Command Paths

- `gpc auth`
- `gpc auth init`
- `gpc auth status`
- `gpc auth profiles`
- `gpc auth profiles list`
- `gpc auth switch`
- `gpc auth logout`
- `gpc bootstrap`
- `gpc appinit`
- `gpc appinit export`
- `gpc apps`
- `gpc apps list`
- `gpc apps get`
- `gpc apps data-safety`
- `gpc apps add-package`
- `gpc apps remove-package`
- `gpc app-recoveries`
- `gpc app-recoveries list`
- `gpc app-recoveries create`
- `gpc app-recoveries add-targeting`
- `gpc app-recoveries cancel`
- `gpc app-recoveries deploy`
- `gpc changelog`
- `gpc changelog sync`
- `gpc custom-apps`
- `gpc custom-apps create`
- `gpc edits`
- `gpc edits create`
- `gpc edits get`
- `gpc edits validate`
- `gpc edits commit`
- `gpc edits delete`
- `gpc edits details`
- `gpc edits details get`
- `gpc edits details update`
- `gpc edits testers`
- `gpc edits testers get`
- `gpc edits testers update`
- `gpc edits country-availability`
- `gpc edits country-availability get`
- `gpc edits listings`
- `gpc edits listings list`
- `gpc edits listings get`
- `gpc edits listings update`
- `gpc edits listings batch-update`
- `gpc edits listings delete`
- `gpc edits listings delete-all`
- `gpc edits images`
- `gpc edits images list`
- `gpc edits images upload`
- `gpc edits images upload-dir`
- `gpc edits images delete`
- `gpc edits images delete-all`
- `gpc edits expansion-files`
- `gpc edits expansion-files get`
- `gpc edits expansion-files patch`
- `gpc edits expansion-files update`
- `gpc edits expansion-files upload`
- `gpc tracks`
- `gpc tracks list`
- `gpc tracks get`
- `gpc tracks create`
- `gpc tracks patch`
- `gpc tracks update`
- `gpc tracks promote`
- `gpc apks`
- `gpc apks list`
- `gpc apks upload`
- `gpc apks add-externally-hosted`
- `gpc bundles`
- `gpc bundles list`
- `gpc bundles upload`
- `gpc bundles wait`
- `gpc deobfuscation`
- `gpc deobfuscation upload`
- `gpc deploy`
- `gpc diff`
- `gpc diff listing`
- `gpc diff track`
- `gpc doctor`
- `gpc e2e`
- `gpc e2e fixtures`
- `gpc e2e fixtures status`
- `gpc release`
- `gpc release verify`
- `gpc release alpha`
- `gpc release full`
- `gpc release promote`
- `gpc rollback`
- `gpc screenshots`
- `gpc screenshots sync`
- `gpc setup`
- `gpc status`
- `gpc reviews`
- `gpc reviews list`
- `gpc reviews get`
- `gpc reviews triage`
- `gpc reviews reply`
- `gpc reports`
- `gpc reports apps`
- `gpc reports apps list`
- `gpc reports anomalies`
- `gpc reports anomalies list`
- `gpc reports errors`
- `gpc reports errors counts`
- `gpc reports errors counts get`
- `gpc reports errors counts query`
- `gpc reports errors issues`
- `gpc reports errors issues list`
- `gpc reports errors reports`
- `gpc reports errors reports list`
- `gpc reports financial`
- `gpc reports financial list`
- `gpc reports financial get`
- `gpc reports summary`
- `gpc reports vitals`
- `gpc reports vitals get`
- `gpc reports vitals query`
- `gpc orders`
- `gpc orders get`
- `gpc orders batch-get`
- `gpc orders refund`
- `gpc external-transactions`
- `gpc external-transactions get`
- `gpc external-transactions create`
- `gpc external-transactions refund`
- `gpc device-tier-configs`
- `gpc device-tier-configs list`
- `gpc device-tier-configs get`
- `gpc device-tier-configs create`
- `gpc system-apks`
- `gpc system-apks list`
- `gpc system-apks get`
- `gpc system-apks create`
- `gpc system-apks download`
- `gpc generated-apks`
- `gpc generated-apks list`
- `gpc generated-apks download`
- `gpc games`
- `gpc games achievements`
- `gpc games achievements list`
- `gpc games events`
- `gpc games events list`
- `gpc games leaderboards`
- `gpc games leaderboards list`
- `gpc games leaderboards get`
- `gpc subscriptions`
- `gpc subscriptions list`
- `gpc subscriptions get`
- `gpc subscriptions sync`
- `gpc subscriptions batch-get`
- `gpc subscriptions create`
- `gpc subscriptions batch-update`
- `gpc subscriptions update`
- `gpc subscriptions delete`
- `gpc subscriptions archive`
- `gpc subscriptions base-plans`
- `gpc subscriptions base-plans activate`
- `gpc subscriptions base-plans deactivate`
- `gpc subscriptions base-plans batch-update-states`
- `gpc subscriptions base-plans delete`
- `gpc subscriptions base-plans migrate-prices`
- `gpc subscriptions base-plans batch-migrate-prices`
- `gpc subscriptions offers`
- `gpc subscriptions offers list`
- `gpc subscriptions offers get`
- `gpc subscriptions offers batch-get`
- `gpc subscriptions offers batch-update`
- `gpc subscriptions offers batch-update-states`
- `gpc subscriptions offers activate`
- `gpc subscriptions offers deactivate`
- `gpc subscriptions offers create`
- `gpc subscriptions offers update`
- `gpc subscriptions offers delete`
- `gpc monetization`
- `gpc monetization regions`
- `gpc monetization setup`
- `gpc monetization sync`
- `gpc migrate`
- `gpc migrate fastlane`
- `gpc migrate fastlane import`
- `gpc notify`
- `gpc notify webhook`
- `gpc notify slack`
- `gpc notify discord`
- `gpc products`
- `gpc products list`
- `gpc products get`
- `gpc products sync`
- `gpc products batch-get`
- `gpc products batch-update`
- `gpc products batch-delete`
- `gpc products create`
- `gpc products update`
- `gpc products delete`
- `gpc products offers`
- `gpc products offers list`
- `gpc products offers batch-get`
- `gpc products offers batch-update`
- `gpc products offers batch-update-states`
- `gpc products offers batch-delete`
- `gpc products offers activate`
- `gpc products offers deactivate`
- `gpc products offers cancel`
- `gpc products purchase-options`
- `gpc products purchase-options activate`
- `gpc products purchase-options deactivate`
- `gpc products purchase-options delete`
- `gpc publish`
- `gpc publish alpha`
- `gpc publish production`
- `gpc iap`
- `gpc iap list`
- `gpc iap get`
- `gpc iap batch-get`
- `gpc iap create`
- `gpc iap update`
- `gpc iap replace`
- `gpc iap batch-update`
- `gpc iap batch-delete`
- `gpc iap delete`
- `gpc listing`
- `gpc listing sync`
- `gpc purchases`
- `gpc purchases products`
- `gpc purchases products get`
- `gpc purchases products acknowledge`
- `gpc purchases products consume`
- `gpc purchases products-v2`
- `gpc purchases products-v2 get`
- `gpc purchases subscriptions`
- `gpc purchases subscriptions get`
- `gpc purchases subscriptions cancel`
- `gpc purchases subscriptions defer`
- `gpc purchases subscriptions revoke`
- `gpc purchases subscriptions-legacy`
- `gpc purchases subscriptions-legacy get`
- `gpc purchases subscriptions-legacy acknowledge`
- `gpc purchases subscriptions-legacy cancel`
- `gpc purchases subscriptions-legacy defer`
- `gpc purchases subscriptions-legacy refund`
- `gpc purchases subscriptions-legacy revoke`
- `gpc purchases voided`
- `gpc purchases voided list`
- `gpc users`
- `gpc users list`
- `gpc users create`
- `gpc users update`
- `gpc users delete`
- `gpc validate`
- `gpc workflow`
- `gpc workflow run`
- `gpc grants`
- `gpc grants create`
- `gpc grants update`
- `gpc grants delete`
- `gpc internal-sharing`
- `gpc internal-sharing upload`
- `gpc integrity`
- `gpc integrity decode`
- `gpc update`
- `gpc completion`
- `gpc completion bash`
- `gpc completion zsh`
- `gpc completion fish`
- `gpc completion values`

## `gpc --help`

```text
DESCRIPTION
  Google Play Console CLI

USAGE
  gpc [flags] <command>

SUBCOMMANDS
  auth                   Manage authentication profiles
  bootstrap              Export current Play state into a local bootstrap workspace
  appinit                Bootstrap app store presence from a manifest
  apps                   App visibility and metadata commands
  app-recoveries         Manage Play app recovery actions
  changelog              Release changelog workflows
  custom-apps            Create custom private apps for managed Play organizations
  edits                  Manage Google Play edit transactions
  tracks                 Manage release tracks inside an edit
  apks                   Manage APK uploads in an edit
  bundles                Manage Android App Bundles in an edit
  deobfuscation          Manage deobfuscation files in an edit
  deploy                 Upload artifact and publish to a track in one flow
  diff                   Compare live Play state against local listing or track drafts
  doctor                 Run read-only diagnostics for auth, package access, and e2e fixtures
  e2e                    E2E fixture and smoke-testing helpers
  release                Release workflows for staged Google Play deploys
  rollback               Halt the active staged rollout on a track
  screenshots            Manage screenshot-only sync workflows
  setup                  Provision auth and optional bootstrap workspace for gpc
  status                 Summarize tracks and recent review health for an app
  reviews                Read and reply to Play Store reviews
  reports                Google Play Developer Reporting commands
  orders                 Inspect and refund Play orders
  external-transactions  Report and refund external transactions
  device-tier-configs    Manage application device tier configs
  system-apks            Manage generated system APK variants
  generated-apks         List and download APKs generated from bundles
  games                  Inspect Play Games Services achievements, events, and leaderboards
  subscriptions          Manage monetization subscriptions
  monetization           Monetization workflows
  migrate                Import or transform metadata from other tool layouts
  notify                 Notification delivery helpers
  products               Manage monetization one-time products
  publish                Common publish flows with track presets
  iap                    Manage legacy in-app products
  listing                Store listing workflows
  purchases              Manage one-time and subscription purchases
  users                  Manage Play Console account users
  validate               Run pre-submission validation checks
  workflow               Run declarative gpc workflows from .gpc/workflow.yml
  grants                 Manage per-app user grants
  internal-sharing       Upload artifacts for internal app sharing
  integrity              Decode and inspect Play Integrity tokens
  update                 Check for and install newer gpc releases
  completion             Generate shell completion script

FLAGS
  -bootstrap-assist=false  Enable interactive bootstrap build assistance
  -debug string            Enable debug logging
  -fields string           Comma-separated JSON field projection
  -output string           Output format override: json, table, markdown, yaml
  -package-name string     App package name
  -paginate=false          Fetch all paginated API responses
  -pretty=false            Pretty print JSON output
  -profile string          Authentication profile override
  -service-account string  Path to service account JSON
  -strict-auth=false       Fail when credentials are resolved from multiple sources
  -timeout 0s              Request timeout
  -upload-timeout 0s       Upload request timeout
  -version=false           Show build version information
```

## `gpc auth --help`

```text
DESCRIPTION
  Manage authentication profiles

USAGE
  auth

SUBCOMMANDS
  init      Initialize authentication profile
  status    Show authentication status
  profiles  Manage authentication profiles
  switch    Switch active authentication profile
  logout    Log out current profile
```

## `gpc auth init --help`

```text
DESCRIPTION
  Initialize authentication profile

USAGE
  init

FLAGS
  -developer-id string        Optional developer account ID (numeric or developers/<id>)
  -package-name string        Verify package access for this package
  -profile default            Auth profile name
  -prompt-developer-id=false  Prompt for developer ID when missing (interactive terminals only)
  -service-account string     Path to service account JSON
```

## `gpc auth status --help`

```text
DESCRIPTION
  Show authentication status

USAGE
  status

FLAGS
  -output string  Output format: json, table, markdown, yaml
```

## `gpc auth profiles --help`

```text
DESCRIPTION
  Manage authentication profiles

USAGE
  profiles

SUBCOMMANDS
  list  List authentication profiles
```

## `gpc auth profiles list --help`

```text
DESCRIPTION
  List authentication profiles

USAGE
  list

FLAGS
  -output string  Output format: json, table, markdown, yaml, csv, tsv
```

## `gpc auth switch --help`

```text
DESCRIPTION
  Switch active authentication profile

USAGE
  switch

FLAGS
  -profile string  Profile name to activate
```

## `gpc auth logout --help`

```text
DESCRIPTION
  Log out current profile

USAGE
  logout

FLAGS
  -all=false       Remove all stored profiles and credentials
  -profile string  Profile name to remove (defaults to selected active profile)
```

## `gpc bootstrap --help`

```text
DESCRIPTION
  Export current Play state into a local bootstrap workspace

USAGE
  bootstrap

FLAGS
  -dir string                  Bootstrap directory
  -package-name string         Package name
  -skip-images=false           Skip downloading listing images
  -tracks string               Comma-separated tracks to export changelogs from
  -write-project-config=false  Write .gpc.yaml with project-local defaults
```

## `gpc appinit --help`

```text
DESCRIPTION
  Bootstrap app store presence from a manifest

USAGE
  appinit

SUBCOMMANDS
  export  Export existing Play store state into local files

FLAGS
  -confirm=false        Confirm applying bootstrap changes (required unless --dry-run)
  -dry-run=false        Validate all sections and use dry-run flows where supported
  -manifest string      Path to app bootstrap manifest (.json/.yaml/.yml)
  -package-name string  Package name
```

## `gpc appinit export --help`

```text
DESCRIPTION
  Export existing Play store state into local files

USAGE
  export

FLAGS
  -dir string                  Export directory
  -include string              Comma-separated sections: app-details,listing,changelog,products,subscriptions
  -layout gpc                  Export layout: gpc or gpp
  -package-name string         Package name
  -skip-images=false           Skip downloading listing images
  -tracks string               Comma-separated tracks to export changelogs from
  -write-project-config=false  Write .gpc.yaml with project-local defaults
```

## `gpc apps --help`

```text
DESCRIPTION
  App visibility and metadata commands

USAGE
  apps

SUBCOMMANDS
  list            List configured packages
  get             Get app details for a package
  data-safety     Write the Data Safety CSV declaration for an app
  add-package     Add package to local app list
  remove-package  Remove package from local app list
```

## `gpc apps list --help`

```text
DESCRIPTION
  List configured packages

USAGE
  list

FLAGS
  -output string  Output format: json, table, markdown, yaml, csv, tsv, minimal
  -verify=false   Verify API access for each configured package
```

## `gpc apps get --help`

```text
DESCRIPTION
  Get app details for a package

USAGE
  get

FLAGS
  -output string        Output format: json, table, markdown, yaml, csv, tsv
  -package-name string  Package name
```

## `gpc apps data-safety --help`

```text
DESCRIPTION
  Write the Data Safety CSV declaration for an app

USAGE
  data-safety

FLAGS
  -input string         Path to Data Safety CSV payload (use - for stdin)
  -package-name string  Package name
```

## `gpc apps add-package --help`

```text
DESCRIPTION
  Add package to local app list

USAGE
  add-package

FLAGS
  -package-name string  Package name to store for list/verify flows
```

## `gpc apps remove-package --help`

```text
DESCRIPTION
  Remove package from local app list

USAGE
  remove-package

FLAGS
  -package-name string  Package name to remove from local app list
```

## `gpc app-recoveries --help`

```text
DESCRIPTION
  Manage Play app recovery actions

USAGE
  app-recoveries

SUBCOMMANDS
  list           List app recovery actions for one version code
  create         Create a draft app recovery action
  add-targeting  Add targeting to an app recovery action
  cancel         Cancel an app recovery action
  deploy         Deploy an app recovery action
```

## `gpc app-recoveries list --help`

```text
DESCRIPTION
  List app recovery actions for one version code

USAGE
  list

FLAGS
  -package-name string  Package name
  -version-code 0       Version code targeted by the recovery actions
```

## `gpc app-recoveries create --help`

```text
DESCRIPTION
  Create a draft app recovery action

USAGE
  create

FLAGS
  -input string         Path to app recovery JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc app-recoveries add-targeting --help`

```text
DESCRIPTION
  Add targeting to an app recovery action

USAGE
  add-targeting

FLAGS
  -app-recovery-id 0    App recovery action ID
  -input string         Path to targeting update JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc app-recoveries cancel --help`

```text
DESCRIPTION
  Cancel an app recovery action

USAGE
  cancel

FLAGS
  -app-recovery-id 0    App recovery action ID
  -confirm=false        Confirm canceling the app recovery action (required)
  -package-name string  Package name
```

## `gpc app-recoveries deploy --help`

```text
DESCRIPTION
  Deploy an app recovery action

USAGE
  deploy

FLAGS
  -app-recovery-id 0    App recovery action ID
  -confirm=false        Confirm deploying the app recovery action (required)
  -package-name string  Package name
```

## `gpc changelog --help`

```text
DESCRIPTION
  Release changelog workflows

USAGE
  changelog

SUBCOMMANDS
  sync  Sync track release notes from locale-named text files
```

## `gpc changelog sync --help`

```text
DESCRIPTION
  Sync track release notes from locale-named text files

USAGE
  sync

FLAGS
  -confirm=false           Confirm committing the edit (required unless --dry-run)
  -dir string              Changelog directory
  -dry-run=false           Create and validate the edit, then delete it instead of updating Play
  -fallback-locale string  Locale file to reuse when a locale-specific file is missing
  -package-name string     Package name
  -track string            Track name
```

## `gpc custom-apps --help`

```text
DESCRIPTION
  Create custom private apps for managed Play organizations

USAGE
  custom-apps

SUBCOMMANDS
  create  Create a custom app
```

## `gpc custom-apps create --help`

```text
DESCRIPTION
  Create a custom app

USAGE
  create

FLAGS
  -developer-id string  Developer account ID (numeric or developers/<id>)
  -input string         Path to custom app JSON payload (use - for stdin)
```

## `gpc edits --help`

```text
DESCRIPTION
  Manage Google Play edit transactions

USAGE
  edits

SUBCOMMANDS
  create                Create a new edit
  get                   Get edit details
  validate              Validate an edit
  commit                Commit an edit
  delete                Delete an edit
  details               Manage app details inside an edit
  testers               Manage testers for a track inside an edit
  country-availability  Inspect track country availability inside an edit
  listings              Manage listing changes inside an edit
  images                Manage store listing images inside an edit
  expansion-files       Manage APK expansion files inside an edit
```

## `gpc edits create --help`

```text
DESCRIPTION
  Create a new edit

USAGE
  create

FLAGS
  -package-name string  Package name
```

## `gpc edits get --help`

```text
DESCRIPTION
  Get edit details

USAGE
  get

FLAGS
  -edit-id string       Edit ID
  -package-name string  Package name
```

## `gpc edits validate --help`

```text
DESCRIPTION
  Validate an edit

USAGE
  validate

FLAGS
  -edit-id string       Edit ID
  -package-name string  Package name
```

## `gpc edits commit --help`

```text
DESCRIPTION
  Commit an edit

USAGE
  commit

FLAGS
  -changes-not-sent-for-review=false  Indicate that the changes in this edit will not be reviewed until they are explicitly sent for review from the Google Play Console UI
  -confirm=false                      Confirm committing the edit (required unless --dry-run)
  -dry-run=false                      Validate the edit without committing it
  -edit-id string                     Edit ID
  -package-name string                Package name
```

## `gpc edits delete --help`

```text
DESCRIPTION
  Delete an edit

USAGE
  delete

FLAGS
  -confirm=false        Confirm deleting the edit (required unless --dry-run)
  -dry-run=false        Verify the edit exists without deleting it
  -edit-id string       Edit ID
  -package-name string  Package name
```

## `gpc edits details --help`

```text
DESCRIPTION
  Manage app details inside an edit

USAGE
  details

SUBCOMMANDS
  get     Get app details in an edit
  update  Update app details in an edit
```

## `gpc edits details get --help`

```text
DESCRIPTION
  Get app details in an edit

USAGE
  get

FLAGS
  -edit-id string       Edit ID
  -package-name string  Package name
```

## `gpc edits details update --help`

```text
DESCRIPTION
  Update app details in an edit

USAGE
  update

FLAGS
  -contact-email string     Contact email address
  -contact-phone string     Contact phone number
  -contact-website string   Contact website URL
  -default-language string  Default listing language (BCP-47, e.g. en-US)
  -edit-id string           Edit ID
  -method patch             Update method: patch or update
  -package-name string      Package name
```

## `gpc edits testers --help`

```text
DESCRIPTION
  Manage testers for a track inside an edit

USAGE
  testers

SUBCOMMANDS
  get     Get tester Google Groups for a track in an edit
  update  Update tester Google Groups for a track in an edit
```

## `gpc edits testers get --help`

```text
DESCRIPTION
  Get tester Google Groups for a track in an edit

USAGE
  get

FLAGS
  -edit-id string       Edit ID
  -package-name string  Package name
  -track string         Track name (e.g. internal, closed)
```

## `gpc edits testers update --help`

```text
DESCRIPTION
  Update tester Google Groups for a track in an edit

USAGE
  update

FLAGS
  -edit-id string        Edit ID
  -google-groups string  Comma-separated Google Group email addresses
  -method patch          Update method: patch or update
  -package-name string   Package name
  -track string          Track name (e.g. internal, closed)
```

## `gpc edits country-availability --help`

```text
DESCRIPTION
  Inspect track country availability inside an edit

USAGE
  country-availability

SUBCOMMANDS
  get  Get country availability for a track in an edit
```

## `gpc edits country-availability get --help`

```text
DESCRIPTION
  Get country availability for a track in an edit

USAGE
  get

FLAGS
  -edit-id string       Edit ID
  -package-name string  Package name
  -track string         Track name (e.g. production)
```

## `gpc edits listings --help`

```text
DESCRIPTION
  Manage listing changes inside an edit

USAGE
  listings

SUBCOMMANDS
  list          List localized listings in an edit
  get           Get listing in an edit
  update        Update listing fields in an edit
  batch-update  Batch update listing fields from per-locale JSON files
  delete        Delete one localized listing in an edit
  delete-all    Delete all localized listings in an edit
```

## `gpc edits listings list --help`

```text
DESCRIPTION
  List localized listings in an edit

USAGE
  list

FLAGS
  -edit-id string       Edit ID
  -package-name string  Package name
```

## `gpc edits listings get --help`

```text
DESCRIPTION
  Get listing in an edit

USAGE
  get

FLAGS
  -edit-id string       Edit ID
  -locale string        Listing locale (BCP-47, e.g. en-US)
  -package-name string  Package name
```

## `gpc edits listings update --help`

```text
DESCRIPTION
  Update listing fields in an edit

USAGE
  update

FLAGS
  -edit-id string            Edit ID
  -full-description string   Localized full description
  -locale string             Listing locale (BCP-47, e.g. en-US)
  -method patch              Update method: patch or update
  -package-name string       Package name
  -short-description string  Localized short description
  -title string              Localized app title
```

## `gpc edits listings batch-update --help`

```text
DESCRIPTION
  Batch update listing fields from per-locale JSON files

USAGE
  batch-update

FLAGS
  -continue-on-error=true  Continue processing locales after errors
  -dry-run=false           Preview updates without calling the API
  -edit-id string          Edit ID
  -from-dir string         Directory containing per-locale JSON files (<locale>.json)
  -locales string          Optional comma-separated locale filter
  -package-name string     Package name
```

## `gpc edits listings delete --help`

```text
DESCRIPTION
  Delete one localized listing in an edit

USAGE
  delete

FLAGS
  -edit-id string       Edit ID
  -locale string        Listing locale (BCP-47, e.g. en-US)
  -package-name string  Package name
```

## `gpc edits listings delete-all --help`

```text
DESCRIPTION
  Delete all localized listings in an edit

USAGE
  delete-all

FLAGS
  -edit-id string       Edit ID
  -package-name string  Package name
```

## `gpc edits images --help`

```text
DESCRIPTION
  Manage store listing images inside an edit

USAGE
  images

SUBCOMMANDS
  list        List images for one locale/type inside an edit
  upload      Upload one image inside an edit
  upload-dir  Upload all image files from a directory inside an edit
  delete      Delete one image inside an edit
  delete-all  Delete all images for one locale/type inside an edit
```

## `gpc edits images list --help`

```text
DESCRIPTION
  List images for one locale/type inside an edit

USAGE
  list

FLAGS
  -edit-id string       Edit ID
  -image-type string    Image type (icon, featureGraphic, phoneScreenshots, ...)
  -locale string        Listing locale (BCP-47, e.g. en-US)
  -package-name string  Package name
```

## `gpc edits images upload --help`

```text
DESCRIPTION
  Upload one image inside an edit

USAGE
  upload

FLAGS
  -edit-id string       Edit ID
  -file string          Path to image file (PNG/JPEG)
  -image-type string    Image type (icon, featureGraphic, phoneScreenshots, ...)
  -locale string        Listing locale (BCP-47, e.g. en-US)
  -package-name string  Package name
```

## `gpc edits images upload-dir --help`

```text
DESCRIPTION
  Upload all image files from a directory inside an edit

USAGE
  upload-dir

FLAGS
  -dir string           Directory containing image files (PNG/JPEG)
  -edit-id string       Edit ID
  -image-type string    Image type (icon, featureGraphic, phoneScreenshots, ...)
  -locale string        Listing locale (BCP-47, e.g. en-US)
  -output string        Output format: json
  -package-name string  Package name
  -replace=false        Delete existing images for this locale/type before uploading
```

## `gpc edits images delete --help`

```text
DESCRIPTION
  Delete one image inside an edit

USAGE
  delete

FLAGS
  -edit-id string       Edit ID
  -image-id string      Image ID to delete
  -image-type string    Image type (icon, featureGraphic, phoneScreenshots, ...)
  -locale string        Listing locale (BCP-47, e.g. en-US)
  -package-name string  Package name
```

## `gpc edits images delete-all --help`

```text
DESCRIPTION
  Delete all images for one locale/type inside an edit

USAGE
  delete-all

FLAGS
  -edit-id string       Edit ID
  -image-type string    Image type (icon, featureGraphic, phoneScreenshots, ...)
  -locale string        Listing locale (BCP-47, e.g. en-US)
  -package-name string  Package name
```

## `gpc edits expansion-files --help`

```text
DESCRIPTION
  Manage APK expansion files inside an edit

USAGE
  expansion-files

SUBCOMMANDS
  get     Get expansion file configuration for one APK inside an edit
  patch   Patch expansion file reference for one APK inside an edit
  update  Update expansion file reference for one APK inside an edit
  upload  Upload an expansion file for one APK inside an edit
```

## `gpc edits expansion-files get --help`

```text
DESCRIPTION
  Get expansion file configuration for one APK inside an edit

USAGE
  get

FLAGS
  -apk-version-code 0          APK version code
  -edit-id string              Edit ID
  -expansion-file-type string  Expansion file type: main or patch
  -package-name string         Package name
```

## `gpc edits expansion-files patch --help`

```text
DESCRIPTION
  Patch expansion file reference for one APK inside an edit

USAGE
  patch

FLAGS
  -apk-version-code 0          APK version code
  -edit-id string              Edit ID
  -expansion-file-type string  Expansion file type: main or patch
  -package-name string         Package name
  -references-version 0        APK version code whose expansion file should be referenced
```

## `gpc edits expansion-files update --help`

```text
DESCRIPTION
  Update expansion file reference for one APK inside an edit

USAGE
  update

FLAGS
  -apk-version-code 0          APK version code
  -edit-id string              Edit ID
  -expansion-file-type string  Expansion file type: main or patch
  -package-name string         Package name
  -references-version 0        APK version code whose expansion file should be referenced
```

## `gpc edits expansion-files upload --help`

```text
DESCRIPTION
  Upload an expansion file for one APK inside an edit

USAGE
  upload

FLAGS
  -apk-version-code 0          APK version code
  -edit-id string              Edit ID
  -expansion-file-type string  Expansion file type: main or patch
  -file string                 Path to expansion file to upload
  -package-name string         Package name
```

## `gpc tracks --help`

```text
DESCRIPTION
  Manage release tracks inside an edit

USAGE
  tracks

SUBCOMMANDS
  list     List tracks in an edit
  get      Get a single track in an edit
  create   Create a new track in an edit
  patch    Patch a track release in an edit
  update   Update a track release in an edit
  promote  Promote a release from one track to another in an edit
```

## `gpc tracks list --help`

```text
DESCRIPTION
  List tracks in an edit

USAGE
  list

FLAGS
  -edit-id string       Edit ID
  -output string        Output format: json, minimal
  -package-name string  Package name
```

## `gpc tracks get --help`

```text
DESCRIPTION
  Get a single track in an edit

USAGE
  get

FLAGS
  -edit-id string       Edit ID
  -package-name string  Package name
  -track string         Track name (e.g. production, internal)
```

## `gpc tracks create --help`

```text
DESCRIPTION
  Create a new track in an edit

USAGE
  create

FLAGS
  -edit-id string       Edit ID
  -form-factor default  Track form factor (default, wear, automotive)
  -package-name string  Package name
  -track string         Track name (e.g. production, internal)
  -type closed-testing  Track type (currently only closed-testing)
```

## `gpc tracks patch --help`

```text
DESCRIPTION
  Patch a track release in an edit

USAGE
  patch

FLAGS
  -edit-id string              Edit ID
  -package-name string         Package name
  -release-name string         Release name
  -release-notes-file string   Path to release notes file (JSON object/array, tagged blocks, or plain text)
  -release-notes-locale en-US  Release notes locale (BCP-47)
  -release-notes-text string   Release notes text
  -status string               Release status (draft, inProgress, halted, completed)
  -track string                Track name (e.g. production, internal)
  -update-priority 0           In-app update priority (0-5)
  -user-fraction -1            Rollout user fraction (0-1)
  -version-codes string        Comma-separated version codes
```

## `gpc tracks update --help`

```text
DESCRIPTION
  Update a track release in an edit

USAGE
  update

FLAGS
  -edit-id string              Edit ID
  -package-name string         Package name
  -release-name string         Release name
  -release-notes-file string   Path to release notes file (JSON object/array, tagged blocks, or plain text)
  -release-notes-locale en-US  Release notes locale (BCP-47)
  -release-notes-text string   Release notes text
  -status string               Release status (draft, inProgress, halted, completed)
  -track string                Track name (e.g. production, internal)
  -update-priority 0           In-app update priority (0-5)
  -user-fraction -1            Rollout user fraction (0-1)
  -version-codes string        Comma-separated version codes
```

## `gpc tracks promote --help`

```text
DESCRIPTION
  Promote a release from one track to another in an edit

USAGE
  promote

FLAGS
  -edit-id string       Edit ID
  -from-track string    Source track name
  -package-name string  Package name
  -release-name string  Override release name
  -status string        Override release status
  -to-track string      Target track name
```

## `gpc apks --help`

```text
DESCRIPTION
  Manage APK uploads in an edit

USAGE
  apks

SUBCOMMANDS
  list                   List APKs in an edit
  upload                 Upload an APK to an edit
  add-externally-hosted  Register an externally hosted APK in an edit
```

## `gpc apks list --help`

```text
DESCRIPTION
  List APKs in an edit

USAGE
  list

FLAGS
  -edit-id string       Edit ID
  -package-name string  Package name
```

## `gpc apks upload --help`

```text
DESCRIPTION
  Upload an APK to an edit

USAGE
  upload

FLAGS
  -edit-id string       Edit ID
  -file string          Path to .apk file
  -package-name string  Package name
```

## `gpc apks add-externally-hosted --help`

```text
DESCRIPTION
  Register an externally hosted APK in an edit

USAGE
  add-externally-hosted

FLAGS
  -edit-id string       Edit ID
  -input string         Path to externally hosted APK JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc bundles --help`

```text
DESCRIPTION
  Manage Android App Bundles in an edit

USAGE
  bundles

SUBCOMMANDS
  list    List bundles in an edit
  upload  Upload an Android App Bundle to an edit
  wait    Wait until a bundle version code finishes processing
```

## `gpc bundles list --help`

```text
DESCRIPTION
  List bundles in an edit

USAGE
  list

FLAGS
  -edit-id string       Edit ID
  -output string        Output format: json, minimal
  -package-name string  Package name
```

## `gpc bundles upload --help`

```text
DESCRIPTION
  Upload an Android App Bundle to an edit

USAGE
  upload

FLAGS
  -edit-id string       Edit ID
  -file string          Path to .aab file
  -package-name string  Package name
```

## `gpc bundles wait --help`

```text
DESCRIPTION
  Wait until a bundle version code finishes processing

USAGE
  wait

FLAGS
  -interval 5s          Polling interval between generated APK checks
  -package-name string  Package name
  -timeout 1m30s        Maximum time to wait for generated APK availability
  -version-code 0       Version code returned from bundle upload
```

## `gpc deobfuscation --help`

```text
DESCRIPTION
  Manage deobfuscation files in an edit

USAGE
  deobfuscation

SUBCOMMANDS
  upload  Upload a deobfuscation file to an edit
```

## `gpc deobfuscation upload --help`

```text
DESCRIPTION
  Upload a deobfuscation file to an edit

USAGE
  upload

FLAGS
  -edit-id string       Edit ID
  -file string          Path to deobfuscation file
  -package-name string  Package name
  -type string          Deobfuscation file type: proguard or nativeCode
  -version-code 0       Version code associated with the mapping file
```

## `gpc deploy --help`

```text
DESCRIPTION
  Upload artifact and publish to a track in one flow

USAGE
  deploy

FLAGS
  -aab string                  Path to .aab file
  -allow-production=false      Allow deploys to production track
  -apk string                  Path to .apk file
  -cleanup-on-failure=true     Delete edit if deploy fails
  -confirm=false               Confirm committing the edit (required unless --dry-run)
  -dry-run=false               Run deploy steps, then delete edit instead of committing
  -mapping-file string         Path to deobfuscation mapping file
  -mapping-type string         Mapping type: proguard or nativeCode (defaults to proguard)
  -package-name string         Package name
  -release-name string         Release name
  -release-notes-file string   Path to release notes file (JSON object/array, tagged blocks, or plain text)
  -release-notes-locale en-US  Release notes locale (BCP-47)
  -release-notes-text string   Release notes text
  -status string               Release status (draft, inProgress, halted, completed)
  -track string                Track name (e.g. internal, production)
  -update-priority 0           In-app update priority (0-5)
  -user-fraction -1            Rollout user fraction (0-1)
```

## `gpc diff --help`

```text
DESCRIPTION
  Compare live Play state against local listing or track drafts

USAGE
  diff

SUBCOMMANDS
  listing  Compare live listings against a local listing directory
  track    Compare a draft track release against the live track
```

## `gpc diff listing --help`

```text
DESCRIPTION
  Compare live listings against a local listing directory

USAGE
  listing

FLAGS
  -delete-missing=false  Mark remote-only locales as deletions
  -dir string            Listings directory root
  -output string         Output format: json, table, markdown, yaml
  -package-name string   Package name
```

## `gpc diff track --help`

```text
DESCRIPTION
  Compare a draft track release against the live track

USAGE
  track

FLAGS
  -output string               Output format: json, table, markdown, yaml
  -package-name string         Package name
  -release-name string         Release name
  -release-notes-file string   Path to release notes file (JSON object/array, tagged blocks, or plain text)
  -release-notes-locale en-US  Release notes locale (BCP-47)
  -release-notes-text string   Release notes text
  -status string               Release status (draft, inProgress, halted, completed)
  -track string                Track name (e.g. production, internal)
  -update-priority 0           In-app update priority (0-5)
  -user-fraction -1            Rollout user fraction (0-1)
  -version-codes string        Comma-separated version codes
```

## `gpc doctor --help`

```text
DESCRIPTION
  Run read-only diagnostics for auth, package access, and e2e fixtures

USAGE
  doctor

FLAGS
  -fixtures-file string  Path to JSON fixture file
  -package-name string   Package name
  -strict=false          Fail on warnings
  -version-code 0        Version code for delivery diagnostics
```

## `gpc e2e --help`

```text
DESCRIPTION
  E2E fixture and smoke-testing helpers

USAGE
  e2e

SUBCOMMANDS
  fixtures  Inspect live e2e fixture readiness
```

## `gpc e2e fixtures --help`

```text
DESCRIPTION
  Inspect live e2e fixture readiness

USAGE
  fixtures

SUBCOMMANDS
  status  Check which live e2e fixtures are valid, missing, or stale
```

## `gpc e2e fixtures status --help`

```text
DESCRIPTION
  Check which live e2e fixtures are valid, missing, or stale

USAGE
  status

FLAGS
  -fixtures-file string  Path to JSON fixture file
  -package-name string   Package name
  -strict=false          Fail on warnings
  -version-code 0        Version code for delivery fixtures
```

## `gpc release --help`

```text
DESCRIPTION
  Release workflows for staged Google Play deploys

USAGE
  release

SUBCOMMANDS
  verify   Run non-mutating release readiness checks
  alpha    Build staging AAB and deploy to alpha track in one flow
  full     Deploy a release from a YAML or JSON manifest
  promote  Promote the latest releasable release from one track to another
```

## `gpc release verify --help`

```text
DESCRIPTION
  Run non-mutating release readiness checks

USAGE
  verify

FLAGS
  -aab string                            Path to prebuilt .aab for artifact validation
  -build-task :app:bundleStagingRelease  Gradle build task for release bundle
  -notes-file string                     Release notes file path when notes-mode=file
  -notes-locale en-US                    Release notes locale
  -notes-mode git                        Release notes mode: git, file, none
  -notes-text string                     Inline release notes text override
  -package-name com.example.app.staging  Target package name
  -probe-track=false                     Create temporary edit and probe target track
  -project-dir .                         Android project directory
  -track alpha                           Target track name
```

## `gpc release alpha --help`

```text
DESCRIPTION
  Build staging AAB and deploy to alpha track in one flow

USAGE
  alpha

FLAGS
  -aab string                            Path to prebuilt .aab (optional)
  -allow-production=false                Allow track=production
  -build-task :app:bundleStagingRelease  Gradle build task
  -cleanup-on-failure=true               Delete edit when deployment fails
  -confirm=false                         Confirm committing release (required unless --dry-run)
  -dry-run=false                         Run all steps but delete edit instead of committing
  -notes-file string                     Release notes file path when notes-mode=file
  -notes-locale en-US                    Release notes locale
  -notes-mode git                        Release notes mode: git, file, none
  -notes-text string                     Inline release notes text override
  -package-name com.example.app.staging  Target package name
  -probe-track=false                     Probe track existence during preflight verify
  -project-dir .                         Android project directory
  -release-name string                   Optional release name
  -skip-build=false                      Skip Gradle build and use prebuilt artifact
  -status completed                      Release status (draft, inProgress, halted, completed)
  -track alpha                           Target track name
  -update-priority 0                     In-app update priority (0-5)
  -user-fraction -1                      Rollout user fraction (0-1)
  -version-code 0                        Release versionCode override (default computed if empty)
  -version-name string                   Release versionName override
```

## `gpc release full --help`

```text
DESCRIPTION
  Deploy a release from a YAML or JSON manifest

USAGE
  full

FLAGS
  -allow-production=false         Allow track=production
  -auto-halt-on-regression=false  Halt an in-progress rollout if monitored vitals cross the configured thresholds
  -confirm=false                  Confirm committing release (required unless --dry-run)
  -dry-run=false                  Run all steps but delete edit instead of committing
  -manifest string                Path to release manifest (.json/.yaml/.yml)
  -package-name string            Target package name
  -vitals-gate string             Comma-separated vitals thresholds (for example: crashRate<2.0,anrRate<0.5)
  -vitals-wait 0s                 Monitor vitals after commit for the given duration
```

## `gpc release promote --help`

```text
DESCRIPTION
  Promote the latest releasable release from one track to another

USAGE
  promote

FLAGS
  -confirm=false        Confirm committing promotion (required unless --dry-run)
  -dry-run=false        Create and validate the promotion but delete the edit instead of committing
  -from-track string    Source track name
  -package-name string  Target package name
  -release-name string  Optional target release name override
  -status string        Optional target release status override
  -to-track string      Target track name
```

## `gpc rollback --help`

```text
DESCRIPTION
  Halt the active staged rollout on a track

USAGE
  rollback

FLAGS
  -confirm=false        Confirm halting the active rollout (required unless --dry-run)
  -dry-run=false        Create and validate the edit, then delete it instead of updating the track
  -package-name string  Package name
  -track string         Track name (e.g. production)
```

## `gpc screenshots --help`

```text
DESCRIPTION
  Manage screenshot-only sync workflows

USAGE
  screenshots

SUBCOMMANDS
  sync  Sync screenshots from a directory
```

## `gpc screenshots sync --help`

```text
DESCRIPTION
  Sync screenshots from a directory

USAGE
  sync

FLAGS
  -confirm=false        Confirm committing the edit (required unless --dry-run)
  -dir string           Screenshot directory root
  -dry-run=false        Create and validate the edit, then delete it instead of mutating Play
  -output string        Output format: json
  -package-name string  Package name
```

## `gpc setup --help`

```text
DESCRIPTION
  Provision auth and optional bootstrap workspace for gpc

USAGE
  setup

FLAGS
  -auto=false                           Run scripted setup without prompts
  -developer-id string                  Optional developer account ID
  -dir string                           Bootstrap directory (defaults to ./play)
  -package-name string                  Optional package name to verify and bootstrap
  -profile default                      Auth profile name
  -project-id string                    Google Cloud project ID
  -service-account-display-name string  Service account display name
  -service-account-key string           Path to write or reuse the service account key JSON
  -service-account-name string          Service account name (defaults to gpc-<profile>)
  -skip-bootstrap=false                 Skip local bootstrap export even when package access is available
  -write-project-config=true            Write .gpc.yaml when bootstrap runs
```

## `gpc status --help`

```text
DESCRIPTION
  Summarize tracks and recent review health for an app

USAGE
  status

FLAGS
  -output string        Output format: json, table, markdown, yaml, minimal
  -package-name string  Package name
```

## `gpc reviews --help`

```text
DESCRIPTION
  Read and reply to Play Store reviews

USAGE
  reviews

SUBCOMMANDS
  list    List app reviews
  get     Get one review by ID
  triage  Group reviews into pending-reply and replied buckets
  reply   Reply to a review
```

## `gpc reviews list --help`

```text
DESCRIPTION
  List app reviews

USAGE
  list

FLAGS
  -max-results 0                Maximum number of reviews per page
  -output string                Output format: json, minimal
  -package-name string          Package name
  -start-index 0                Index of first review to return (non-token pagination)
  -token string                 Pagination token
  -translation-language string  Language localization code for translated responses
```

## `gpc reviews get --help`

```text
DESCRIPTION
  Get one review by ID

USAGE
  get

FLAGS
  -package-name string  Package name
  -review-id string     Review ID
```

## `gpc reviews triage --help`

```text
DESCRIPTION
  Group reviews into pending-reply and replied buckets

USAGE
  triage

FLAGS
  -max-results 100              Maximum number of reviews per page
  -package-name string          Package name
  -start-index 0                Index of first review to return (non-token pagination)
  -token string                 Pagination token
  -translation-language string  Language localization code for translated responses
```

## `gpc reviews reply --help`

```text
DESCRIPTION
  Reply to a review

USAGE
  reply

FLAGS
  -package-name string  Package name
  -reply-text string    Reply text
  -review-id string     Review ID
```

## `gpc reports --help`

```text
DESCRIPTION
  Google Play Developer Reporting commands

USAGE
  reports

SUBCOMMANDS
  apps       Reporting app discovery commands
  anomalies  Reporting anomaly commands
  errors     Reporting error issue and report commands
  financial  Financial report commands backed by Cloud Storage
  summary    Summarize reporting visibility, anomalies, and vitals freshness for an app
  vitals     Vitals metric set reporting commands
```

## `gpc reports apps --help`

```text
DESCRIPTION
  Reporting app discovery commands

USAGE
  apps

SUBCOMMANDS
  list  List reporting-accessible apps
```

## `gpc reports apps list --help`

```text
DESCRIPTION
  List reporting-accessible apps

USAGE
  list

FLAGS
  -page-size 0        Maximum apps per page
  -page-token string  Page token for the next page
```

## `gpc reports anomalies --help`

```text
DESCRIPTION
  Reporting anomaly commands

USAGE
  anomalies

SUBCOMMANDS
  list  List reporting anomalies for an app
```

## `gpc reports anomalies list --help`

```text
DESCRIPTION
  List reporting anomalies for an app

USAGE
  list

FLAGS
  -filter string        Anomaly filter expression
  -package-name string  Package name
  -page-size 0          Maximum anomalies per page
  -page-token string    Page token for the next page
```

## `gpc reports errors --help`

```text
DESCRIPTION
  Reporting error issue and report commands

USAGE
  errors

SUBCOMMANDS
  counts   Error count metric set commands
  issues   Grouped error issue commands
  reports  Raw error report commands
```

## `gpc reports errors counts --help`

```text
DESCRIPTION
  Error count metric set commands

USAGE
  counts

SUBCOMMANDS
  get    Get error count metric set freshness metadata
  query  Query error count metric set rows
```

## `gpc reports errors counts get --help`

```text
DESCRIPTION
  Get error count metric set freshness metadata

USAGE
  get

FLAGS
  -package-name string  Package name
```

## `gpc reports errors counts query --help`

```text
DESCRIPTION
  Query error count metric set rows

USAGE
  query

FLAGS
  -input string         Path to error count query JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc reports errors issues --help`

```text
DESCRIPTION
  Grouped error issue commands

USAGE
  issues

SUBCOMMANDS
  list  Search grouped error issues for an app
```

## `gpc reports errors issues list --help`

```text
DESCRIPTION
  Search grouped error issues for an app

USAGE
  list

FLAGS
  -end-time string              RFC3339 interval end time
  -filter string                Error issue filter expression
  -order-by string              Sort order (for example: errorReportCount desc)
  -package-name string          Package name
  -page-size 0                  Maximum error issues per page
  -page-token string            Page token for the next page
  -sample-error-report-limit 0  Sample error reports per issue
  -start-time string            RFC3339 interval start time
```

## `gpc reports errors reports --help`

```text
DESCRIPTION
  Raw error report commands

USAGE
  reports

SUBCOMMANDS
  list  Search raw error reports for an app
```

## `gpc reports errors reports list --help`

```text
DESCRIPTION
  Search raw error reports for an app

USAGE
  list

FLAGS
  -end-time string      RFC3339 interval end time
  -filter string        Error report filter expression
  -package-name string  Package name
  -page-size 0          Maximum error reports per page
  -page-token string    Page token for the next page
  -start-time string    RFC3339 interval start time
```

## `gpc reports financial --help`

```text
DESCRIPTION
  Financial report commands backed by Cloud Storage

USAGE
  financial

SUBCOMMANDS
  list  List Cloud Storage financial report objects
  get   Download and normalize a financial report CSV from Cloud Storage
```

## `gpc reports financial list --help`

```text
DESCRIPTION
  List Cloud Storage financial report objects

USAGE
  list

FLAGS
  -bucket string      Cloud Storage bucket containing financial reports
  -output string      Output format: json, table, markdown, yaml, csv, tsv
  -page-size 0        Maximum objects per page
  -page-token string  Page token for the next page
  -prefix string      Optional object prefix filter
```

## `gpc reports financial get --help`

```text
DESCRIPTION
  Download and normalize a financial report CSV from Cloud Storage

USAGE
  get

FLAGS
  -bucket string   Cloud Storage bucket containing the report
  -gcs-uri string  Cloud Storage object URI in the form gs://bucket/object.csv
  -object string   Cloud Storage object name
  -output string   Output format: json, table, markdown, yaml, csv, tsv
```

## `gpc reports summary --help`

```text
DESCRIPTION
  Summarize reporting visibility, anomalies, and vitals freshness for an app

USAGE
  summary

FLAGS
  -input string           Path to vitals query JSON payload (defaults to a built-in last-7-days window)
  -metric-set crash-rate  Vitals metric set
  -package-name string    Package name
```

## `gpc reports vitals --help`

```text
DESCRIPTION
  Vitals metric set reporting commands

USAGE
  vitals

SUBCOMMANDS
  get    Get vitals metric set freshness metadata
  query  Query vitals metric set rows
```

## `gpc reports vitals get --help`

```text
DESCRIPTION
  Get vitals metric set freshness metadata

USAGE
  get

FLAGS
  -metric-set string    Vitals metric set
  -package-name string  Package name
```

## `gpc reports vitals query --help`

```text
DESCRIPTION
  Query vitals metric set rows

USAGE
  query

FLAGS
  -input string         Path to vitals query JSON payload (use - for stdin)
  -metric-set string    Vitals metric set
  -package-name string  Package name
```

## `gpc orders --help`

```text
DESCRIPTION
  Inspect and refund Play orders

USAGE
  orders

SUBCOMMANDS
  get        Get one Play order by ID
  batch-get  Get multiple Play orders by ID
  refund     Refund a Play order
```

## `gpc orders get --help`

```text
DESCRIPTION
  Get one Play order by ID

USAGE
  get

FLAGS
  -order-id string      Order ID
  -package-name string  Package name
```

## `gpc orders batch-get --help`

```text
DESCRIPTION
  Get multiple Play orders by ID

USAGE
  batch-get

FLAGS
  -order-ids string     Comma-separated order IDs
  -package-name string  Package name
```

## `gpc orders refund --help`

```text
DESCRIPTION
  Refund a Play order

USAGE
  refund

FLAGS
  -confirm=false        Confirm refunding the order (required)
  -order-id string      Order ID
  -package-name string  Package name
  -revoke=false         Also revoke the purchase entitlement
```

## `gpc external-transactions --help`

```text
DESCRIPTION
  Report and refund external transactions

USAGE
  external-transactions

SUBCOMMANDS
  get     Get an external transaction by ID
  create  Create an external transaction report
  refund  Refund an external transaction
```

## `gpc external-transactions get --help`

```text
DESCRIPTION
  Get an external transaction by ID

USAGE
  get

FLAGS
  -external-transaction-id string  External transaction ID
  -package-name string             Package name
```

## `gpc external-transactions create --help`

```text
DESCRIPTION
  Create an external transaction report

USAGE
  create

FLAGS
  -external-transaction-id string  External transaction ID
  -input string                    Path to external transaction JSON payload (use - for stdin)
  -package-name string             Package name
```

## `gpc external-transactions refund --help`

```text
DESCRIPTION
  Refund an external transaction

USAGE
  refund

FLAGS
  -confirm=false                   Confirm refunding the external transaction (required)
  -external-transaction-id string  External transaction ID
  -input string                    Path to refund request JSON payload (use - for stdin)
  -package-name string             Package name
```

## `gpc device-tier-configs --help`

```text
DESCRIPTION
  Manage application device tier configs

USAGE
  device-tier-configs

SUBCOMMANDS
  list    List device tier configs
  get     Get a device tier config by ID
  create  Create a device tier config
```

## `gpc device-tier-configs list --help`

```text
DESCRIPTION
  List device tier configs

USAGE
  list

FLAGS
  -package-name string  Package name
  -page-size 0          Maximum device tier configs per page
  -page-token string    Page token for the next page
```

## `gpc device-tier-configs get --help`

```text
DESCRIPTION
  Get a device tier config by ID

USAGE
  get

FLAGS
  -device-tier-config-id 0  Device tier config ID
  -package-name string      Package name
```

## `gpc device-tier-configs create --help`

```text
DESCRIPTION
  Create a device tier config

USAGE
  create

FLAGS
  -allow-unknown-devices=false  Allow device IDs unknown to Play's device catalog
  -input string                 Path to device tier config JSON payload (use - for stdin)
  -package-name string          Package name
```

## `gpc system-apks --help`

```text
DESCRIPTION
  Manage generated system APK variants

USAGE
  system-apks

SUBCOMMANDS
  list      List generated system APK variants
  get       Get a generated system APK variant
  create    Create a generated system APK variant
  download  Download a generated system APK variant
```

## `gpc system-apks list --help`

```text
DESCRIPTION
  List generated system APK variants

USAGE
  list

FLAGS
  -package-name string  Package name
  -version-code 0       Version code of the App Bundle
```

## `gpc system-apks get --help`

```text
DESCRIPTION
  Get a generated system APK variant

USAGE
  get

FLAGS
  -package-name string  Package name
  -variant-id 0         System APK variant ID
  -version-code 0       Version code of the App Bundle
```

## `gpc system-apks create --help`

```text
DESCRIPTION
  Create a generated system APK variant

USAGE
  create

FLAGS
  -input string         Path to system APK variant JSON payload (use - for stdin)
  -package-name string  Package name
  -version-code 0       Version code of the App Bundle
```

## `gpc system-apks download --help`

```text
DESCRIPTION
  Download a generated system APK variant

USAGE
  download

FLAGS
  -output string        Path to write the downloaded APK
  -package-name string  Package name
  -variant-id 0         System APK variant ID
  -version-code 0       Version code of the App Bundle
```

## `gpc generated-apks --help`

```text
DESCRIPTION
  List and download APKs generated from bundles

USAGE
  generated-apks

SUBCOMMANDS
  list      List generated APK download metadata
  download  Download one generated APK by download ID
```

## `gpc generated-apks list --help`

```text
DESCRIPTION
  List generated APK download metadata

USAGE
  list

FLAGS
  -package-name string  Package name
  -version-code 0       Version code of the App Bundle
```

## `gpc generated-apks download --help`

```text
DESCRIPTION
  Download one generated APK by download ID

USAGE
  download

FLAGS
  -download-id string   Generated APK download ID
  -output string        Path to write the downloaded APK
  -package-name string  Package name
  -version-code 0       Version code of the App Bundle
```

## `gpc games --help`

```text
DESCRIPTION
  Inspect Play Games Services achievements, events, and leaderboards

USAGE
  games

SUBCOMMANDS
  achievements  List Play Games achievement definitions
  events        List Play Games event definitions
  leaderboards  List and inspect Play Games leaderboards
```

## `gpc games achievements --help`

```text
DESCRIPTION
  List Play Games achievement definitions

USAGE
  achievements

SUBCOMMANDS
  list  List Play Games achievement definitions
```

## `gpc games achievements list --help`

```text
DESCRIPTION
  List Play Games achievement definitions

USAGE
  list

FLAGS
  -language string    Preferred language for localized strings
  -output string      Output format: json, table, markdown
  -page-size 0        Maximum items per page
  -page-token string  Page token for the next page
```

## `gpc games events --help`

```text
DESCRIPTION
  List Play Games event definitions

USAGE
  events

SUBCOMMANDS
  list  List Play Games event definitions
```

## `gpc games events list --help`

```text
DESCRIPTION
  List Play Games event definitions

USAGE
  list

FLAGS
  -language string    Preferred language for localized strings
  -output string      Output format: json, table, markdown
  -page-size 0        Maximum items per page
  -page-token string  Page token for the next page
```

## `gpc games leaderboards --help`

```text
DESCRIPTION
  List and inspect Play Games leaderboards

USAGE
  leaderboards

SUBCOMMANDS
  list  List Play Games leaderboards
  get   Get Play Games leaderboard metadata
```

## `gpc games leaderboards list --help`

```text
DESCRIPTION
  List Play Games leaderboards

USAGE
  list

FLAGS
  -language string    Preferred language for localized strings
  -output string      Output format: json, table, markdown
  -page-size 0        Maximum items per page
  -page-token string  Page token for the next page
```

## `gpc games leaderboards get --help`

```text
DESCRIPTION
  Get Play Games leaderboard metadata

USAGE
  get

FLAGS
  -language string        Preferred language for localized strings
  -leaderboard-id string  Leaderboard ID
  -output string          Output format: json, table, markdown
```

## `gpc subscriptions --help`

```text
DESCRIPTION
  Manage monetization subscriptions

USAGE
  subscriptions

SUBCOMMANDS
  list          List subscriptions
  get           Get a subscription by product ID
  sync          Sync subscriptions from exported JSON files
  batch-get     Batch-get subscriptions by product IDs
  create        Create a subscription
  batch-update  Batch create or update subscriptions
  update        Update a subscription
  delete        Delete a subscription
  archive       Archive a subscription
  base-plans    Manage subscription base plans
  offers        Manage subscription offers under base plans
```

## `gpc subscriptions list --help`

```text
DESCRIPTION
  List subscriptions

USAGE
  list

FLAGS
  -package-name string  Package name
  -page-size 0          Maximum subscriptions per page
  -page-token string    Page token for the next page
```

## `gpc subscriptions get --help`

```text
DESCRIPTION
  Get a subscription by product ID

USAGE
  get

FLAGS
  -full=false           Return the full raw subscription resource
  -package-name string  Package name
  -product-id string    Subscription product ID
  -verbose=false        Include base plan and region diagnostics
```

## `gpc subscriptions sync --help`

```text
DESCRIPTION
  Sync subscriptions from exported JSON files

USAGE
  sync

FLAGS
  -confirm=false         Confirm applying subscription changes (required unless --dry-run)
  -delete-missing=false  Delete remote subscriptions not present in the local directory
  -dir string            Directory containing exported subscription JSON files
  -dry-run=false         Plan subscription changes without mutating Play
  -package-name string   Package name
```

## `gpc subscriptions batch-get --help`

```text
DESCRIPTION
  Batch-get subscriptions by product IDs

USAGE
  batch-get

FLAGS
  -package-name string  Package name
  -product-ids string   Comma-separated subscription product IDs
```

## `gpc subscriptions create --help`

```text
DESCRIPTION
  Create a subscription

USAGE
  create

FLAGS
  -input string         Path to subscription JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc subscriptions batch-update --help`

```text
DESCRIPTION
  Batch create or update subscriptions

USAGE
  batch-update

FLAGS
  -input string         Path to subscriptions batch update JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc subscriptions update --help`

```text
DESCRIPTION
  Update a subscription

USAGE
  update

FLAGS
  -input string         Path to subscription JSON payload (use - for stdin)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions delete --help`

```text
DESCRIPTION
  Delete a subscription

USAGE
  delete

FLAGS
  -confirm=false        Confirm deleting the subscription (required)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions archive --help`

```text
DESCRIPTION
  Archive a subscription

USAGE
  archive

FLAGS
  -confirm=false        Confirm archiving the subscription (required)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions base-plans --help`

```text
DESCRIPTION
  Manage subscription base plans

USAGE
  base-plans

SUBCOMMANDS
  activate              Activate a subscription base plan
  deactivate            Deactivate a subscription base plan
  batch-update-states   Batch update subscription base plan states
  delete                Delete a subscription base plan
  migrate-prices        Migrate legacy cohorts to current prices for one base plan
  batch-migrate-prices  Batch migrate legacy cohorts to current prices across base plans
```

## `gpc subscriptions base-plans activate --help`

```text
DESCRIPTION
  Activate a subscription base plan

USAGE
  activate

FLAGS
  -base-plan-id string  Base plan ID
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions base-plans deactivate --help`

```text
DESCRIPTION
  Deactivate a subscription base plan

USAGE
  deactivate

FLAGS
  -base-plan-id string  Base plan ID
  -confirm=false        Confirm deactivating the base plan (required)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions base-plans batch-update-states --help`

```text
DESCRIPTION
  Batch update subscription base plan states

USAGE
  batch-update-states

FLAGS
  -confirm=false        Confirm batch-updating base plan states (required)
  -input string         Path to base plan state update JSON payload (use - for stdin)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions base-plans delete --help`

```text
DESCRIPTION
  Delete a subscription base plan

USAGE
  delete

FLAGS
  -base-plan-id string  Base plan ID
  -confirm=false        Confirm deleting the base plan (required)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions base-plans migrate-prices --help`

```text
DESCRIPTION
  Migrate legacy cohorts to current prices for one base plan

USAGE
  migrate-prices

FLAGS
  -base-plan-id string  Base plan ID
  -confirm=false        Confirm migrating base plan prices (required)
  -input string         Path to base plan migrate-prices JSON payload (use - for stdin)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions base-plans batch-migrate-prices --help`

```text
DESCRIPTION
  Batch migrate legacy cohorts to current prices across base plans

USAGE
  batch-migrate-prices

FLAGS
  -confirm=false        Confirm migrating base plan prices in batch (required)
  -input string         Path to batch migrate-prices JSON payload (use - for stdin)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions offers --help`

```text
DESCRIPTION
  Manage subscription offers under base plans

USAGE
  offers

SUBCOMMANDS
  list                 List offers under a subscription base plan
  get                  Get one subscription offer
  batch-get            Batch-get subscription offers
  batch-update         Batch create or update subscription offers
  batch-update-states  Batch update subscription offer states
  activate             Activate a subscription offer
  deactivate           Deactivate a subscription offer
  create               Create a subscription offer
  update               Update a subscription offer
  delete               Delete a subscription offer
```

## `gpc subscriptions offers list --help`

```text
DESCRIPTION
  List offers under a subscription base plan

USAGE
  list

FLAGS
  -base-plan-id string  Base plan ID
  -package-name string  Package name
  -page-size 0          Maximum offers per page
  -page-token string    Page token for the next page
  -product-id string    Subscription product ID
```

## `gpc subscriptions offers get --help`

```text
DESCRIPTION
  Get one subscription offer

USAGE
  get

FLAGS
  -base-plan-id string  Base plan ID
  -offer-id string      Offer ID
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions offers batch-get --help`

```text
DESCRIPTION
  Batch-get subscription offers

USAGE
  batch-get

FLAGS
  -base-plan-id string  Base plan ID
  -offer-ids string     Comma-separated offer IDs
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions offers batch-update --help`

```text
DESCRIPTION
  Batch create or update subscription offers

USAGE
  batch-update

FLAGS
  -base-plan-id string  Base plan ID
  -input string         Path to subscription offers batch update JSON payload (use - for stdin)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions offers batch-update-states --help`

```text
DESCRIPTION
  Batch update subscription offer states

USAGE
  batch-update-states

FLAGS
  -base-plan-id string  Base plan ID
  -confirm=false        Confirm batch-updating offer states (required)
  -input string         Path to subscription offer state updates JSON payload (use - for stdin)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions offers activate --help`

```text
DESCRIPTION
  Activate a subscription offer

USAGE
  activate

FLAGS
  -base-plan-id string  Base plan ID
  -offer-id string      Offer ID
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions offers deactivate --help`

```text
DESCRIPTION
  Deactivate a subscription offer

USAGE
  deactivate

FLAGS
  -base-plan-id string  Base plan ID
  -confirm=false        Confirm deactivating the offer (required)
  -offer-id string      Offer ID
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions offers create --help`

```text
DESCRIPTION
  Create a subscription offer

USAGE
  create

FLAGS
  -activate=false       Activate the offer after creating it when needed
  -base-plan-id string  Base plan ID
  -input string         Path to subscription offer JSON payload (use - for stdin)
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc subscriptions offers update --help`

```text
DESCRIPTION
  Update a subscription offer

USAGE
  update

FLAGS
  -activate=false       Activate the offer after updating it when needed
  -base-plan-id string  Base plan ID
  -input string         Path to subscription offer JSON payload (use - for stdin)
  -offer-id string      Offer ID
  -package-name string  Package name
  -product-id string    Subscription product ID
  -update-mask string   Comma-separated list of fields to update
```

## `gpc subscriptions offers delete --help`

```text
DESCRIPTION
  Delete a subscription offer

USAGE
  delete

FLAGS
  -base-plan-id string  Base plan ID
  -confirm=false        Confirm deleting the offer (required)
  -offer-id string      Offer ID
  -package-name string  Package name
  -product-id string    Subscription product ID
```

## `gpc monetization --help`

```text
DESCRIPTION
  Monetization workflows

USAGE
  monetization

SUBCOMMANDS
  regions  List billable monetization regions
  setup    Create a subscription product from a YAML or JSON manifest
  sync     Create or update a subscription product from a YAML or JSON manifest
```

## `gpc monetization regions --help`

```text
DESCRIPTION
  List billable monetization regions

USAGE
  regions

FLAGS
  -package-name string  Package name
```

## `gpc monetization setup --help`

```text
DESCRIPTION
  Create a subscription product from a YAML or JSON manifest

USAGE
  setup

FLAGS
  -activate=false       Activate created base plans and offers after creation
  -confirm=false        Confirm creating monetization resources (required unless --dry-run)
  -dry-run=false        Validate manifest and check for conflicts without creating resources
  -manifest string      Path to monetization manifest (.json/.yaml/.yml)
  -package-name string  Package name
```

## `gpc monetization sync --help`

```text
DESCRIPTION
  Create or update a subscription product from a YAML or JSON manifest

USAGE
  sync

FLAGS
  -activate=false       Activate synced base plans and offers when needed
  -confirm=false        Confirm syncing monetization resources (required unless --dry-run)
  -dry-run=false        Validate manifest and show planned sync actions without applying changes
  -manifest string      Path to monetization manifest (.json/.yaml/.yml)
  -package-name string  Package name
```

## `gpc migrate --help`

```text
DESCRIPTION
  Import or transform metadata from other tool layouts

USAGE
  migrate

SUBCOMMANDS
  fastlane  Migrate Fastlane metadata into gpc workspace layout
```

## `gpc migrate fastlane --help`

```text
DESCRIPTION
  Migrate Fastlane metadata into gpc workspace layout

USAGE
  fastlane

SUBCOMMANDS
  import  Import Fastlane metadata into the local gpc listing/changelog layout
```

## `gpc migrate fastlane import --help`

```text
DESCRIPTION
  Import Fastlane metadata into the local gpc listing/changelog layout

USAGE
  import

FLAGS
  -dir string                  Target gpc workspace directory
  -from-dir fastlane           Fastlane metadata root (fastlane, `fastlane/metadata`, or `fastlane/metadata/android`)
  -package-name string         Package name to persist into .gpc.yaml
  -track production            Track name for imported changelogs
  -version-code 0              Preferred Fastlane changelog version code (falls back to default.txt)
  -write-project-config=false  Write .gpc.yaml with local listing/changelog defaults
```

## `gpc notify --help`

```text
DESCRIPTION
  Notification delivery helpers

USAGE
  notify

SUBCOMMANDS
  webhook  POST a JSON payload to a webhook endpoint
  slack    POST a native Slack webhook message
  discord  POST a native Discord webhook message
```

## `gpc notify webhook --help`

```text
DESCRIPTION
  POST a JSON payload to a webhook endpoint

USAGE
  webhook

FLAGS
  -event string      Event name metadata
  -input string      Path to JSON payload file (use - for stdin)
  -retry-attempts 0  Additional retry attempts for network, 429, or 5xx failures
  -retry-delay 1s    Delay between retry attempts
  -url string        Webhook URL
```

## `gpc notify slack --help`

```text
DESCRIPTION
  POST a native Slack webhook message

USAGE
  slack

FLAGS
  -event string      Event name metadata
  -input string      Optional path to JSON context payload (use - for stdin)
  -message string    Notification message text
  -retry-attempts 0  Additional retry attempts for network, 429, or 5xx failures
  -retry-delay 1s    Delay between retry attempts
  -title string      Optional notification title (defaults to --event)
  -url string        Slack webhook URL
```

## `gpc notify discord --help`

```text
DESCRIPTION
  POST a native Discord webhook message

USAGE
  discord

FLAGS
  -event string      Event name metadata
  -input string      Optional path to JSON context payload (use - for stdin)
  -message string    Notification message text
  -retry-attempts 0  Additional retry attempts for network, 429, or 5xx failures
  -retry-delay 1s    Delay between retry attempts
  -title string      Optional embed title (defaults to --event)
  -url string        Discord webhook URL
```

## `gpc products --help`

```text
DESCRIPTION
  Manage monetization one-time products

USAGE
  products

SUBCOMMANDS
  list              List one-time products
  get               Get a one-time product by product ID
  sync              Sync one-time products from exported JSON files
  batch-get         Batch-get one-time products by product IDs
  batch-update      Batch create or update one-time products
  batch-delete      Batch delete one-time products
  create            Create a one-time product (patch with allowMissing=true)
  update            Update a one-time product
  delete            Delete a one-time product
  offers            Manage one-time product offers under purchase options
  purchase-options  Manage one-time product purchase options
```

## `gpc products list --help`

```text
DESCRIPTION
  List one-time products

USAGE
  list

FLAGS
  -package-name string  Package name
  -page-size 0          Maximum one-time products per page
  -page-token string    Page token for the next page
```

## `gpc products get --help`

```text
DESCRIPTION
  Get a one-time product by product ID

USAGE
  get

FLAGS
  -package-name string  Package name
  -product-id string    One-time product ID
  -verbose=false        Include purchase option and region diagnostics
```

## `gpc products sync --help`

```text
DESCRIPTION
  Sync one-time products from exported JSON files

USAGE
  sync

FLAGS
  -confirm=false         Confirm applying product changes (required unless --dry-run)
  -delete-missing=false  Delete remote products not present in the local directory
  -dir string            Directory containing exported product JSON files
  -dry-run=false         Plan product changes without mutating Play
  -package-name string   Package name
```

## `gpc products batch-get --help`

```text
DESCRIPTION
  Batch-get one-time products by product IDs

USAGE
  batch-get

FLAGS
  -package-name string  Package name
  -product-ids string   Comma-separated one-time product IDs
```

## `gpc products batch-update --help`

```text
DESCRIPTION
  Batch create or update one-time products

USAGE
  batch-update

FLAGS
  -input string         Path to one-time products batch update JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc products batch-delete --help`

```text
DESCRIPTION
  Batch delete one-time products

USAGE
  batch-delete

FLAGS
  -confirm=false        Confirm deleting the one-time products (required)
  -input string         Path to one-time products batch delete JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc products create --help`

```text
DESCRIPTION
  Create a one-time product (patch with allowMissing=true)

USAGE
  create

FLAGS
  -input string         Path to one-time product JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc products update --help`

```text
DESCRIPTION
  Update a one-time product

USAGE
  update

FLAGS
  -input string         Path to one-time product JSON payload (use - for stdin)
  -package-name string  Package name
  -product-id string    One-time product ID
  -update-mask string   Comma-separated list of fields to update
```

## `gpc products delete --help`

```text
DESCRIPTION
  Delete a one-time product

USAGE
  delete

FLAGS
  -confirm=false        Confirm deleting the one-time product (required)
  -package-name string  Package name
  -product-id string    One-time product ID
```

## `gpc products offers --help`

```text
DESCRIPTION
  Manage one-time product offers under purchase options

USAGE
  offers

SUBCOMMANDS
  list                 List offers for a one-time product purchase option
  batch-get            Batch-get one-time product offers
  batch-update         Batch create or update one-time product offers
  batch-update-states  Batch update one-time product offer states
  batch-delete         Batch delete one-time product offers
  activate             Activate a one-time product offer
  deactivate           Deactivate a one-time product offer
  cancel               Cancel a one-time product pre-order offer
```

## `gpc products offers list --help`

```text
DESCRIPTION
  List offers for a one-time product purchase option

USAGE
  list

FLAGS
  -package-name string        Package name
  -page-size 0                Maximum offers per page
  -page-token string          Page token for the next page
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products offers batch-get --help`

```text
DESCRIPTION
  Batch-get one-time product offers

USAGE
  batch-get

FLAGS
  -offer-ids string           Comma-separated offer IDs
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products offers batch-update --help`

```text
DESCRIPTION
  Batch create or update one-time product offers

USAGE
  batch-update

FLAGS
  -input string               Path to one-time product offers batch update JSON payload (use - for stdin)
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products offers batch-update-states --help`

```text
DESCRIPTION
  Batch update one-time product offer states

USAGE
  batch-update-states

FLAGS
  -confirm=false              Confirm batch-updating offer states (required)
  -input string               Path to one-time product offer state updates JSON payload (use - for stdin)
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products offers batch-delete --help`

```text
DESCRIPTION
  Batch delete one-time product offers

USAGE
  batch-delete

FLAGS
  -confirm=false              Confirm deleting the offers (required)
  -input string               Path to one-time product offers batch delete JSON payload (use - for stdin)
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products offers activate --help`

```text
DESCRIPTION
  Activate a one-time product offer

USAGE
  activate

FLAGS
  -offer-id string            Offer ID
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products offers deactivate --help`

```text
DESCRIPTION
  Deactivate a one-time product offer

USAGE
  deactivate

FLAGS
  -confirm=false              Confirm deactivating the offer (required)
  -offer-id string            Offer ID
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products offers cancel --help`

```text
DESCRIPTION
  Cancel a one-time product pre-order offer

USAGE
  cancel

FLAGS
  -confirm=false              Confirm canceling the offer (required)
  -offer-id string            Offer ID
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products purchase-options --help`

```text
DESCRIPTION
  Manage one-time product purchase options

USAGE
  purchase-options

SUBCOMMANDS
  activate    Activate a one-time product purchase option
  deactivate  Deactivate a one-time product purchase option
  delete      Delete a one-time product purchase option
```

## `gpc products purchase-options activate --help`

```text
DESCRIPTION
  Activate a one-time product purchase option

USAGE
  activate

FLAGS
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products purchase-options deactivate --help`

```text
DESCRIPTION
  Deactivate a one-time product purchase option

USAGE
  deactivate

FLAGS
  -confirm=false              Confirm deactivating the purchase option (required)
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc products purchase-options delete --help`

```text
DESCRIPTION
  Delete a one-time product purchase option

USAGE
  delete

FLAGS
  -confirm=false              Confirm deleting the purchase option (required)
  -force=false                Delete even when managed externally
  -package-name string        Package name
  -product-id string          One-time product ID
  -purchase-option-id string  Purchase option ID
```

## `gpc publish --help`

```text
DESCRIPTION
  Common publish flows with track presets

USAGE
  publish

SUBCOMMANDS
  alpha       Upload and publish to the alpha track in one flow
  production  Upload and publish to the production track in one flow
```

## `gpc publish alpha --help`

```text
DESCRIPTION
  Upload and publish to the alpha track in one flow

USAGE
  alpha

FLAGS
  -aab string                  Path to .aab file
  -apk string                  Path to .apk file
  -cleanup-on-failure=true     Delete edit if publish fails before commit
  -confirm=false               Confirm committing the edit (required unless --dry-run)
  -dry-run=false               Run publish steps, then delete edit instead of committing
  -mapping-file string         Path to deobfuscation mapping file
  -mapping-type string         Mapping type: proguard or nativeCode (defaults to proguard)
  -package-name string         Package name
  -release-name string         Release name
  -release-notes-file string   Path to release notes file (JSON object/array, tagged blocks, or plain text)
  -release-notes-locale en-US  Release notes locale (BCP-47)
  -release-notes-text string   Release notes text
  -status completed            Release status (draft, inProgress, halted, completed)
  -update-priority 0           In-app update priority (0-5)
  -user-fraction -1            Rollout user fraction (0-1)
  -wait-interval 5s            Polling interval between generated APK checks
  -wait-timeout 1m30s          Maximum time to wait for generated APK availability after bundle upload
```

## `gpc publish production --help`

```text
DESCRIPTION
  Upload and publish to the production track in one flow

USAGE
  production

FLAGS
  -aab string                  Path to .aab file
  -apk string                  Path to .apk file
  -cleanup-on-failure=true     Delete edit if publish fails before commit
  -confirm=false               Confirm committing the edit (required unless --dry-run)
  -dry-run=false               Run publish steps, then delete edit instead of committing
  -mapping-file string         Path to deobfuscation mapping file
  -mapping-type string         Mapping type: proguard or nativeCode (defaults to proguard)
  -package-name string         Package name
  -release-name string         Release name
  -release-notes-file string   Path to release notes file (JSON object/array, tagged blocks, or plain text)
  -release-notes-locale en-US  Release notes locale (BCP-47)
  -release-notes-text string   Release notes text
  -status completed            Release status (draft, inProgress, halted, completed)
  -update-priority 0           In-app update priority (0-5)
  -user-fraction -1            Rollout user fraction (0-1)
  -wait-interval 5s            Polling interval between generated APK checks
  -wait-timeout 1m30s          Maximum time to wait for generated APK availability after bundle upload
```

## `gpc iap --help`

```text
DESCRIPTION
  Manage legacy in-app products

USAGE
  iap

SUBCOMMANDS
  list          List legacy in-app products
  get           Get a legacy in-app product by SKU
  batch-get     Get multiple legacy in-app products by SKU
  create        Create a legacy in-app product
  update        Update a legacy in-app product
  replace       Replace a legacy in-app product
  batch-update  Create or update multiple legacy in-app products
  batch-delete  Delete multiple legacy in-app products
  delete        Delete a legacy in-app product
```

## `gpc iap list --help`

```text
DESCRIPTION
  List legacy in-app products

USAGE
  list

FLAGS
  -max-results 0        Maximum in-app products per page
  -package-name string  Package name
  -page-token string    Page token for the next page
```

## `gpc iap get --help`

```text
DESCRIPTION
  Get a legacy in-app product by SKU

USAGE
  get

FLAGS
  -package-name string  Package name
  -sku string           In-app product SKU
```

## `gpc iap batch-get --help`

```text
DESCRIPTION
  Get multiple legacy in-app products by SKU

USAGE
  batch-get

FLAGS
  -package-name string  Package name
  -skus string          Comma-separated in-app product SKUs
```

## `gpc iap create --help`

```text
DESCRIPTION
  Create a legacy in-app product

USAGE
  create

FLAGS
  -input string         Path to in-app product JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc iap update --help`

```text
DESCRIPTION
  Update a legacy in-app product

USAGE
  update

FLAGS
  -input string         Path to in-app product JSON payload (use - for stdin)
  -package-name string  Package name
  -sku string           In-app product SKU
```

## `gpc iap replace --help`

```text
DESCRIPTION
  Replace a legacy in-app product

USAGE
  replace

FLAGS
  -input string         Path to in-app product JSON payload (use - for stdin)
  -package-name string  Package name
  -sku string           In-app product SKU
```

## `gpc iap batch-update --help`

```text
DESCRIPTION
  Create or update multiple legacy in-app products

USAGE
  batch-update

FLAGS
  -input string         Path to in-app products batch update JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc iap batch-delete --help`

```text
DESCRIPTION
  Delete multiple legacy in-app products

USAGE
  batch-delete

FLAGS
  -confirm=false        Confirm deleting the in-app products (required)
  -input string         Path to in-app products batch delete JSON payload (use - for stdin)
  -package-name string  Package name
```

## `gpc iap delete --help`

```text
DESCRIPTION
  Delete a legacy in-app product

USAGE
  delete

FLAGS
  -confirm=false        Confirm deleting the in-app product (required)
  -package-name string  Package name
  -sku string           In-app product SKU
```

## `gpc listing --help`

```text
DESCRIPTION
  Store listing workflows

USAGE
  listing

SUBCOMMANDS
  sync  Sync store listing metadata and images from a directory
```

## `gpc listing sync --help`

```text
DESCRIPTION
  Sync store listing metadata and images from a directory

USAGE
  sync

FLAGS
  -confirm=false         Confirm committing the edit (required unless --dry-run)
  -delete-missing=false  Delete remote locales that do not exist locally
  -dir string            Listings directory root
  -dry-run=false         Create and validate the edit, then delete it instead of mutating Play
  -package-name string   Package name
```

## `gpc purchases --help`

```text
DESCRIPTION
  Manage one-time and subscription purchases

USAGE
  purchases

SUBCOMMANDS
  products              Inspect and mutate one-time product purchases
  products-v2           Inspect one-time product purchases via Purchases.Productsv2
  subscriptions         Inspect and mutate subscription purchases
  subscriptions-legacy  Inspect and mutate legacy subscription purchases
  voided                Inspect voided purchases
```

## `gpc purchases products --help`

```text
DESCRIPTION
  Inspect and mutate one-time product purchases

USAGE
  products

SUBCOMMANDS
  get          Get one-time product purchase details
  acknowledge  Acknowledge a one-time product purchase
  consume      Consume a one-time product purchase
```

## `gpc purchases products get --help`

```text
DESCRIPTION
  Get one-time product purchase details

USAGE
  get

FLAGS
  -package-name string  Package name
  -product-id string    One-time product ID
  -token string         Purchase token
```

## `gpc purchases products acknowledge --help`

```text
DESCRIPTION
  Acknowledge a one-time product purchase

USAGE
  acknowledge

FLAGS
  -developer-payload string  Optional developer payload
  -package-name string       Package name
  -product-id string         One-time product ID
  -token string              Purchase token
```

## `gpc purchases products consume --help`

```text
DESCRIPTION
  Consume a one-time product purchase

USAGE
  consume

FLAGS
  -confirm=false        Confirm consuming the purchase (required)
  -package-name string  Package name
  -product-id string    One-time product ID
  -token string         Purchase token
```

## `gpc purchases products-v2 --help`

```text
DESCRIPTION
  Inspect one-time product purchases via Purchases.Productsv2

USAGE
  products-v2

SUBCOMMANDS
  get  Get one-time product purchase details (v2)
```

## `gpc purchases products-v2 get --help`

```text
DESCRIPTION
  Get one-time product purchase details (v2)

USAGE
  get

FLAGS
  -package-name string  Package name
  -token string         Purchase token
```

## `gpc purchases subscriptions --help`

```text
DESCRIPTION
  Inspect and mutate subscription purchases

USAGE
  subscriptions

SUBCOMMANDS
  get     Get subscription purchase details
  cancel  Cancel a subscription purchase
  defer   Defer a subscription renewal
  revoke  Revoke a subscription purchase
```

## `gpc purchases subscriptions get --help`

```text
DESCRIPTION
  Get subscription purchase details

USAGE
  get

FLAGS
  -package-name string  Package name
  -token string         Purchase token
```

## `gpc purchases subscriptions cancel --help`

```text
DESCRIPTION
  Cancel a subscription purchase

USAGE
  cancel

FLAGS
  -cancellation-type USER_REQUESTED_STOP_RENEWALS  Cancellation type: USER_REQUESTED_STOP_RENEWALS or DEVELOPER_REQUESTED_STOP_PAYMENTS
  -confirm=false                                   Confirm canceling the subscription purchase (required)
  -package-name string                             Package name
  -token string                                    Purchase token
```

## `gpc purchases subscriptions defer --help`

```text
DESCRIPTION
  Defer a subscription renewal

USAGE
  defer

FLAGS
  -confirm=false          Confirm deferring the subscription purchase (required unless --validate-only)
  -defer-duration string  Deferral duration (protobuf format, for example 604800s)
  -etag string            Current subscription etag from purchases subscriptions get
  -package-name string    Package name
  -token string           Purchase token
  -validate-only=false    Validate deferral request without applying changes
```

## `gpc purchases subscriptions revoke --help`

```text
DESCRIPTION
  Revoke a subscription purchase

USAGE
  revoke

FLAGS
  -confirm=false        Confirm revoking the subscription purchase (required)
  -package-name string  Package name
  -refund-type full     Refund type: full or prorated
  -token string         Purchase token
```

## `gpc purchases subscriptions-legacy --help`

```text
DESCRIPTION
  Inspect and mutate legacy subscription purchases

USAGE
  subscriptions-legacy

SUBCOMMANDS
  get          Get legacy subscription purchase details
  acknowledge  Acknowledge a legacy subscription purchase
  cancel       Cancel a legacy subscription purchase
  defer        Defer a legacy subscription renewal
  refund       Refund a legacy subscription purchase
  revoke       Revoke a legacy subscription purchase
```

## `gpc purchases subscriptions-legacy get --help`

```text
DESCRIPTION
  Get legacy subscription purchase details

USAGE
  get

FLAGS
  -package-name string     Package name
  -subscription-id string  Legacy subscription product ID
  -token string            Purchase token
```

## `gpc purchases subscriptions-legacy acknowledge --help`

```text
DESCRIPTION
  Acknowledge a legacy subscription purchase

USAGE
  acknowledge

FLAGS
  -developer-payload string  Optional developer payload
  -package-name string       Package name
  -subscription-id string    Legacy subscription product ID
  -token string              Purchase token
```

## `gpc purchases subscriptions-legacy cancel --help`

```text
DESCRIPTION
  Cancel a legacy subscription purchase

USAGE
  cancel

FLAGS
  -confirm=false           Confirm canceling the legacy subscription purchase (required)
  -package-name string     Package name
  -subscription-id string  Legacy subscription product ID
  -token string            Purchase token
```

## `gpc purchases subscriptions-legacy defer --help`

```text
DESCRIPTION
  Defer a legacy subscription renewal

USAGE
  defer

FLAGS
  -confirm=false                       Confirm deferring the legacy subscription purchase (required)
  -desired-expiry-time-millis string   Desired next expiry time in milliseconds since epoch
  -expected-expiry-time-millis string  Current expiry time in milliseconds since epoch
  -package-name string                 Package name
  -subscription-id string              Legacy subscription product ID
  -token string                        Purchase token
```

## `gpc purchases subscriptions-legacy refund --help`

```text
DESCRIPTION
  Refund a legacy subscription purchase

USAGE
  refund

FLAGS
  -confirm=false           Confirm refunding the legacy subscription purchase (required)
  -package-name string     Package name
  -subscription-id string  Legacy subscription product ID
  -token string            Purchase token
```

## `gpc purchases subscriptions-legacy revoke --help`

```text
DESCRIPTION
  Revoke a legacy subscription purchase

USAGE
  revoke

FLAGS
  -confirm=false           Confirm revoking the legacy subscription purchase (required)
  -package-name string     Package name
  -subscription-id string  Legacy subscription product ID
  -token string            Purchase token
```

## `gpc purchases voided --help`

```text
DESCRIPTION
  Inspect voided purchases

USAGE
  voided

SUBCOMMANDS
  list  List voided purchases
```

## `gpc purchases voided list --help`

```text
DESCRIPTION
  List voided purchases

USAGE
  list

FLAGS
  -end-time 0                                   End time in milliseconds since epoch
  -include-quantity-based-partial-refund=false  Include quantity-based partial refunds
  -max-results 0                                Maximum number of results
  -package-name string                          Package name
  -start-index 0                                Index of first result (indexed pagination)
  -start-time 0                                 Start time in milliseconds since epoch
  -token string                                 Token from previous paginated response
  -type 0                                       0 for one-time products, 1 for products and subscriptions
```

## `gpc users --help`

```text
DESCRIPTION
  Manage Play Console account users

USAGE
  users

SUBCOMMANDS
  list    List account users
  create  Create an account user
  update  Update an account user
  delete  Delete an account user
```

## `gpc users list --help`

```text
DESCRIPTION
  List account users

USAGE
  list

FLAGS
  -developer-id string  Developer account ID (numeric or developers/<id>)
  -page-size -1         Maximum users per page (-1 uses API default)
  -page-token string    Page token for the next page
```

## `gpc users create --help`

```text
DESCRIPTION
  Create an account user

USAGE
  create

FLAGS
  -developer-id string  Developer account ID (numeric or developers/<id>)
  -input string         Path to user JSON payload (use - for stdin)
```

## `gpc users update --help`

```text
DESCRIPTION
  Update an account user

USAGE
  update

FLAGS
  -developer-id string  Developer account ID (numeric or developers/<id>)
  -input string         Path to user JSON payload (use - for stdin)
  -name string          User resource name (developers/<developer-id>/users/<email>)
  -update-mask string   Comma-separated list of fields to update
  -user-email string    User email for resource name synthesis
```

## `gpc users delete --help`

```text
DESCRIPTION
  Delete an account user

USAGE
  delete

FLAGS
  -confirm=false        Confirm deleting the user (required)
  -developer-id string  Developer account ID (numeric or developers/<id>)
  -name string          User resource name (developers/<developer-id>/users/<email>)
  -user-email string    User email for resource name synthesis
```

## `gpc validate --help`

```text
DESCRIPTION
  Run pre-submission validation checks

USAGE
  validate

FLAGS
  -aab string                            Path to prebuilt .aab for artifact validation
  -build-task :app:bundleStagingRelease  Gradle build task for release bundle
  -edit-id string                        Existing edit ID to validate
  -notes-file string                     Release notes file path when notes-mode=file
  -notes-locale en-US                    Release notes locale
  -notes-mode git                        Release notes mode: git, file, none
  -notes-text string                     Inline release notes text override
  -package-name string                   Target package name
  -probe-track=false                     Create temporary edit and probe target track
  -project-dir .                         Android project directory
  -track alpha                           Target track name
```

## `gpc workflow --help`

```text
DESCRIPTION
  Run declarative gpc workflows from .gpc/workflow.yml

USAGE
  workflow

SUBCOMMANDS
  run  Execute or plan a workflow manifest
```

## `gpc workflow run --help`

```text
DESCRIPTION
  Execute or plan a workflow manifest

USAGE
  run

FLAGS
  -confirm=false  Execute the workflow steps
  -dry-run=false  Plan the workflow without running any steps
  -file string    Workflow file path (defaults to nearest .gpc/workflow.yml or .gpc/workflow.yaml)
  -output string  Output format: json or table
  -var value      Workflow variable override in key=value form (repeatable)
```

## `gpc grants --help`

```text
DESCRIPTION
  Manage per-app user grants

USAGE
  grants

SUBCOMMANDS
  create  Create a grant under a user
  update  Update a grant
  delete  Delete a grant
```

## `gpc grants create --help`

```text
DESCRIPTION
  Create a grant under a user

USAGE
  create

FLAGS
  -developer-id string  Developer account ID (numeric or developers/<id>)
  -input string         Path to grant JSON payload (use - for stdin)
  -parent string        User resource name (developers/<developer-id>/users/<email>)
  -user-email string    User email for parent name synthesis (requires --developer-id or stored auth developer ID)
```

## `gpc grants update --help`

```text
DESCRIPTION
  Update a grant

USAGE
  update

FLAGS
  -developer-id string  Developer account ID (numeric or developers/<id>)
  -input string         Path to grant JSON payload (use - for stdin)
  -name string          Grant resource name (developers/<developer-id>/users/<email>/grants/<package-name>)
  -package-name string  Package name for grant name synthesis
  -update-mask string   Comma-separated list of fields to update
  -user-email string    User email for grant name synthesis
```

## `gpc grants delete --help`

```text
DESCRIPTION
  Delete a grant

USAGE
  delete

FLAGS
  -confirm=false        Confirm deleting the grant (required)
  -developer-id string  Developer account ID (numeric or developers/<id>)
  -name string          Grant resource name (developers/<developer-id>/users/<email>/grants/<package-name>)
  -package-name string  Package name for grant name synthesis
  -user-email string    User email for grant name synthesis
```

## `gpc internal-sharing --help`

```text
DESCRIPTION
  Upload artifacts for internal app sharing

USAGE
  internal-sharing

SUBCOMMANDS
  upload  Upload one APK or AAB for internal app sharing
```

## `gpc internal-sharing upload --help`

```text
DESCRIPTION
  Upload one APK or AAB for internal app sharing

USAGE
  upload

FLAGS
  -aab string           Path to .aab file
  -apk string           Path to .apk file
  -package-name string  Package name
```

## `gpc integrity --help`

```text
DESCRIPTION
  Decode and inspect Play Integrity tokens

USAGE
  integrity

SUBCOMMANDS
  decode  Decode a Play Integrity token
```

## `gpc integrity decode --help`

```text
DESCRIPTION
  Decode a Play Integrity token

USAGE
  decode

FLAGS
  -input string         Path to a file containing the integrity token (use - for stdin)
  -package-name string  Package name
  -token string         Integrity token
```

## `gpc update --help`

```text
DESCRIPTION
  Check for and install newer gpc releases

USAGE
  update

FLAGS
  -check=false     Check for updates without downloading or replacing the binary
  -confirm=false   Download and replace the current gpc binary
  -version string  Install a specific release tag (for example v0.3.0)
```

## `gpc completion --help`

```text
DESCRIPTION
  Generate shell completion script

USAGE
  completion

SUBCOMMANDS
  bash    Print bash completion script
  zsh     Print zsh completion script
  fish    Print fish completion script
  values  Print dynamic completion values
```

## `gpc completion bash --help`

```text
DESCRIPTION
  Print bash completion script

USAGE
  bash
```

## `gpc completion zsh --help`

```text
DESCRIPTION
  Print zsh completion script

USAGE
  zsh
```

## `gpc completion fish --help`

```text
DESCRIPTION
  Print fish completion script

USAGE
  fish
```

## `gpc completion values --help`

```text

```
