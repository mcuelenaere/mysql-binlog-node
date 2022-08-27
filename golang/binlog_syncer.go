package main

import (
	"fmt"
	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/go-mysql-org/go-mysql/schema"
	logger "github.com/siddontang/go-log/log"
	"log"
)

type MysqlBinlogChangeEvent struct {
	BinlogPosition mysql.Position
	Table          *schema.Table
	Action         string
	Rows           [][]interface{}
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
	eh.events <- MysqlBinlogChangeEvent{
		BinlogPosition: eh.canal.SyncedPosition(),
		Table:          event.Table,
		Action:         event.Action,
		Rows:           event.Rows,
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

type MysqlBinlogSyncer struct {
	canal  *canal.Canal
	events chan MysqlBinlogChangeEvent
}

func NewSyncer(config MysqlBinlogConfig) (*MysqlBinlogSyncer, error) {
	canalCfg := canal.NewDefaultConfig()
	canalCfg.Addr = fmt.Sprintf("%s:%d", config.Hostname, config.Port)
	canalCfg.User = config.Username
	canalCfg.Password = config.Password
	canalCfg.Charset = "utf8mb4"
	canalCfg.IncludeTableRegex = config.TableRegexes
	canalCfg.ParseTime = true
	canalCfg.Dump = canal.DumpConfig{}
	logHandler, _ := logger.NewNullHandler()
	canalCfg.Logger = logger.NewDefault(logHandler)

	c, err := canal.NewCanal(canalCfg)
	if err != nil {
		return nil, err
	}
	events := make(chan MysqlBinlogChangeEvent)

	eventHandler := &canalEventHandler{
		canal:  c,
		events: events,
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

	return &MysqlBinlogSyncer{canal: c, events: events}, nil
}

func (s *MysqlBinlogSyncer) ChangeEvents() <-chan MysqlBinlogChangeEvent {
	return s.events
}

func (s *MysqlBinlogSyncer) Close() {
	s.canal.Close()
	close(s.events)
}
