# @solashi/go-mysql-binlog-node

A library wrapping the [go-mysql](https://github.com/go-mysql-org/go-mysql) package, providing a MySQL client connector and binlog parsing implementation.

> This project is forked from [go-mysql-js](https://github.com/go-mysql-org/go-mysql-js)

## Installation

```bash
npm i --save @solashi/go-mysql-binlog-node
```

* Enable MySQL binlog in `my.cnf`, restart MySQL server after making the changes.
  > binlog checksum is enabled by default. @solashi/go-mysql-binlog-node can work with it, but it doesn't really verify it.

  ```
  # Must be unique integer from 1-2^32
  server-id        = 1
  # Row format required
  binlog_format    = row
  # Directory must exist. This path works for Linux. Other OS may require
  #   different path.
  log_bin          = /var/log/mysql/mysql-bin.log

  binlog_do_db     = employees   # Optional, limit which databases to log
  expire_logs_days = 10          # Optional, purge old logs
  max_binlog_size  = 100M        # Optional, limit log size
  ```

* Create an account with replication privileges, e.g. given privileges to account `root` (or any account that you use to read binary logs)

  ```sql
  GRANT REPLICATION SLAVE, REPLICATION CLIENT, SELECT ON *.* TO 'root'@'localhost'
  ```

## Example

```js
import MysqlBinlog from '@solashi/go-mysql-binlog-node';

async function main() {
    let syncer = await MysqlBinlog.create({
        hostname: "localhost",
        port: 3306,
        username: "root",
        password: "mypassword",
        tableRegexes: ['Users'],
    });
    syncer.on('event', (event) => {
        console.log('got row event', event);
    });
    syncer.on('error', (err) => {
        console.error('got error', err);
    });

    process.on('SIGINT', function() {
        console.log("Caught interrupt signal");
        syncer.close();
    });
}

main().catch(err => {
    console.error(err);
    process.exit(1);
});
```