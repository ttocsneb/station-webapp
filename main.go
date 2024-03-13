package main

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ttocsneb/station-webapp/database"
	"github.com/ttocsneb/station-webapp/station"
	"github.com/ttocsneb/station-webapp/util"
	"github.com/ttocsneb/station-webapp/web"
)

func main() {

	path := "conf.toml"
	if len(os.Args) >= 2 {
		path = os.Args[1]
	}

	conf, err := util.LoadConfig(path)
	if err != nil {
		panic(err)
	}

	db, err := sql.Open("sqlite3", conf.Db)
	if err != nil {
		panic(err)
	}

	database.DB = db

	err = database.Migrate(db)
	if err != nil {
		panic(err)
	}

	client, err := station.NewStation(db, conf.MqttId, conf.StationId, conf.MqttServer)
	if err != nil {
		panic(err)
	}
	station.Client = client

	web.Main(db, client)
}
