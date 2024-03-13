# Station Web App

The station-webapp is a program that integrates with a weatherstation over mqtt.
It displays current weather conditions and keeps a history of previous conditions.

Everything needed to run is self contained within the application.

Some configuration is needed to start the server. The configuration file is by 
default `conf.toml`.


```toml
listen = ":8080"  # Listening Address for the http server
base = "/my-app"  # URL Prefix to all routes

db = "db.sqlite3" # File path to sqlite3 database

mqtt_server = "tcp://localhost:1883" # mqtt server to connect to 
mqtt_id = "my-mqtt-id" # id to join the mqtt server with
station_id = "station-mqtt-id" # id of the station that the server will connect to
```

Running the application is as simple as

```bash
mqtt-server [config.toml]
```


You can build and install the program using the provided Makefile. 

```bash
make all
make install # installs to /usr/bin/local/station-webapp
```
