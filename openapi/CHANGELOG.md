# API Changelog

This documents changes to the API version. The S3 Broker code is versioned separately. See the top-level [CHANGELOG](../CHANGELOG.md) for info about software versions.


## 0.2.0

Significant change to the API to make it compatible with other software besides SciCat.

- Add `/urls` endpoint, indended to replace `/datasets/urls`. Rename the `pid` parameter to `id`. The old endpoint is kept but moved to the 'SciCat' section.
- Rename `/datasets/s3-creds` to `/s3-creds`.


## 0.1.0

Initial API version.