package web

import (
	"database/sql"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/ttocsneb/station-webapp/station"
	"github.com/ttocsneb/station-webapp/util"
)

type embedFunc func(w io.Writer, arg []any) error

var embedFuncs = map[string]embedFunc{}

func Main(db *sql.DB, client *station.Station) {
	router := mux.NewRouter()

	err := loadTemplates()
	if err != nil {
		panic(err)
	}

	router.PathPrefix("/static/").Handler(http.HandlerFunc(serveStatic))
	router.HandleFunc("/", serveMain(db))
	router.HandleFunc("/rapid/", serveRapid(db))
	router.Handle("/sse/updates/", serveUpdates(db, client))
	router.Handle("/sse/rapid-updates/", serveRapidUpdates(db, client))
	router.HandleFunc("/system/", serveSystemForm)
	router.HandleFunc("/dynamic/wind.svg", serveWind)
	embedFuncs["wind.svg"] = embedWind

	log.Infof("Listening on %v", util.Conf.Listen)
	err = http.ListenAndServe(util.Conf.Listen, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			sentHeader:     false,
			request:        r,
		}
		router.ServeHTTP(wrapper, r)
	}))
	if err != nil {
		panic(err)
	}
}

func logError(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	logrus.Error(err)
}

type responseWriterWrapper struct {
	http.ResponseWriter
	sentHeader bool
	request    *http.Request
}

func (w *responseWriterWrapper) WriteHeader(status int) {
	if !w.sentHeader {
		log.Infof("%v\t[%v]\t%v", w.request.Method, status, w.request.URL.Path)
	}
	w.sentHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.sentHeader {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriterWrapper) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}
