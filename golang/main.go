package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/go-mysql-org/go-mysql/mysql"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type MysqlBinlogConfig struct {
	Hostname       string
	Port           uint16
	Username       string
	Password       string
	TableRegexes   []string
	BinlogPosition *mysql.Position
}

type UnknownMessage struct {
	Type string `json:"type"`
}
type ConnectMessage struct {
	Type   string            `json:"type"`
	Config MysqlBinlogConfig `json:"config"`
}
type ConnectErrorMessage struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}
type ConnectOkMessage struct {
	Type string `json:"type"`
}
type BinlogChangeMessage struct {
	Type  string                 `json:"type"`
	Event MysqlBinlogChangeEvent `json:"event"`
}
type LogMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
type ErrorMessage struct {
	Type           string               `json:"type"`
	Error          string               `json:"error"`
	BinlogPosition *MysqlBinlogPosition `json:"binlogPosition,omitempty"`
}

var sendMutex sync.Mutex

func sendMsg(msg any) {
	sendMutex.Lock()
	defer sendMutex.Unlock()

	serializedMsg, err := json.Marshal(msg)
	if err != nil {
		log.Printf("could not serialize error message: %v", err)
		return
	}
	serializedMsg = append(serializedMsg, '\n')

	_, err = os.Stdout.Write(serializedMsg)
	if err != nil {
		log.Fatalf("could not send error message to stdout: %v", err)
	}
}

func sendError(err error) {
	if errWithBinLog, ok := err.(ErrorWithBinlogPosition); ok {
		sendMsg(ErrorMessage{
			Type:           "error",
			Error:          err.Error(),
			BinlogPosition: &errWithBinLog.BinlogPosition,
		})
	} else {
		sendMsg(ErrorMessage{
			Type:  "error",
			Error: err.Error(),
		})
	}
}

func parseMessages(c chan<- any) {
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Printf("received unexpected err when reading from stdin: %v", err)
			break
		}

		var unkMsg UnknownMessage
		err = json.Unmarshal([]byte(line), &unkMsg)
		if err != nil {
			sendError(err)
			continue
		}
		switch unkMsg.Type {
		case "connect":
			var connectMsg ConnectMessage
			err = json.Unmarshal([]byte(line), &connectMsg)
			if err != nil {
				sendError(err)
				continue
			}

			c <- connectMsg
		default:
			sendError(errors.New("unknown message type"))
		}
	}
}

func main() {
	log.SetOutput(os.Stderr)

	messages := make(chan any)
	go parseMessages(messages)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)

	shouldLoop := true
	connectTimer := time.NewTimer(10 * time.Second)
	var binlogChanges <-chan MysqlBinlogChangeEvent
	var binlogErrors <-chan error
	var logEvents <-chan string
	var syncer *MysqlBinlogSyncer
	for shouldLoop {
		select {
		case msg := <-messages:
			switch msg := msg.(type) {
			case ConnectMessage:
				connectTimer.Stop()

				if syncer != nil {
					syncer.Close()
					syncer = nil
				}

				var err error
				syncer, err = NewSyncer(msg.Config)
				if err != nil {
					sendMsg(ConnectErrorMessage{
						Type:  "connect_error",
						Error: err.Error(),
					})
				} else {
					binlogChanges = syncer.ChangeEvents()
					binlogErrors = syncer.Errors()
					logEvents = syncer.LogEvents()
					sendMsg(ConnectOkMessage{
						Type: "connect_ok",
					})
				}
			}
		case event := <-binlogChanges:
			sendMsg(BinlogChangeMessage{
				Type:  "binlog_change",
				Event: event,
			})
		case err := <-binlogErrors:
			sendError(err)
		case event := <-logEvents:
			sendMsg(LogMessage{
				Type:    "log",
				Message: event,
			})
		case <-connectTimer.C:
			if syncer == nil {
				sendError(errors.New("timeout was hit while waiting for connect message"))
				shouldLoop = false
			}
		case <-signals:
			shouldLoop = false
		}
	}

	err := os.Stdin.Close()
	if err != nil {
		sendError(err)
	}
	close(messages)
	if syncer != nil {
		syncer.Close()
		syncer = nil
	}
}
