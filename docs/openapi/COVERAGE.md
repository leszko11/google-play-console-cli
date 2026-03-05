# OpenAPI Coverage

This file is generated from:

- `docs/openapi/paths.txt`
- live Google API service calls detected under `internal/gpc`

To regenerate:

```bash
make generate-openapi-coverage
```

## Summary

| Metric | Count |
| --- | ---: |
| Total discovery endpoints | 136 |
| Implemented endpoints | 102 |
| Missing endpoints | 34 |
| Detected service method IDs | 102 |
| Unmatched service method IDs | 0 |

## Family Summary

| Family | Implemented | Missing | Total |
| --- | ---: | ---: | ---: |
| `applications.dataSafety` | 0 | 1 | 1 |
| `applications.deviceTierConfigs` | 0 | 3 | 3 |
| `apprecovery` | 0 | 5 | 5 |
| `edits` | 5 | 0 | 5 |
| `edits.apks` | 2 | 1 | 3 |
| `edits.bundles` | 2 | 0 | 2 |
| `edits.countryavailability` | 1 | 0 | 1 |
| `edits.deobfuscationfiles` | 1 | 0 | 1 |
| `edits.details` | 2 | 1 | 3 |
| `edits.expansionfiles` | 0 | 4 | 4 |
| `edits.images` | 4 | 0 | 4 |
| `edits.listings` | 5 | 1 | 6 |
| `edits.testers` | 2 | 1 | 3 |
| `edits.tracks` | 3 | 2 | 5 |
| `externaltransactions` | 3 | 0 | 3 |
| `generatedapks` | 0 | 2 | 2 |
| `grants` | 3 | 0 | 3 |
| `inappproducts` | 8 | 1 | 9 |
| `internalappsharingartifacts` | 2 | 0 | 2 |
| `monetization.convertRegionPrices` | 1 | 0 | 1 |
| `monetization.onetimeproducts` | 17 | 0 | 17 |
| `monetization.subscriptions` | 22 | 2 | 24 |
| `orders` | 3 | 0 | 3 |
| `purchases.products` | 3 | 0 | 3 |
| `purchases.productsv2` | 1 | 0 | 1 |
| `purchases.subscriptions` | 0 | 6 | 6 |
| `purchases.subscriptionsv2` | 4 | 0 | 4 |
| `purchases.voidedpurchases` | 1 | 0 | 1 |
| `reviews` | 3 | 0 | 3 |
| `systemapks` | 0 | 4 | 4 |
| `users` | 4 | 0 | 4 |

## Missing Endpoints

### `applications.dataSafety`

- `androidpublisher.applications.dataSafety` | `POST` `androidpublisher/v3/applications/{packageName}/dataSafety`

### `applications.deviceTierConfigs`

- `androidpublisher.applications.deviceTierConfigs.list` | `GET` `androidpublisher/v3/applications/{packageName}/deviceTierConfigs`
- `androidpublisher.applications.deviceTierConfigs.get` | `GET` `androidpublisher/v3/applications/{packageName}/deviceTierConfigs/{deviceTierConfigId}`
- `androidpublisher.applications.deviceTierConfigs.create` | `POST` `androidpublisher/v3/applications/{packageName}/deviceTierConfigs`

### `apprecovery`

- `androidpublisher.apprecovery.list` | `GET` `androidpublisher/v3/applications/{packageName}/appRecoveries`
- `androidpublisher.apprecovery.create` | `POST` `androidpublisher/v3/applications/{packageName}/appRecoveries`
- `androidpublisher.apprecovery.addTargeting` | `POST` `androidpublisher/v3/applications/{packageName}/appRecoveries/{appRecoveryId}:addTargeting`
- `androidpublisher.apprecovery.cancel` | `POST` `androidpublisher/v3/applications/{packageName}/appRecoveries/{appRecoveryId}:cancel`
- `androidpublisher.apprecovery.deploy` | `POST` `androidpublisher/v3/applications/{packageName}/appRecoveries/{appRecoveryId}:deploy`

