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

	tc := TermContext{}

	e.File("/", "static/index.html")
	e.Static("/static", "static/")

	e.GET("/websocket", tc.wsHandler)

	e.Logger.Fatal(e.Start("127.0.0.1:8080"))
}

func (tc *TermContext) wsHandler(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		var err error
		tc.pty, err = pty.Start(exec.Command("bash"))
		if err != nil {
			panic(err)
		}

		// onOpen
		buf := make([]byte, 1024)
		size, err := tc.pty.Read(buf)
		out, _ := json.Marshal([]string{"stdout", string(buf[:size])})
		fmt.Println(string(out))
		err = websocket.Message.Send(ws, string(out))
		if err != nil {
			c.Logger().Error(err)
		}

		for {
			var err error

			// Read from ws
			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err)
			}

			fmt.Println("****************************")

			var dat []string
			if err := json.Unmarshal([]byte(msg), &dat); err != nil {
				c.Logger().Error(err)
				return
			}
			msgType := dat[0]
			command := dat[1]
			fmt.Printf("%s %q\n", msgType, command)

			switch msgType {
			case "stdin":
				done := make(chan bool)

				go func() {
					// Write to pty
					tc.pty.Write([]byte(command))
					done <- true
				}()
				go func() {
					buf := make([]byte, 1024)

					for {
						size, err = tc.pty.Read(buf)

						// Write back to ws
						out, _ := json.Marshal([]string{"stdout", string(buf[:size])})
						fmt.Println(string(out))
						err = websocket.Message.Send(ws, string(out))
						if err != nil {
							c.Logger().Error(err)
						}
					}
				}()

				<-done
			case "set_size":
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}
