// This file is copied from https://github.com/hashrocket/ws

// package ws provides business logic for websocket-cli
package ws

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

type session struct {
	ws      *websocket.Conn
	rl      *readline.Instance
	errChan chan error
}

func Connect(originUrl, origin, auth string, rlConf *readline.Config) error {
	headers := make(http.Header)
	headers.Add("Origin", origin)
	if auth[7:] != "" {
		headers.Add("Authorization", auth)
	}

	ws, _, err := websocket.DefaultDialer.Dial(originUrl, headers)
	if err != nil {
		return err
	}

	rl, err := readline.NewEx(rlConf)
	if err != nil {
		return err
	}
	defer rl.Close()

	sess := &session{
		ws:      ws,
		rl:      rl,
		errChan: make(chan error),
	}

	go sess.readConsole()
	go sess.readWebsocket()

	return <-sess.errChan
}

func (s *session) readConsole() {
	for {
		line, err := s.rl.Readline()
		if err != nil {
			s.errChan <- err
			return
		}

		err = s.ws.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			s.errChan <- err
			return
		}
	}
}

func bytesToFormattedHex(bytes []byte) string {
	text := hex.EncodeToString(bytes)
	return regexp.MustCompile("(..)").ReplaceAllString(text, "$1 ")
}

func (s *session) readWebsocket() {
	rxSprintf := color.New(color.FgGreen).SprintfFunc()

	for {
		msgType, buf, err := s.ws.ReadMessage()
		if err != nil {
			s.errChan <- err
			return
		}

		var text string
		switch msgType {
		case websocket.TextMessage:
			text = string(buf)
		case websocket.BinaryMessage:
			text = bytesToFormattedHex(buf)
		case websocket.CloseMessage:
			s.errChan <-fmt.Errorf("websocket is closed")
		default:
			s.errChan <- fmt.Errorf("unknown websocket frame type: %d", msgType)
			return
		}

		fmt.Fprint(s.rl.Stdout(), rxSprintf("< %s\n", text))
	}
}
