package web

import (
	"database/sql"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/ttocsneb/station-webapp/database"
	"github.com/ttocsneb/station-webapp/station"
	"github.com/ttocsneb/station-webapp/util"
)

type updateRenderer struct {
	db          *sql.DB
	client      *station.Station
	subscribe   func() chan database.Condition
	unsubscribe func(chan database.Condition)
	updates     chan []byte
	done        chan any
	args        map[string]any
}

func (self *updateRenderer) start() {
	updates := self.subscribe()

	go func() {
		if _, exists := self.args["Rapid"]; exists {
			logrus.Infof("Starting %v rapid updates", self.args["System"])
		} else {
			logrus.Infof("Starting %v updates", self.args["System"])
		}
		for {
			select {
			case update := <-updates:
				self.args["Condition"] = update

				buf := util.BufPool.Get()
				err := renderTemplate(buf, "update-partial.html", self.args)
				if err != nil {
					util.BufPool.Put(buf)
					continue
				}
				self.updates <- buf.Bytes()
				util.BufPool.Put(buf)
			case <-self.done:
				self.unsubscribe(updates)
				if _, exists := self.args["Rapid"]; exists {
					logrus.Infof("Stopping %v rapid updates", self.args["System"])
				} else {
					logrus.Infof("Stopping %v updates", self.args["System"])
				}
				return
			}
		}
	}()

}
func (self *updateRenderer) stop() {
	self.done <- true
}

func serveUpdates(db *sql.DB, client *station.Station) http.Handler {
	var muxes = make(map[string]*util.ChanMux[[]byte])

	create_mux := func(system string) *util.ChanMux[[]byte] {
		updator := &updateRenderer{
			db:          db,
			client:      client,
			subscribe:   client.SubscribeUpdates,
			unsubscribe: client.UnsubscribeUpdates,
			updates:     make(chan []byte),
			done:        make(chan any),
			args:        map[string]any{"System": system},
		}
		mux := util.NewChanMux(updator.updates)
		mux.OnSubscribe = updator.start
		mux.OnEmpty = updator.stop
		return mux
	}

	muxes[METRIC] = create_mux(METRIC)
	muxes[IMPERIAL] = create_mux(IMPERIAL)
	muxes[MIXED] = create_mux(MIXED)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("system")
		system := METRIC
		if err == nil {
			system = cookie.Value
		}

		mux, exists := muxes[system]
		if !exists {
			w.WriteHeader(400)
			return
		}

		condition, err := database.FetchLatestCondition(db)
		if err != nil {
			logrus.Error(err)
			w.WriteHeader(500)
			return
		}

		args := map[string]any{
			"Condition": condition,
			"System":    system,
		}

		buf := util.BufPool.Get()
		err = renderTemplate(buf, "update-partial.html", args)
		if err != nil {
			logrus.Error(err)
			w.WriteHeader(500)
			util.BufPool.Put(buf)
			return
		}
		msg := buf.Bytes()
		util.BufPool.Put(buf)

		data := mux.Subscribe(2)
		data <- msg

		util.RunSse(w, r, data, func() {
			mux.Unsubscribe(data)
		})
	})
}

func serveRapidUpdates(db *sql.DB, client *station.Station) http.Handler {
	var muxes = make(map[string]*util.ChanMux[[]byte])

	create_mux := func(system string) *util.ChanMux[[]byte] {
		updator := &updateRenderer{
			db:          db,
			client:      client,
			subscribe:   client.SubscribeRapid,
			unsubscribe: client.UnsubscribeRapid,
			updates:     make(chan []byte),
			done:        make(chan any),
			args: map[string]any{
				"System": system,
				"Rapid":  true,
			},
		}
		mux := util.NewChanMux(updator.updates)
		mux.OnSubscribe = updator.start
		mux.OnEmpty = updator.stop
		return mux
	}

	muxes[METRIC] = create_mux(METRIC)
	muxes[IMPERIAL] = create_mux(IMPERIAL)
	muxes[MIXED] = create_mux(MIXED)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("system")
		system := METRIC
		if err == nil {
			system = cookie.Value
		}

		mux, exists := muxes[system]
		if !exists {
			w.WriteHeader(400)
			return
		}

		condition, err := database.FetchLatestCondition(db)
		if err != nil {
			logrus.Error(err)
			w.WriteHeader(500)
			return
		}

		args := map[string]any{
			"Condition": condition,
			"System":    system,
		}

		buf := util.BufPool.Get()
		err = renderTemplate(buf, "update-partial.html", args)
		if err != nil {
			logrus.Error(err)
			w.WriteHeader(500)
			util.BufPool.Put(buf)
			return
		}
		msg := buf.Bytes()
		util.BufPool.Put(buf)

		data := mux.Subscribe(2)
		data <- msg

		util.RunSse(w, r, data, func() {
			mux.Unsubscribe(data)
		})
	})
}
