package main

/*
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef void* BinlogUserData;
typedef void (*BinlogEventCallback)(BinlogUserData user_data, const char* json);

static inline void invoke_event_callback(BinlogEventCallback cb, BinlogUserData user_data, const char* json) {
    cb(user_data, json);
}

typedef void (*BinlogLogCallback)(const char* message);

static inline void invoke_log_callback(BinlogLogCallback cb, const char* message) {
    cb(message);
}

typedef struct BinlogPosition {
    const char*             name;
    uint32_t                position;
} BinlogPosition;

typedef struct BinlogConfig {
    const char*             hostname;
    int16_t                 port;
    const char*             username;
    const char*             password;
    const char**            table_regexes;
    size_t                  table_regexes_count;
    BinlogPosition*         binlog_position;
    BinlogEventCallback     callback;
    BinlogUserData          user_data;
} BinlogConfig;

#ifdef __cplusplus
}
#endif
*/
import "C"

import (
    "encoding/json"
    "fmt"
    "github.com/go-mysql-org/go-mysql/canal"
    "github.com/go-mysql-org/go-mysql/mysql"
    "github.com/go-mysql-org/go-mysql/replication"
    "github.com/go-mysql-org/go-mysql/schema"
    "log"
    "runtime/cgo"
    "unsafe"
)

//export Binlog_SetLogger
func Binlog_SetLogger(callback C.BinlogLogCallback) {
    // TODO
}

type canalEventHandler struct {
    canal    *canal.Canal
    callback C.BinlogEventCallback
	userData C.BinlogUserData
}

func (eh *canalEventHandler) OnRotate(_ *replication.RotateEvent) error {
	return nil
}

func (eh *canalEventHandler) OnTableChanged(schema string, table string) error {
	log.Printf("OnTableChanged(%s, %s)", schema, table)
	return nil
}

func (eh *canalEventHandler) OnDDL(nextPos mysql.Position, queryEvent *replication.QueryEvent) error {
	log.Printf("OnDDL(%v, %v)", nextPos, queryEvent)
	return nil
}

func (eh *canalEventHandler) OnRow(event *canal.RowsEvent) error {
	log.Printf("OnRow(%v)", event)

    binlogEvent := struct {
        BinlogPosition mysql.Position
        Table *schema.Table
        Rows [][]any
    } {
        BinlogPosition: eh.canal.SyncedPosition(),
        Table: event.Table,
        Rows: event.Rows,
    }
    jsonEvent, err := json.Marshal(binlogEvent)
    if err == nil {
        C.invoke_event_callback(eh.callback, eh.userData, C.CString(string(jsonEvent)))
    }

	return nil
}

func (eh *canalEventHandler) OnXID(_ mysql.Position) error {
	return nil
}

func (eh *canalEventHandler) OnGTID(_ mysql.GTIDSet) error {
	return nil
}

func (eh *canalEventHandler) OnPosSynced(_ mysql.Position, _ mysql.GTIDSet, _ bool) error {
	return nil
}

func (eh *canalEventHandler) String() string {
	return "canalEventHandler"
}

type MysqlBinlogChangeEvent struct {
	BinlogPosition mysql.Position
	Table          *schema.Table
	Action         string
	Rows           [][]interface{}
}

type MysqlBinlogSyncer struct {
	canal *canal.Canal
}

//export Binlog_New
func Binlog_New(config C.BinlogConfig, error_out **C.char) C.uintptr_t {
    tableRegexes := unsafe.Slice(config.table_regexes, config.table_regexes_count)
    includeTableRegex := make([]string, 0, len(tableRegexes))
    for _, tableRegex := range tableRegexes {
        includeTableRegex = append(includeTableRegex, C.GoString(tableRegex))
    }

	canalCfg := canal.NewDefaultConfig()
	canalCfg.Addr = fmt.Sprintf("%s:%d", C.GoString(config.hostname), int(config.port))
	canalCfg.User = C.GoString(config.username)
	canalCfg.Password = C.GoString(config.password)
	canalCfg.Charset = "utf8mb4"
	canalCfg.IncludeTableRegex = includeTableRegex
	canalCfg.ParseTime = true
	canalCfg.Dump = canal.DumpConfig{}

	c, err := canal.NewCanal(canalCfg)
	if err != nil {
	    if error_out != nil {
	        *error_out = C.CString(err.Error());
	    }
		return C.uintptr_t(0)
	}

	eventHandler := &canalEventHandler{
	    canal:    c,
	    callback: config.callback,
	    userData: config.user_data,
	}
	c.SetEventHandler(eventHandler)

    var mysqlPosition mysql.Position
    if config.binlog_position == nil {
        masterPos, err := c.GetMasterPos()
        if err != nil {
            if error_out != nil {
                *error_out = C.CString(err.Error());
            }
            return C.uintptr_t(0)
        }

        mysqlPosition = masterPos
    } else {
        mysqlPosition = mysql.Position{
            Name: C.GoString(config.binlog_position.name),
            Pos: uint32(config.binlog_position.position),
        }
    }

    go func() {
        err := c.RunFrom(mysqlPosition)
        if err != nil {
            log.Printf("got error: %v", err)
        }
    }()

    syncer := &MysqlBinlogSyncer{
        canal: c,
    }
	return C.uintptr_t(cgo.NewHandle(syncer))
}

//export Binlog_Close
func Binlog_Close(handle C.uintptr_t) {
    s := cgo.Handle(handle).Value().(*MysqlBinlogSyncer)

	s.canal.Close()
}

//export Binlog_Free
func Binlog_Free(handle C.uintptr_t) {
    cgo.Handle(handle).Delete()
}

func main() {
    // no-op
}