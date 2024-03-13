package station

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"github.com/ttocsneb/station-webapp/database"
	"github.com/ttocsneb/station-webapp/util"
)

var Client *Station = nil

type Station struct {
	Client        mqtt.Client
	db            *sql.DB
	station       string
	updates       *util.ChanMux[database.Condition]
	rapid         *util.ChanMux[database.Condition]
	updates_chan  chan database.Condition
	rapid_chan    chan database.Condition
	rapid_done    chan any
	rapid_running bool
}

func WaitOrErr(fut mqtt.Token) error {
	fut.Wait()
	return fut.Error()
}

func NewStation(db *sql.DB, client_id string, station_id string, server string) (*Station, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(server)
	opts.SetClientID(client_id)
	opts.SetOrderMatters(false)
	client := mqtt.NewClient(opts)

	if err := WaitOrErr(client.Connect()); err != nil {
		return nil, err
	}
	updates_chan := make(chan database.Condition)
	rapid_chan := make(chan database.Condition)
	self := &Station{
		Client:        client,
		db:            db,
		station:       station_id,
		updates:       util.NewChanMux(updates_chan),
		rapid:         util.NewChanMux(rapid_chan),
		updates_chan:  updates_chan,
		rapid_chan:    rapid_chan,
		rapid_done:    make(chan any),
		rapid_running: false,
	}
	self.rapid.OnEmpty = self.stopRapdiUpdates
	self.rapid.OnSubscribe = self.startRapidUpdates

	if err := WaitOrErr(client.Subscribe(fmt.Sprintf("/station/weather/%v", station_id), 0, self.weatherListener())); err != nil {
		return nil, err
	}
	logrus.Infof("Subscribing to /station/weather/%v", station_id)

	return self, nil
}

func (self *Station) weatherListener() mqtt.MessageHandler {
	return func(cient mqtt.Client, msg mqtt.Message) {
		var payload weatherMessage
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			logrus.Errorf("Unable to parse message: %v\n", err)
			return
		}

		conditions := database.NewCondition(payload.Time)

		for sensor, value := range payload.Sensors {
			if len(value) == 0 {
				continue
			}
			val := value[0]
			conditions.Sensors[sensor] = val.Value
		}

		if err := conditions.InsertDb(self.db); err != nil {
			logrus.Errorf("Unable to insert condition to db: %v\n", err)
			return
		}

		logrus.Info("Received conditions update")

		self.updates_chan <- conditions

		yes, err := database.IsTimeToReduce(self.db)
		if err != nil {
			logrus.Error(err)
			return
		}
		if yes {
			go func() {
				logrus.Info("Reducing database")
				err := database.ReduceConditions(self.db)
				if err != nil {
					logrus.Errorf("Error While reducing database: %v\n", err)
				}
			}()
		}
	}
}

func (self *Station) startRapidUpdates() {
	logrus.Info("Starting rapid updates")
	subscription := fmt.Sprintf("/station/rapid-weather/%v", self.station)
	request := fmt.Sprintf("/station/request/%v", self.station)

	logrus.Infof("Subscribing to %v", subscription)
	err := WaitOrErr(self.Client.Subscribe(subscription, 1, func(client mqtt.Client, msg mqtt.Message) {
		var payload weatherMessage
		err := json.Unmarshal(msg.Payload(), &payload)
		if err != nil {
			logrus.Errorf("Could not parse rapid-weather message: %v\n", err)
			return
		}

		message := database.NewCondition(payload.Time)

		for sensor, values := range payload.Sensors {
			if len(values) == 0 {
				continue
			}
			message.Sensors[sensor] = values[0].Value
		}

		self.rapid_chan <- message
	}))
	if err != nil {
		logrus.Errorf("%v\n", err)
		return
	}

	payload, err := json.Marshal(requestMessage{Action: "rapid-weather"})
	if err != nil {
		logrus.Errorf("%v\n", err)
		return
	}

	err = WaitOrErr(self.Client.Publish(request, 1, false, payload))
	if err != nil {
		logrus.Errorf("%v\n", err)
		return
	}

	go func() {
		self.rapid_running = true
		timeout := time.After(time.Second * 50)
		for true {
			select {
			case <-self.rapid_done:
				err := WaitOrErr(self.Client.Unsubscribe(subscription))
				if err != nil {
					logrus.Errorf("Could not Unsubscribe from rapid-weather updates: %v\n", err)
				}
				self.rapid_running = false
				logrus.Infof("Unsubscribe from %v", subscription)
				return
			case <-timeout:
				logrus.Infof("Publishing to %v", request)
				err := WaitOrErr(self.Client.Publish(request, 1, false, payload))
				if err != nil {
					logrus.Errorf("Could not send rapid-weather request: %v\n", err)
				}
				timeout = time.After(time.Second * 50)
			}
		}
	}()
}

func (self *Station) stopRapdiUpdates() {
	if self.rapid_running {
		self.rapid_done <- true
	}
}

func (self *Station) SubscribeUpdates() chan database.Condition {
	return self.updates.Subscribe(1)
}
func (self *Station) UnsubscribeUpdates(c chan database.Condition) {
	self.updates.Unsubscribe(c)
}

func (self *Station) SubscribeRapid() chan database.Condition {
	return self.rapid.Subscribe(1)
}
func (self *Station) UnsubscribeRapid(c chan database.Condition) {
	self.rapid.Unsubscribe(c)
}
