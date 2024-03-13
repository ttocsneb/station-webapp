sass := web/style
style := web/static/style.css

sass_files = $(shell find $(sass) -type f -name '*.scss')
go_files = $(shell find -type f -name '*.go')
extra_files = $(shell find web/static web/templates database/migrations -type f)

all: station-webapp
run: station-webapp
	./station-webapp
install: station-webapp
	sudo install station-webapp /usr/local/bin/station-webapp

clean:
	rm -rf station-webapp $(style)

station-webapp: $(go_files) $(extra_files) $(style)
	go build

$(style): $(sass_files)
	sass --no-source-map --style compressed $(sass)/index.scss $@

.PHONY: all run clean install
