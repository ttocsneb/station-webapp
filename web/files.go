package web

import (
	"embed"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	log "github.com/sirupsen/logrus"
	"github.com/ttocsneb/station-webapp/util"
)

//go:embed static/*
var staticFiles embed.FS

var modTime time.Time = time.Time{}

func lastModTime() time.Time {
	none := time.Time{}
	if modTime == none {
		exec, err := os.Executable()
		if err != nil {
			log.Error(err)
			modTime = time.Now().UTC().Truncate(time.Second)
			return modTime
		}
		stat, err := os.Stat(exec)
		if err != nil {
			log.Error(err)
			modTime = time.Now().UTC().Truncate(time.Second)
			return modTime
		}
		modTime = stat.ModTime().UTC().Truncate(time.Second)
	}
	return modTime
}

var mimes map[string]string = make(map[string]string)

type staticKey struct {
	Name string
	Gzip bool
}

type cachedFile struct {
	Data        []byte
	contentType string
	modTime     time.Time
}

var static_cache = map[staticKey]cachedFile{}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		w.Write([]byte("<p>405 Method Not Allowed</p>"))
		return
	}

	notfound := func() {
		w.WriteHeader(404)
		w.Write([]byte("<p>400 file not found</p>"))
	}

	f, err := staticFiles.Open(r.URL.Path[1:])
	if err != nil {
		log.Error(err)
		notfound()
		return
	}

	encoding := r.Header.Get("Accept-Encoding")
	is_gzip := false
	for _, enc := range strings.Split(encoding, ",") {
		if strings.ToLower(strings.TrimSpace(enc)) == "gzip" {
			is_gzip = true
			break
		}
	}

	cache_key := staticKey{
		Name: r.URL.Path[1:],
		Gzip: is_gzip,
	}
	if cache, exists := static_cache[cache_key]; exists {
		w.Header().Set("Last-Modified", cache.modTime.Format(time.RFC1123))
		w.Header().Set("Content-Type", cache.contentType)

		if since := r.Header.Get("If-Modified-Since"); since != "" {
			since, err := time.Parse(time.RFC1123, since)
			if err != nil {
				log.Error(err)
				http.Error(w, "Invalid Header", 400)
				return
			}
			if since.Sub(modTime).Seconds() >= 0 {
				w.WriteHeader(304)
				return
			}
		}

		if is_gzip {
			w.Header().Set("Content-Encoding", "gzip")
		}

		w.WriteHeader(200)
		w.Write(cache.Data)
		return
	}

	cache := cachedFile{}

	cache.modTime = lastModTime()

	w.Header().Set("Last-Modified", cache.modTime.Format(time.RFC1123))

	buf := [512]byte{}
	read, err := f.Read(buf[:])
	if err != nil {
		log.Error(err)
		notfound()
		return
	}
	mt, exists := mimes[r.URL.Path]
	if !exists {
		m := mimetype.Detect(buf[:read])
		mt = m.String()
		if mt == "application/octet-stream" || strings.HasPrefix(mt, "text/plain") {
			index := strings.LastIndex(r.URL.Path, ".")
			if index != -1 {
				m := mime.TypeByExtension(r.URL.Path[index:])
				if m != "" {
					mt = m
				}
			}
		}
		mimes[r.URL.Path] = mt
	}
	cache.contentType = mt
	w.Header().Set("Content-Type", mt)

	if since := r.Header.Get("If-Modified-Since"); since != "" {
		since, err := time.Parse(time.RFC1123, since)
		if err != nil {
			log.Error(err)
			w.WriteHeader(400)
			w.Write([]byte("<p>400 Invalid header</p>"))
			return
		}
		if since.Sub(modTime).Seconds() >= 0 {
			w.WriteHeader(304)
			return
		}
	}

	buffer := util.BufPool.Get()
	defer util.BufPool.Put(buffer)
	buffer.Write(buf[:])
	_, err = buffer.ReadFrom(f)
	if err != nil {
		log.Error(err)
		http.Error(w, "Could not read file", 500)
		return
	}

	data := buffer.Bytes()

	if util.CanMinify(cache.contentType) {
		data, err = util.Minify(cache.contentType, data)
		if err != nil {
			log.Errorf("Minify Error: %v", err)
			http.Error(w, "Could not read file", 500)
			return
		}
	}

	if is_gzip {
		data, err = util.GzipBestCompression(data)
		if err != nil {
			log.Errorf("Gzip Error: %v", err)
			http.Error(w, "Could not read file", 500)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
	}

	new_data := make([]byte, len(data))
	for i, c := range data {
		new_data[i] = c
	}
	cache.Data = new_data
	static_cache[cache_key] = cache

	w.Write(data)
}
