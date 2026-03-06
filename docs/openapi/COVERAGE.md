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
| Implemented endpoints | 126 |
| Missing endpoints | 10 |
| Detected service method IDs | 126 |
| Unmatched service method IDs | 0 |

## Family Summary

| Family | Implemented | Missing | Total |
| --- | ---: | ---: | ---: |
| `applications.dataSafety` | 1 | 0 | 1 |
| `applications.deviceTierConfigs` | 3 | 0 | 3 |
| `apprecovery` | 5 | 0 | 5 |
| `edits` | 5 | 0 | 5 |
| `edits.apks` | 2 | 1 | 3 |
| `edits.bundles` | 2 | 0 | 2 |
| `edits.countryavailability` | 1 | 0 | 1 |
| `edits.deobfuscationfiles` | 1 | 0 | 1 |
| `edits.details` | 3 | 0 | 3 |
| `edits.expansionfiles` | 4 | 0 | 4 |
| `edits.images` | 4 | 0 | 4 |
| `edits.listings` | 6 | 0 | 6 |
| `edits.testers` | 3 | 0 | 3 |
| `edits.tracks` | 5 | 0 | 5 |
| `externaltransactions` | 3 | 0 | 3 |
| `generatedapks` | 2 | 0 | 2 |
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
| `systemapks` | 4 | 0 | 4 |
| `users` | 4 | 0 | 4 |

## Missing Endpoints

### `edits.apks`

- `androidpublisher.edits.apks.addexternallyhosted` | `POST` `androidpublisher/v3/applications/{packageName}/edits/{editId}/apks/externallyHosted`

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

## Unmatched Service Method IDs

No unmatched service method IDs detected.
