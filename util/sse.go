package util

import (
	"bytes"
	"net/http"

	"github.com/sirupsen/logrus"
)

type Sse struct {
	w       http.ResponseWriter
	r       *http.Request
	ch      chan []byte
	on_done func()
}

func (self *Sse) send_message(msg []byte) error {
	i := 0
	for i < len(msg) {
		buf := msg[i:]
		newline := bytes.Index(buf, []byte("\n"))
		if newline != -1 {
			buf = buf[:newline]
			i += newline + 1
		} else {
			i += len(buf)
		}
		_, err := self.w.Write([]byte("data:"))
		if err != nil {
			return err
		}
		_, err = self.w.Write(buf)
		if err != nil {
			return err
		}
		_, err = self.w.Write([]byte("\n"))
		if err != nil {
			return err
		}
	}
	_, err := self.w.Write([]byte("\n"))
	if f, ok := self.w.(http.Flusher); ok {
		f.Flush()
	}
	return err
}

func (self *Sse) runner() {
	for {
		select {
		case msg, exists := <-self.ch:
			if !exists {
				self.on_done()
				return
			}
			err := self.send_message(msg)
			if err != nil {
				logrus.Error(err)
				continue
			}
		case <-self.r.Context().Done():
			self.on_done()
			return
		}
	}
}

func (self *Sse) Send(data []byte) {
	self.ch <- data
}

func RunSse(w http.ResponseWriter, r *http.Request, ch chan []byte, on_done func()) {
	sse := &Sse{
		w:       w,
		r:       r,
		ch:      ch,
		on_done: on_done,
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)

	sse.runner()
}
