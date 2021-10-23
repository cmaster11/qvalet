# Schedule

If you want to schedule a command execution in the future, you can use the `schedule` plugin!

Whenever you add a `schedule` plugin to a listener, you will have a new `/schedule` route, which you can use to schedule
the command associated with the listener in the future.

You can use any of the following syntax's:

* `curl "http://localhost:7055/hello/schedule/10s"`: schedule the execution of the command 10 seconds in the future.
  This uses the Go `Duration` syntax, which you can extensively explore in
  the [Go documentation](https://pkg.go.dev/time#ParseDuration).
* `curl "http://localhost:7055/hello/schedule/1634917558"`: schedule the execution of the command at a specific point in
  time, defined by a [Unix timestamp](https://www.unixtimestamp.com/) (expressed in seconds).
* `curl "http://localhost:7055/hello/schedule/1634917558123"`: schedule the execution of the command at a specific point
  in time, defined by a [Unix timestamp](https://www.unixtimestamp.com/) (expressed in milliseconds).
* Additionally, the `schedule` plugin supports all the ISO layouts defined in
  the [Go documentation of the `time` package](https://pkg.go.dev/time#pkg-constants),
  like `curl "http://localhost:7055/hello/schedule/2021-10-23T05:18:37+00:00"`.

Also, while evaluating the command templates, you will have access to the `__gteScheduleTime` key, which contains the
command execution time. This field is of [`time.Time`](https://pkg.go.dev/time#Time) type, which means you can e.g.
extract the Unix milliseconds value using `{{ .__gteScheduleTime.UnixMilli }}`.

## Configuration

[filename](../../pkg/plugin_schedule.go ':include :type=code :fragment=config')

NOTE: to use this plugin, you **have to** set up a [database](/0090-database.md) connection.

## Examples

> Example code at: [`/examples/config.plugin.schedule.yaml`](https://github.com/cmaster11/go-to-exec/tree/main/examples/config.plugin.schedule.yaml)

This is a simple example on how to use the schedule plugin:

[filename](../../examples/config.plugin.schedule.yaml ':include :type=code')
