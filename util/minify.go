package util

import (
	"bytes"
	"compress/gzip"
	"regexp"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/json"
	"github.com/tdewolff/minify/svg"
	"github.com/tdewolff/minify/xml"
)

// TODO, setup a minifier that can minify static files and rendered files. Then
// make it so that I can gzip static files and cache static files so that it
// doesn't have to be redone each time

var minifier *minify.M = nil

func get_minifier() *minify.M {
	if minifier == nil {
		minifier = minify.New()
		minifier.AddFunc("text/css", css.Minify)
		minifier.AddFunc("text/html", html.Minify)
		minifier.AddFunc("text/svg+xml", svg.Minify)
		minifier.AddFuncRegexp(
			regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"),
			js.Minify,
		)
		minifier.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
		minifier.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)
	}
	return minifier
}

func Minify(mediatype string, data []byte) ([]byte, error) {
	return get_minifier().Bytes(mediatype, data)
}

func CanMinify(mediatype string) bool {
	_, _, fn := get_minifier().Match(mediatype)
	return fn != nil
}

func Gzip(data []byte, compression int) ([]byte, error) {
	buf := BufPool.Get()
	defer BufPool.Put(buf)

	writer, err := gzip.NewWriterLevel(buf, compression)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}
	writer.Close()

	data = buf.Bytes()

	return data, nil
}

func GzipBestCompression(data []byte) ([]byte, error) {
	return Gzip(data, gzip.BestCompression)
}

func GzipBestSpeed(data []byte) ([]byte, error) {
	return Gzip(data, gzip.BestSpeed)
}

func GzipUncompress(data []byte) ([]byte, error) {
	buf := BufPool.Get()
	defer BufPool.Put(buf)

	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}

	reader.Close()

	data = buf.Bytes()
	return data, nil
}
