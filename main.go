package main

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/net/websocket"

	"os/exec"

	"github.com/kr/pty"
	"github.com/labstack/echo"
)

type TermContext struct {
	pty *os.File
}

func main() {
	e := echo.New()

	f, err := pty.Start(exec.Command("bash"))
	if err != nil {
		panic(err)
	}
	tc := TermContext{pty: f}

	e.File("/", "index.html")

	e.GET("/websocket", tc.wsHandler)

	e.Logger.Fatal(e.Start("127.0.0.1:8080"))
}

func (tc *TermContext) wsHandler(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			var err error

			// Read from ws
			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err)
			}

			var dat []string
			if err := json.Unmarshal([]byte(msg), &dat); err != nil {
				panic(err)
			}
			msgType := dat[0]
			command := dat[1]
			fmt.Println(msgType, command)

			switch msgType {
			case "stdin":
				done := make(chan bool, 2)

				go func() {
					// Write to pty
					tc.pty.Write([]byte(command))
					done <- true
				}()
				go func() {
					buf := make([]byte, 1024)
					//io.Copy(os.Stdout, tc.pty)

					_, err := tc.pty.Read(buf)
					// Write back to ws
					err = websocket.Message.Send(ws, string(buf))
					if err != nil {
						c.Logger().Error(err)
					}
					done <- true
				}()

				<-done
				<-done
			case "set_size":
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}
