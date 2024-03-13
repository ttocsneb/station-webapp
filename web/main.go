package web

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ttocsneb/station-webapp/database"
)

func serveMain(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		condition, err := database.FetchLatestCondition(db)
		if err != nil {
			logError(w, err)
			return
		}

		cookie, err := r.Cookie("system")
		system := METRIC
		if err == nil {
			system = cookie.Value
		}

		logrus.Info(system)

		err = renderTemplate(w, "main.html", vars{
			"Condition": condition,
			"System":    system,
			"Page":      r.URL.Path,
		})

		if err != nil {
			logError(w, err)
			w.Write([]byte("<p>Invalid template</p>"))
			return
		}
	}
}

func serveRapid(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		condition, err := database.FetchLatestCondition(db)
		if err != nil {
			logError(w, err)
			return
		}

		cookie, err := r.Cookie("system")
		system := METRIC
		if err == nil {
			system = cookie.Value
		}

		logrus.Info(system)

		err = renderTemplate(w, "main.html", vars{
			"Condition": condition,
			"System":    system,
			"Page":      r.URL.Path,
			"Rapid":     true,
		})

		if err != nil {
			logError(w, err)
			w.Write([]byte("<p>Invalid template</p>"))
			return
		}
	}
}

func serveSystemForm(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(400)
		return
	}

	system := r.Form.Get("system")
	if system != METRIC && system != IMPERIAL && system != MIXED {
		w.WriteHeader(400)
		return
	}

	expires := time.Now().UTC().Add(time.Hour * 24 * 365)

	w.Header().Set("Set-Cookie", fmt.Sprintf(
		"system=%v; Expires=%v; Path=/",
		system, expires.Format(time.RFC1123),
	))
	next := r.Form.Get("next")
	if next == "" {
		next = route("/")
	}
	next, err = url.PathUnescape(next)
	if err != nil {
		logrus.Error(err)
		http.Error(w, "Invalid Request", 400)
		return
	}
	http.Redirect(w, r, next, 302)
}

func embedWind(w io.Writer, args []any) error {
	rot, exists := args[0].(int)
	if !exists {
		f, exists := args[0].(float64)
		if !exists {
			return errors.New("angle must be an int")
		}
		rot = int(f)
	}

	var id string = "wind"
	if len(args) > 1 {
		id, exists = args[1].(string)
		if !exists {
			return errors.New("id must be a string")
		}
	}

	err := renderTemplate(w, "wind.svg", vars{
		"Angle": rot,
		// "Text":  text,
		"Id": id,
	})
	return err
}

func serveWind(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(400)
		return
	}

	angle := r.Form.Get("angle")
	if angle == "" {
		angle = "0"
	}

	rotation, err := strconv.ParseFloat(angle, 64)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	err = embedWind(w, []any{int(rotation)})
	if err != nil {
		w.WriteHeader(500)
		logrus.Error(err)
	}
}
