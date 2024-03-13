package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/ttocsneb/station-webapp/util"
)

//go:embed templates/*
var templFiles embed.FS

func readDirRecursive(fsys fs.FS, name string) ([]string, error) {
	ents, err := fs.ReadDir(fsys, name)
	if err != nil {
		return nil, err
	}

	result := []string{}

	for _, ent := range ents {
		ent_name := ent.Name()
		if ent.Type().IsDir() {
			n := fmt.Sprintf("%v/%v", name, ent_name)
			files, err := readDirRecursive(fsys, n)
			if err == fs.ErrPermission {
				continue
			}
			if err != nil {
				return nil, err
			}
			for _, f := range files {
				result = append(result, f)
			}
		} else {
			result = append(result, fmt.Sprintf("%v/%v", name, ent_name))
		}
	}
	return result, nil
}

var templs map[string]*template.Template

type vars map[string]any

func loadTemplates() error {
	layouts, err := readDirRecursive(templFiles, "templates/layouts")
	if err != nil {
		return err
	}
	includes, err := readDirRecursive(templFiles, "templates/includes")
	if err != nil {
		return err
	}

	templs = make(map[string]*template.Template)

	for _, layout := range layouts {
		name := layout[strings.LastIndexByte(layout, '/')+1:]
		files := append(includes, layout)
		templ, err := template.New("").Funcs(funcs).ParseFS(templFiles, files...)
		if err != nil {
			return err
		}
		templs[name] = templ
	}

	include_templ, err := template.New("").Funcs(funcs).ParseFS(templFiles, includes...)
	if err != nil {
		return err
	}
	for _, include := range includes {
		name := include[strings.LastIndexByte(include, '/')+1:]
		templs[name] = include_templ
	}

	return nil
}

func renderTemplate(response io.Writer, name string, vars any) error {
	var err error
	if len(templs) == 0 {
		err = loadTemplates()
		if err != nil {
			return err
		}
	}
	templ, exists := templs[name]
	if !exists {
		return fmt.Errorf("Template %v does not exist", name)
	}
	buf := util.BufPool.Get()
	defer util.BufPool.Put(buf)

	err = templ.ExecuteTemplate(buf, name, vars)
	if err != nil {
		return err
	}

	data := buf.Bytes()

	m := mimetype.Detect(data[:min(512, len(data))])
	mt := m.String()
	if mt == "application/octet-stream" || strings.HasPrefix(mt, "text/plain") {
		index := strings.LastIndex(name, ".")
		if index != -1 {
			m := mime.TypeByExtension(name[index:])
			if m != "" {
				mt = m
			}
		}
	}

	if util.CanMinify(mt) {
		data, err = util.Minify(mt, data)
		if err != nil {
			return err
		}
	}
	response.Write(data)

	return nil
}
