import { EventEmitter } from 'events';
import { debug } from 'debug';
import { ChildProcess, spawn } from 'child_process';
import * as readline from 'readline';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

const debugChannel = debug('mysql_binlog');

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

export interface BinlogEvent {
    binlogPosition: {
        name: string;
        position: number;
    }
    table: {
        schema: string;
        name: string;
    }
    insert?: Record<string, any>[];
    update?: {
        old: Record<string, any>;
        new: Record<string, any>;
    }[];
    delete?: Record<string, any>[];
}

export class ErrorWithBinLogPosition extends Error {
    public readonly binlogPosition: BinlogPosition;

    constructor(message: string, binlogPosition: BinlogPosition) {
        super(message);
        this.name = this.constructor.name;
        this.binlogPosition = binlogPosition;
    }
}

function _discoverGoBinary() {
    let suffix = '';
    if (os.platform() === 'win32') {
        suffix = '.exe';
    }
    const filename = path.join(__dirname, '..', 'prebuilds', `${os.platform()}-${os.arch()}${suffix}`);
    if (!fs.existsSync(filename)) {
        throw new Error(
            'Could not find pre-compiled Go binary. Either your platform is unsupported or you need to compile the binaries'
        );
    }
    return filename;
}

class MysqlBinlog extends EventEmitter {
    private _process: ChildProcess;
    private _readline: readline.ReadLine;

    private constructor(config: Config, process: ChildProcess) {
        super();
        this._process = process;
        this._readline = readline.createInterface({
            input: this._process.stdout!,
            crlfDelay: Infinity,
        })

        this._readline.on('line', (line) => {
            let msg;
            try {
                msg = JSON.parse(line);
                if (typeof msg !== 'object') {
                    throw new Error();
                }
            } catch (err) {
                debugChannel('received unexpected message on stdout: %s', line);
                this.emit('error', new Error('received unexpected message'));
                return;
            }

            switch (msg.type) {
                case 'connect_ok':
                    this.emit('_connect_ok');
                    break;
                case 'connect_error':
                    this.emit('_connect_err', new Error(msg.error));
                    break;
                case 'binlog_change':
                    this.emit('event', msg.event);
                    break;
                case 'log':
                    debugChannel(msg.message.trimEnd());
                    break;
                case 'error':
                    console.log(msg);
                    if (msg.binlogPosition) {
                        this.emit('error', new ErrorWithBinLogPosition(msg.error, msg.binlogPosition));
                    } else {
                        this.emit('error', new Error(msg.error));
                    }
                    break;
                default:
                    debugChannel('received unexpected message on stdout: %o', msg);
                    this.emit('error', new Error('received unexpected message'));
                    break
            }
        });
        this._process.stderr!.on('data', (chunk) => {
            debugChannel('received unexpected data on stderr: %s', chunk);
            this.emit('error', new Error('received unexpected data on stderr'));
        });
        this._process.on('close', () => {
            this.emit('close');
        });
        this._process.on('error', (err) => {
            debugChannel('received unexpected error: %s', err);
            this.emit('error', err);
        });

        this.send({
            type: 'connect',
            config,
        });
    }

    private send(message: any) {
        this._process.stdin!.write(JSON.stringify(message) + '\n');
    }

    public static create(config: Config): Promise<MysqlBinlog> {
        return new Promise((resolve, reject) => {
            const process = spawn(_discoverGoBinary(), {
                stdio: 'pipe',
            });
            process.on('error', (err) => {
                reject(err);
            });
            process.once('spawn', () => {
                process.removeAllListeners('error');
                const obj = new MysqlBinlog(config, process);
                obj.once('_connect_ok', () => {
                    resolve(obj);
                });
                obj.once('_connect_err', (err) => {
                    obj.close();
                    reject(err);
                });
            });
        });
    }

    public async close(): Promise<void> {
        if (this._process.exitCode !== null) {
            // do not try to kill the process multiple times
            return;
        }

        this.emit('beforeClose');
        this._readline.close();
        this._process.kill();
        return new Promise((resolve, reject) => {
            this._process.once('error', (err) => {
                reject(err);
            });
            this._process.once('close', () => {
                resolve(undefined);
            });
        });
    }

    public on(name: 'event', listener: (event: BinlogEvent) => void): this;
    public on(name: 'error', listener: (err: Error) => void): this;
    public on(name: 'close', listener: () => void): this;
    public on(name: 'beforeClose', listener: () => void): this;
    public on(eventName: string | symbol, listener: (...args: any[]) => void): this {
        return super.on(eventName, listener);
    }
}

export default MysqlBinlog;
