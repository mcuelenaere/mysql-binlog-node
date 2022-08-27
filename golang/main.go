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
type ErrorMessage struct {
	Type  string `json:"type"`
	Error string `json:"error"`
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
		log.Printf("could not send error message to stdout: %v", err)
	}
}

func sendError(err error) {
	sendMsg(ErrorMessage{
		Type:  "error",
		Error: err.Error(),
	})
}

func main() {
	log.SetOutput(os.Stderr)
	reader := bufio.NewReader(os.Stdin)

	messages := make(chan any)
	go func(c chan<- any) {
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
	}(messages)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)

	shouldLoop := true
	var binlogChanges <-chan MysqlBinlogChangeEvent
	var syncer *MysqlBinlogSyncer
	for shouldLoop {
		var err error
		select {
		case msg := <-messages:
			switch msg := msg.(type) {
			case ConnectMessage:
				syncer, err = NewSyncer(msg.Config)
				if err != nil {
					sendMsg(ConnectErrorMessage{
						Type:  "connect_error",
						Error: err.Error(),
					})
				} else {
					binlogChanges = syncer.ChangeEvents()
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
		case <-signals:
			err = os.Stdin.Close()
			if err != nil {
				sendError(err)
			}
			close(messages)
			if syncer != nil {
				syncer.Close()
				syncer = nil
			}
			shouldLoop = false
		}
	}
}
