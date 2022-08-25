import { EventEmitter } from 'events';
import { debug } from 'debug';
import * as path from 'path';
const native = require('node-gyp-build')(path.join(__dirname, '..'));

const debugChannel = debug('mysql_binlog');
native.setLogger((message: string) => {
    debugChannel.log(message);
});

export interface BinlogPosition {
    name: string;
    position: number;
}

export interface Config {
    hostname: string;
    port: number;
    username: string;
    password: string;
    tableRegexes: string[];
    binlogPosition?: BinlogPosition;
}

class MysqlBinlog extends EventEmitter {
    private _native: any;

    constructor(config: Config) {
        super();
        this._native = new native.MysqlBinlog(
            config.hostname,
            config.port,
            config.username,
            config.password,
            config.tableRegexes,
            (jsonEvent: string) => {
                this.emit('event', JSON.parse(jsonEvent));
            },
            config.binlogPosition ?? null,
        );
    }

    public close() {
        this.emit('beforeClose');
        this._native.close();
        this.emit('closed');
    }
}

export default MysqlBinlog;
