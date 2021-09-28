# Storage

Every listener can be configured to store to different services (S3, GCP, azblob, etc…):

* The payloads they receive, which means the dump of the HTTP request (`args` key).
* The executed command, its arguments and environment variables (`command` key).
* The command output (`output` key).

`go-to-exec` uses the amazing [`go-storage`](https://github.com/beyondstorage/go-storage) library,
which [supports](https://beyondstorage.io/docs/go-storage/services/index) a broad variety of storage
destinations. `go-to-exec` tries to support all the storage options in the "Stable" category. If you notice there is a
missing library, please open an [issue](https://github.com/cmaster11/go-to-exec/issues)!

[filename](../pkg/storage.go ':include :type=code :fragment=storage-docs')

Here is an example, which uses GCS and FS (file-system) as a storage backend:

[filename](../examples/config.storage.yaml ':include :type=code')