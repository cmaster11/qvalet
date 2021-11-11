<a name="v2.0.2"></a>
# [v2.0.2](https://github.com/cmaster11/qvalet/releases/tag/v2.0.2) - 11 Nov 2021



[Changes][v2.0.2]


<a name="v2.0.1"></a>
# [v2.0.1](https://github.com/cmaster11/qvalet/releases/tag/v2.0.1) - 11 Nov 2021

fix: bad validation flag for listener methods

[Changes][v2.0.1]


<a name="v2.0.0"></a>
# [v2.0.0](https://github.com/cmaster11/qvalet/releases/tag/v2.0.0) - 30 Oct 2021

**qValet** replaces the old `go-to-exec`!

Some changes:

* CORS support for HTTP response plugin
* Log by default to `os.Stdout` instead of `os.Stderr`
* Upgraded some dependencies

BREAKING:

* The repository URL is now `https://github.com/cmaster11/qvalet`
* All new executables qill be prefixed with `qvalet`, e.g. `qvalet-linux-amd64`
* The new Docker image will be `cmaster11/qvalet`
* All environment variables' prefix is now `QV_` (instead of the old `GTE_`)
* The old `__gteHeaders` payload object is now part of the `__qvRequest` object, as `__qvRequest.Headers`.
* Prefix changes:
  * `__gtePayloadArrayLength` -> `__qvPayloadArrayLength`
  * `__gteResult` -> `__qvResult`
  * `__gteRetry` -> `__qvRetry`
  * `__gteScheduleTime` -> `__qvScheduleTime`
  * `__gteApiKey` -> `__qvApiKey`
  * `__gtePayloadArrayLength` -> `__qvPayloadArrayLength`



[Changes][v2.0.0]


<a name="v1.3.5"></a>
# [v1.3.5](https://github.com/cmaster11/qvalet/releases/tag/v1.3.5) - 25 Oct 2021

fix: When using a defaults file, env variables are now mapped using the `GTE_DEFAULTS_` prefix

[Changes][v1.3.5]


<a name="v1.3.4"></a>
# [v1.3.4](https://github.com/cmaster11/qvalet/releases/tag/v1.3.4) - 23 Oct 2021

better pgsql dsn support (e.g. unix socket)

[Changes][v1.3.4]


<a name="v1.3.3"></a>
# [v1.3.3](https://github.com/cmaster11/qvalet/releases/tag/v1.3.3) - 23 Oct 2021

feat: schedule plugin now supports ISO date formats, and the id config option is required

Also, sunrise example!

[Changes][v1.3.3]


<a name="v1.3.2"></a>
# [v1.3.2](https://github.com/cmaster11/qvalet/releases/tag/v1.3.2) - 23 Oct 2021

