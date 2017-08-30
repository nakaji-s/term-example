package main

import (
	"encoding/json"
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
		go func() {
			buf := make([]byte, 1024)

			for {
				// Read from pty
				size, err := tc.pty.Read(buf)
				if err != nil {
					c.Logger().Error(err)
				}

				// Write back to ws
				out, err := json.Marshal([]string{"stdout", string(buf[:size])})
				if err != nil {
					c.Logger().Error(err)
				}
				err = websocket.Message.Send(ws, string(out))
				if err != nil {
					c.Logger().Error(err)
				}
			}
		}()

		for {
			// Read from ws
			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err)
			}

			var dat []string
			if err := json.Unmarshal([]byte(msg), &dat); err != nil {
				c.Logger().Error(err)
				return
			}
			msgType := dat[0]
			command := dat[1]

			switch msgType {
			case "stdin":
				// Write to pty
				tc.pty.Write([]byte(command))
			case "set_size":
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}
