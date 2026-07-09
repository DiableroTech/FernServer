// Temporary smoke-test client; delete after verification.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func main() {
	token := os.Args[1]
	url := "ws://localhost:8080/api/v1/chat/ws?token=" + token
	if len(os.Args) > 2 {
		url = "ws://localhost:" + os.Args[2] + "/api/v1/chat/ws?token=" + token
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		fmt.Println("DIAL_FAIL:", err)
		os.Exit(1)
	}
	defer conn.CloseNow()

	send := func(m map[string]string) {
		if err := wsjson.Write(ctx, conn, m); err != nil {
			fmt.Println("WRITE_FAIL:", err)
			os.Exit(1)
		}
	}
	waitFor := func(terminal ...string) map[string]any {
		for {
			var msg map[string]any
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				fmt.Println("READ_END:", err)
				os.Exit(1)
			}
			t, _ := msg["type"].(string)
			if t == "delta" {
				fmt.Print(msg["text"])
				continue
			}
			fmt.Printf("\n[%s] %v\n", t, msg)
			for _, want := range terminal {
				if t == want {
					return msg
				}
			}
		}
	}

	if len(os.Args) > 3 && os.Args[3] == "memcheck" {
		fmt.Println("=== memory check (no wrap-up, nothing saved) ===")
		send(map[string]string{"type": "message", "text": "Just checking in quickly — do you remember what we've been working on together recently?", "modality": "freeform"})
		waitFor("done", "error")
		return
	}

	fmt.Println("=== turn 1 ===")
	send(map[string]string{"type": "message", "text": "I've been stressed about work deadlines lately and it's affecting my sleep.", "modality": "cbt"})
	waitFor("done", "error")

	fmt.Println("=== turn 2 ===")
	send(map[string]string{"type": "message", "text": "I keep thinking if I miss a deadline everyone will decide I'm useless.", "modality": "cbt"})
	waitFor("done", "error")

	fmt.Println("=== wrap up ===")
	send(map[string]string{"type": "wrap_up"})
	msg := waitFor("summary", "error")
	if msg["type"] == "summary" {
		fmt.Println("WRAP_UP_OK session:", msg["sessionId"])
	}
}
