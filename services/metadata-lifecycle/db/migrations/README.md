# Mapping to EPOS-DCAT-AP v3.?

## Web Service -> Data Service

| Our                                                   | DCAT                       | What                           |
| ----------------------------------------------------- | -------------------------- | ------------------------------ |
| use the root of the template                          | `dcat:endpointURL`         | the base url of the service    |
| use the distribution title                            | `dct:title`                |                                |
| `documentation_uri`                                   | `dct:conformsTo`           | url of the documentation       |
| connected contact points                              | `dcat:contactPoint`        |                                |
| use the distribution description                      | `dct:description`          |                                |
| there is no direct mapping but it's a separate object | `dcat:endpointDescription` | the reference to the operation |
| `keywords`                                            | `dcat:keyword`             |                                |
| `provider`                                            | `dct:publisher`            |                                |
| `format` for generic web services                     | `dct:format`               |                                |

## Distribution -> Distribution

| Our                                                                                    | DCAT                 | What                              |
| -------------------------------------------------------------------------------------- | -------------------- | --------------------------------- |
| there is no direct mapping but it's a separate object                                  | `dcat:accessURL`     | the reference to the operation    |
| `description`                                                                          | `dct:description`    |                                   |
| `format` for generic web services or `downloadable_file_format` for downloadable files | `dct:format`         |                                   |
| `license`                                                                              | `dct:license`        |                                   |
| there is no direct mapping but it's a separate object                                  | `dcat:accessService` | the reference to the Data Service |
| `byte_size`                                                                            | `dcat:byteSize`      |                                   |
|                                                                                        | `dct:conformsTo`     | ?                                 |
| `downloadable_file_url`                                                                | `dcat:downloadURL`   |                                   |
| `title`                                                                                | `dct:title`          |                                   |

## Dataset -> Dataset

| Our                                                   | DCAT                             | What                                 |
| ----------------------------------------------------- | -------------------------------- | ------------------------------------ |
| `description`                                         | `dct:description`                |                                      |
| `title`                                               | `dct:title`                      |                                      |
| the connected contact points to a dataset             | `dcat:contactPoint`              |                                      |
| there is no direct mapping but it's a separate object | `dcat:distribution`              | the reference to the distribution(s) |
| `keywords`                                            | `dcat:keyword`                   |                                      |
| `providers`                                           | `dct:publisher`                  |                                      |
| `spatial`                                             | `dct:spatial`                    |                                      |
| `temporal`                                            | `dct:temporal`                   |                                      |
| linked categories                                     | `dcat:theme`                     |                                      |
| `access_rights`                                       | `dct:accessRights`               |                                      |
| `accrual_periodicity`                                 | `dct:accrualPeriodicity`         |                                      |
| `conforms_to`                                         | `dct:conformsTo`                 |                                      |
| ignore for now                                        | `dct:creator`                    |                                      |
| `quality_annotation`                                  | `dqv:hasQualityAnnotation`       | Absolute HTTP(S) URL                 |
| the connected identifiers to a dataset                | `adms:identifier`                |                                      |
| `created_at`                                          | `dct:issued`                     |                                      |
| `updated_at`                                          | `dct:modified`                   |                                      |
| `provenance`                                          | `dct:provenance`                 |                                      |
| `spatial_resolution_in_meters`                        | `dcat:spatialResolutionInMeters` |                                      |
| `temporal_resolution`                                 | `dcat:temporalResolution`        |                                      |
| `type`                                                | `dct:type`                       |                                      |
| `version`                                             | `dcat:version`                   |                                      |