### `edits.apks`

- `androidpublisher.edits.apks.addexternallyhosted` | `POST` `androidpublisher/v3/applications/{packageName}/edits/{editId}/apks/externallyHosted`

### `edits.details`

- `androidpublisher.edits.details.update` | `PUT` `androidpublisher/v3/applications/{packageName}/edits/{editId}/details`

### `edits.expansionfiles`

- `androidpublisher.edits.expansionfiles.get` | `GET` `androidpublisher/v3/applications/{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}`
- `androidpublisher.edits.expansionfiles.patch` | `PATCH` `androidpublisher/v3/applications/{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}`
- `androidpublisher.edits.expansionfiles.upload` | `POST` `androidpublisher/v3/applications/{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}`
- `androidpublisher.edits.expansionfiles.update` | `PUT` `androidpublisher/v3/applications/{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}`

### `edits.listings`

- `androidpublisher.edits.listings.update` | `PUT` `androidpublisher/v3/applications/{packageName}/edits/{editId}/listings/{language}`

### `edits.testers`

- `androidpublisher.edits.testers.update` | `PUT` `androidpublisher/v3/applications/{packageName}/edits/{editId}/testers/{track}`

### `edits.tracks`

- `androidpublisher.edits.tracks.patch` | `PATCH` `androidpublisher/v3/applications/{packageName}/edits/{editId}/tracks/{track}`
- `androidpublisher.edits.tracks.create` | `POST` `androidpublisher/v3/applications/{packageName}/edits/{editId}/tracks`

### `generatedapks`

- `androidpublisher.generatedapks.list` | `GET` `androidpublisher/v3/applications/{packageName}/generatedApks/{versionCode}`
- `androidpublisher.generatedapks.download` | `GET` `androidpublisher/v3/applications/{packageName}/generatedApks/{versionCode}/downloads/{downloadId}:download`

### `inappproducts`

- `androidpublisher.inappproducts.update` | `PUT` `androidpublisher/v3/applications/{packageName}/inappproducts/{sku}`

### `monetization.subscriptions`

- `androidpublisher.monetization.subscriptions.basePlans.activate` | `POST` `androidpublisher/v3/applications/{packageName}/subscriptions/{productId}/basePlans/{basePlanId}:activate`
- `androidpublisher.monetization.subscriptions.basePlans.deactivate` | `POST` `androidpublisher/v3/applications/{packageName}/subscriptions/{productId}/basePlans/{basePlanId}:deactivate`

### `purchases.subscriptions`

- `androidpublisher.purchases.subscriptions.get` | `GET` `androidpublisher/v3/applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}`
- `androidpublisher.purchases.subscriptions.acknowledge` | `POST` `androidpublisher/v3/applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:acknowledge`
- `androidpublisher.purchases.subscriptions.cancel` | `POST` `androidpublisher/v3/applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:cancel`
- `androidpublisher.purchases.subscriptions.defer` | `POST` `androidpublisher/v3/applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:defer`
- `androidpublisher.purchases.subscriptions.refund` | `POST` `androidpublisher/v3/applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:refund`
- `androidpublisher.purchases.subscriptions.revoke` | `POST` `androidpublisher/v3/applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:revoke`

### `systemapks`

- `androidpublisher.systemapks.variants.list` | `GET` `androidpublisher/v3/applications/{packageName}/systemApks/{versionCode}/variants`
- `androidpublisher.systemapks.variants.get` | `GET` `androidpublisher/v3/applications/{packageName}/systemApks/{versionCode}/variants/{variantId}`
- `androidpublisher.systemapks.variants.download` | `GET` `androidpublisher/v3/applications/{packageName}/systemApks/{versionCode}/variants/{variantId}:download`
- `androidpublisher.systemapks.variants.create` | `POST` `androidpublisher/v3/applications/{packageName}/systemApks/{versionCode}/variants`

## Unmatched Service Method IDs

No unmatched service method IDs detected.