Schedule plugin (#25)

[Changes][v1.3.2]


<a name="v1.3.1"></a>
# [v1.3.1](https://github.com/cmaster11/qvalet/releases/tag/v1.3.1) - 12 Oct 2021

Retry plugin (#22)

[Changes][v1.3.1]


<a name="v1.3.0"></a>
# [v1.3.0](https://github.com/cmaster11/qvalet/releases/tag/v1.3.0) - 28 Sep 2021

breaking: renamed `__gtePayloadLength` to `__gtePayloadArrayLength` (#8)

docs: example for array payload, some sorting of docs

others: some refactoring

[Changes][v1.3.0]


<a name="v1.2.8"></a>
# [v1.2.8](https://github.com/cmaster11/qvalet/releases/tag/v1.2.8) - 26 Sep 2021

fix: properly support multipart form data

feat: support yaml payloads

[Changes][v1.2.8]


<a name="v1.2.7"></a>
# [v1.2.7](https://github.com/cmaster11/qvalet/releases/tag/v1.2.7) - 26 Sep 2021

Preview plugin (#7)

Also, authentication api keys can now be loaded from environment variables.

[Changes][v1.2.7]


<a name="v1.2.6"></a>
# [v1.2.6](https://github.com/cmaster11/qvalet/releases/tag/v1.2.6) - 25 Sep 2021

Refactoring + fix bad cloning of listeners for plugins.

[Changes][v1.2.6]


<a name="v1.2.5"></a>
# [v1.2.5](https://github.com/cmaster11/qvalet/releases/tag/v1.2.5) - 25 Sep 2021

Initial support for HTTP response plugin (#6)

[Changes][v1.2.5]


<a name="v1.2.4"></a>
# [v1.2.4](https://github.com/cmaster11/qvalet/releases/tag/v1.2.4) - 23 Sep 2021

Initial support for AWS SNS plugin (#5)

[Changes][v1.2.4]


<a name="v1.2.3"></a>
# [v1.2.3](https://github.com/cmaster11/qvalet/releases/tag/v1.2.3) - 20 Aug 2021

Change filename order, so that error handlers will append `-error` AFTER the timestamp, for easier sorting.

[Changes][v1.2.3]


<a name="v1.2.2"></a>
# [v1.2.2](https://github.com/cmaster11/qvalet/releases/tag/v1.2.2) - 16 Aug 2021

Added storage cache (speed up startup)

[Changes][v1.2.2]


<a name="v1.2.1"></a>
# [v1.2.1](https://github.com/cmaster11/qvalet/releases/tag/v1.2.1) - 16 Aug 2021

Add `env` entry to log/store/return configs

[Changes][v1.2.1]


<a name="v1.2.0"></a>
# [v1.2.0](https://github.com/cmaster11/qvalet/releases/tag/v1.2.0) - 16 Aug 2021

Support for payload storage (#4)

* BREAKING CHANGE: grouped logging and return configuration:

  * `logArgs`, `logCommand`, `logOutput` are now `log: args,command,output,all`.
  * `returnOutput` is now: `return: output,all`

* Support for storage
* Some refactoring

[Changes][v1.2.0]


<a name="v1.1.10"></a>
# [v1.1.10](https://github.com/cmaster11/qvalet/releases/tag/v1.1.10) - 15 Aug 2021

Support multiple config files and dotenv files

[Changes][v1.1.10]


<a name="v1.1.9"></a>
# [v1.1.9](https://github.com/cmaster11/qvalet/releases/tag/v1.1.9) - 14 Aug 2021

Support a dotenv file to load environment variables

[Changes][v1.1.9]


<a name="v1.1.8"></a>
# [v1.1.8](https://github.com/cmaster11/qvalet/releases/tag/v1.1.8) - 14 Aug 2021

Test action (#3)

[Changes][v1.1.8]


<a name="v1.1.7"></a>
# [v1.1.7](https://github.com/cmaster11/qvalet/releases/tag/v1.1.7) - 14 Aug 2021



[Changes][v1.1.7]


<a name="v1.1.6"></a>
# [v1.1.6](https://github.com/cmaster11/qvalet/releases/tag/v1.1.6) - 14 Aug 2021



[Changes][v1.1.6]


<a name="v1.1.5"></a>
# [v1.1.5](https://github.com/cmaster11/qvalet/releases/tag/v1.1.5) - 13 Aug 2021



[Changes][v1.1.5]


<a name="v1.1.4"></a>
# [v1.1.4](https://github.com/cmaster11/qvalet/releases/tag/v1.1.4) - 12 Aug 2021



[Changes][v1.1.4]


<a name="v1.1.3"></a>
# [v1.1.3](https://github.com/cmaster11/qvalet/releases/tag/v1.1.3) - 11 Aug 2021



[Changes][v1.1.3]


<a name="v1.1.2"></a>
# [v1.1.2](https://github.com/cmaster11/qvalet/releases/tag/v1.1.2) - 11 Aug 2021



[Changes][v1.1.2]


<a name="v1.1.1"></a>
# [v1.1.1](https://github.com/cmaster11/qvalet/releases/tag/v1.1.1) - 10 Aug 2021



[Changes][v1.1.1]


<a name="v1.1.0"></a>
# [v1.1.0](https://github.com/cmaster11/qvalet/releases/tag/v1.1.0) - 10 Aug 2021



[Changes][v1.1.0]


<a name="v1.0.12"></a>
# [v1.0.12](https://github.com/cmaster11/qvalet/releases/tag/v1.0.12) - 10 Aug 2021



[Changes][v1.0.12]


<a name="v1.0.11"></a>
# [v1.0.11](https://github.com/cmaster11/qvalet/releases/tag/v1.0.11) - 10 Aug 2021



[Changes][v1.0.11]


<a name="v1.0.9"></a>
# [v1.0.9](https://github.com/cmaster11/qvalet/releases/tag/v1.0.9) - 10 Aug 2021



[Changes][v1.0.9]


<a name="v1.0.8"></a>
# [v1.0.8](https://github.com/cmaster11/qvalet/releases/tag/v1.0.8) - 09 Aug 2021



[Changes][v1.0.8]


<a name="v1.0.7"></a>
# [v1.0.7](https://github.com/cmaster11/qvalet/releases/tag/v1.0.7) - 09 Aug 2021



[Changes][v1.0.7]


<a name="v1.0.6"></a>
# [v1.0.6](https://github.com/cmaster11/qvalet/releases/tag/v1.0.6) - 09 Aug 2021



[Changes][v1.0.6]


<a name="v1.0.5"></a>
# [v1.0.5](https://github.com/cmaster11/qvalet/releases/tag/v1.0.5) - 06 Jul 2021



[Changes][v1.0.5]


<a name="v1.0.4"></a>
# [v1.0.4](https://github.com/cmaster11/qvalet/releases/tag/v1.0.4) - 06 Jul 2021



[Changes][v1.0.4]


<a name="v1.0.3"></a>
# [v1.0.3](https://github.com/cmaster11/qvalet/releases/tag/v1.0.3) - 05 Jul 2021



[Changes][v1.0.3]


<a name="v1.0.2"></a>
# [v1.0.2](https://github.com/cmaster11/qvalet/releases/tag/v1.0.2) - 04 Jul 2021



[Changes][v1.0.2]


<a name="v1.0.1"></a>
# [v1.0.1](https://github.com/cmaster11/qvalet/releases/tag/v1.0.1) - 30 May 2021




[Changes][v1.0.1]


<a name="v1.0.0"></a>
# [v1.0.0](https://github.com/cmaster11/qvalet/releases/tag/v1.0.0) - 30 May 2021



[Changes][v1.0.0]


[v2.0.2]: https://github.com/cmaster11/qvalet/compare/v2.0.1...v2.0.2
[v2.0.1]: https://github.com/cmaster11/qvalet/compare/v2.0.0...v2.0.1
[v2.0.0]: https://github.com/cmaster11/qvalet/compare/v1.3.5...v2.0.0
[v1.3.5]: https://github.com/cmaster11/qvalet/compare/v1.3.4...v1.3.5
[v1.3.4]: https://github.com/cmaster11/qvalet/compare/v1.3.3...v1.3.4
[v1.3.3]: https://github.com/cmaster11/qvalet/compare/v1.3.2...v1.3.3
[v1.3.2]: https://github.com/cmaster11/qvalet/compare/v1.3.1...v1.3.2
[v1.3.1]: https://github.com/cmaster11/qvalet/compare/v1.3.0...v1.3.1
[v1.3.0]: https://github.com/cmaster11/qvalet/compare/v1.2.8...v1.3.0
[v1.2.8]: https://github.com/cmaster11/qvalet/compare/v1.2.7...v1.2.8
[v1.2.7]: https://github.com/cmaster11/qvalet/compare/v1.2.6...v1.2.7
[v1.2.6]: https://github.com/cmaster11/qvalet/compare/v1.2.5...v1.2.6
[v1.2.5]: https://github.com/cmaster11/qvalet/compare/v1.2.4...v1.2.5
[v1.2.4]: https://github.com/cmaster11/qvalet/compare/v1.2.3...v1.2.4
[v1.2.3]: https://github.com/cmaster11/qvalet/compare/v1.2.2...v1.2.3
[v1.2.2]: https://github.com/cmaster11/qvalet/compare/v1.2.1...v1.2.2
[v1.2.1]: https://github.com/cmaster11/qvalet/compare/v1.2.0...v1.2.1
[v1.2.0]: https://github.com/cmaster11/qvalet/compare/v1.1.10...v1.2.0
[v1.1.10]: https://github.com/cmaster11/qvalet/compare/v1.1.9...v1.1.10
[v1.1.9]: https://github.com/cmaster11/qvalet/compare/v1.1.8...v1.1.9
[v1.1.8]: https://github.com/cmaster11/qvalet/compare/v1.1.7...v1.1.8
[v1.1.7]: https://github.com/cmaster11/qvalet/compare/v1.1.6...v1.1.7
[v1.1.6]: https://github.com/cmaster11/qvalet/compare/v1.1.5...v1.1.6
[v1.1.5]: https://github.com/cmaster11/qvalet/compare/v1.1.4...v1.1.5
[v1.1.4]: https://github.com/cmaster11/qvalet/compare/v1.1.3...v1.1.4
[v1.1.3]: https://github.com/cmaster11/qvalet/compare/v1.1.2...v1.1.3
[v1.1.2]: https://github.com/cmaster11/qvalet/compare/v1.1.1...v1.1.2
[v1.1.1]: https://github.com/cmaster11/qvalet/compare/v1.1.0...v1.1.1
[v1.1.0]: https://github.com/cmaster11/qvalet/compare/v1.0.12...v1.1.0
[v1.0.12]: https://github.com/cmaster11/qvalet/compare/v1.0.11...v1.0.12
[v1.0.11]: https://github.com/cmaster11/qvalet/compare/v1.0.9...v1.0.11
[v1.0.9]: https://github.com/cmaster11/qvalet/compare/v1.0.8...v1.0.9
[v1.0.8]: https://github.com/cmaster11/qvalet/compare/v1.0.7...v1.0.8
[v1.0.7]: https://github.com/cmaster11/qvalet/compare/v1.0.6...v1.0.7
[v1.0.6]: https://github.com/cmaster11/qvalet/compare/v1.0.5...v1.0.6
[v1.0.5]: https://github.com/cmaster11/qvalet/compare/v1.0.4...v1.0.5
[v1.0.4]: https://github.com/cmaster11/qvalet/compare/v1.0.3...v1.0.4
[v1.0.3]: https://github.com/cmaster11/qvalet/compare/v1.0.2...v1.0.3
[v1.0.2]: https://github.com/cmaster11/qvalet/compare/v1.0.1...v1.0.2
[v1.0.1]: https://github.com/cmaster11/qvalet/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/cmaster11/qvalet/tree/v1.0.0

 <!-- Generated by https://github.com/rhysd/changelog-from-release -->
