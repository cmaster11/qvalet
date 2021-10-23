# Database

For certain plugins, like the [`schedule`](./0110-plugins/schedule.md) one, you need to set up a PostgreSQL-compatible
database connection.

## Configuration

Each listener can have its own database configuration, but if you plan on using only one database you can then place the
configuration in the defaults section.

[filename](../pkg/database.go ':include :type=code :fragment=database-docs')

## Example (schedule plugin)

> Example code at: [`/examples/config.plugin.schedule.yaml`](https://github.com/cmaster11/go-to-exec/tree/main/examples/config.plugin.schedule.yaml)

[filename](../examples/config.plugin.schedule.yaml ':include :type=code')