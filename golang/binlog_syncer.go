package main

import (
	"fmt"
	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	logger "github.com/siddontang/go-log/log"
	"log"
)

type MysqlBinlogPosition struct {
	Name     string `json:"name"`
	Position uint32 `json:"position"`
}

type MysqlBinlogRowUpdate struct {
	Old map[string]any `json:"old"`
	New map[string]any `json:"new"`
}

type MysqlBinlogTable struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

type MysqlBinlogChangeEvent struct {
	BinlogPosition MysqlBinlogPosition   `json:"binlogPosition"`
	Table          MysqlBinlogTable      `json:"table"`
	Insert         map[string]any        `json:"insert,omitempty"`
	Update         *MysqlBinlogRowUpdate `json:"update,omitempty"`
	Delete         map[string]any        `json:"delete,omitempty"`
}

type canalEventHandler struct {
	canal  *canal.Canal
	events chan<- MysqlBinlogChangeEvent
}

func (eh *canalEventHandler) OnRotate(_ *replication.RotateEvent) error {
	return nil
}

func (eh *canalEventHandler) OnTableChanged(_ string, _ string) error {
	return nil
}

func (eh *canalEventHandler) OnDDL(_ mysql.Position, _ *replication.QueryEvent) error {
	return nil
}

func (eh *canalEventHandler) OnRow(event *canal.RowsEvent) error {
	parseRow := func(row []any) map[string]any {
		parsedRow := make(map[string]any)
		for idx, column := range event.Table.Columns {
			parsedRow[column.Name] = row[idx]
		}
		return parsedRow
	}

	binlogPosition := eh.canal.SyncedPosition()
	changeEvent := MysqlBinlogChangeEvent{
		BinlogPosition: MysqlBinlogPosition{
			Name:     binlogPosition.Name,
			Position: binlogPosition.Pos,
		},
		Table: MysqlBinlogTable{
			Schema: event.Table.Schema,
			Name:   event.Table.Name,
		},
	}
	switch event.Action {
	case canal.InsertAction:
		changeEvent.Insert = parseRow(event.Rows[0])
	case canal.UpdateAction:
		changeEvent.Update = &MysqlBinlogRowUpdate{
			Old: parseRow(event.Rows[0]),
			New: parseRow(event.Rows[1]),
		}
	case canal.DeleteAction:
		changeEvent.Delete = parseRow(event.Rows[0])
	}
	eh.events <- changeEvent

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

type channelLogHandler struct {
	c chan<- string
}

func (h *channelLogHandler) Write(b []byte) (n int, err error) {
	h.c <- string(b)
	return len(b), nil
}

func (h *channelLogHandler) Close() error {
	return nil
}

type MysqlBinlogSyncer struct {
	canal         *canal.Canal
	binlogEvents  chan MysqlBinlogChangeEvent
	loggingEvents chan string
}

func NewSyncer(config MysqlBinlogConfig) (*MysqlBinlogSyncer, error) {
	loggingEvents := make(chan string, 100)

	canalCfg := canal.NewDefaultConfig()
	canalCfg.Addr = fmt.Sprintf("%s:%d", config.Hostname, config.Port)
	canalCfg.User = config.Username
	canalCfg.Password = config.Password
	canalCfg.Charset = "utf8mb4"
	canalCfg.IncludeTableRegex = config.TableRegexes
	canalCfg.ParseTime = true
	canalCfg.Dump = canal.DumpConfig{}
	canalCfg.Logger = logger.New(&channelLogHandler{c: loggingEvents}, logger.Llevel|logger.Lfile)

	c, err := canal.NewCanal(canalCfg)
	if err != nil {
		return nil, err
	}
	binlogEvents := make(chan MysqlBinlogChangeEvent)

	eventHandler := &canalEventHandler{
		canal:  c,
		events: binlogEvents,
	}
	c.SetEventHandler(eventHandler)

	var mysqlPosition mysql.Position
	if config.BinlogPosition == nil {
		masterPos, err := c.GetMasterPos()
		if err != nil {
			return nil, err
		}

		mysqlPosition = masterPos
	} else {
		mysqlPosition = *config.BinlogPosition
	}

	go func() {
		err := c.RunFrom(mysqlPosition)
		if err != nil {
			log.Printf("got error: %v", err)
		}
	}()

	return &MysqlBinlogSyncer{
		canal:         c,
		binlogEvents:  binlogEvents,
		loggingEvents: loggingEvents,
	}, nil
}

func (s *MysqlBinlogSyncer) ChangeEvents() <-chan MysqlBinlogChangeEvent {
	return s.binlogEvents
}

func (s *MysqlBinlogSyncer) LogEvents() <-chan string {
	return s.loggingEvents
}

func (s *MysqlBinlogSyncer) Close() {
	s.canal.Close()
	close(s.binlogEvents)
	close(s.loggingEvents)
}
