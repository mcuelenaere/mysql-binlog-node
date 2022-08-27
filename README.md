# go-mysql-js

A library wrapping the [go-mysql](https://github.com/go-mysql-org/go-mysql) package, providing a MySQL client connector and binlog parsing implementation.

## Installation

```bash
npm i --save go-mysql-js
```

## Example

```js
import MysqlBinlog from 'go-mysql-js';

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