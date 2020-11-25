package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/apognu/gocal"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Ical struct {
		IcalUri string        `yaml:"icaluri"`
		ChkInt  time.Duration `yaml:"chkint"`
	} `yaml:"ical"`

	Capa       int `yaml:"capa"`
	MaxCur     int `yaml:"maxcur"`
	Mqttbroker struct {
		Name     string `yaml:"name"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"mqttbroker"`
}

var Cfg Config
var running bool = false
var tsoc int = 0
var strt *time.Time
var chktime = time.Now()

func (c *Config) getConf() *Config {

	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		panic(err)
	}

	return c
}

func getCalData() (int, *time.Time, error) {
	// open calendar server
	resp, err := http.Get(Cfg.Ical.IcalUri)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	// define timeslot
	start, end := time.Now(), time.Now().Add(24*time.Hour)

	// parse server response
	cal := gocal.NewParser(resp.Body)
	cal.Start, cal.End = &start, &end
	err = cal.Parse()
	if err != nil {
		return 0, nil, err
	}

	// check if events are planned for the next 24 h
	if len(cal.Events) == 0 {
		return 0, nil, fmt.Errorf("no event")
	}

	// return values for first event only
	fmt.Printf("%s on %s\n", cal.Events[0].Summary, cal.Events[0].Start)

	sparts := strings.Split(cal.Events[0].Summary, " ")
	soc64, err2 := strconv.ParseInt(strings.Trim(sparts[1], "%"), 10, 32)
	if err2 != nil {
		return 0, nil, err2
	}

	// check if valid request
	if sparts[0] != "SoC" {
		return 0, nil, fmt.Errorf("keyword SoC missing")
	}

	// check if requested SoC is in reasonable range
	if soc64 <= 0 || soc64 > 100 {
		return 0, nil, fmt.Errorf("SoC out of range")
	}

	return int(soc64), cal.Events[0].Start, nil
}

var msgSubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	// get actual soc via MQTT
	sb, err := strconv.ParseInt(string(msg.Payload()), 10, 64)
	if err != nil {
		panic(err)
	}
	asoc := int(sb)

	// check calendar every chkint minutes and get target soc and time
	if !running && time.Until(chktime.Add(Cfg.Ical.ChkInt*time.Minute)) < 0 {
		var e error
		tsoc, strt, e = getCalData()
		if e == nil {
			running = true
		} else {
			// no valid data
			fmt.Printf("Fehler: %s\n", e)
		}
		chktime = time.Now()
	}

	if running {
		// valid data available, calculate charging time @ max charge current
		ct := 10.0 * float64(Cfg.Capa*(tsoc-asoc)) / 230.0 / float64(Cfg.MaxCur)
		fmt.Printf("Ladezeit: %f\n", ct)

		//calculate start time
		tstart := strt.Add(-time.Minute * time.Duration(60*ct))
		fmt.Printf("Start: %v\n", tstart)

		// check if the point of no return is reached
		if time.Until(tstart) <= 0 {
			fmt.Printf("%v: nun aber volle Pulle!\n", time.Now())
			token := client.Publish("evcc/loadpoints/1/mode/set", 0, false, "now")
			token.Wait()
		} else {
			fmt.Printf("kann noch warten\n")
		}
		if time.Now().After(*strt) {
			// charging finished
			running = false
		}
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	token := client.Subscribe("evcc/loadpoints/1/socCharge", 0, msgSubHandler)
	if token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}

func connectToMqtt() {
	muri := "tcp://" + Cfg.Mqttbroker.Name + ":" + Cfg.Mqttbroker.Port
	opts := mqtt.NewClientOptions().AddBroker(muri)
	opts.SetUsername(Cfg.Mqttbroker.User)
	opts.SetPassword(Cfg.Mqttbroker.Password)
	opts.SetClientID("evcc-CalCh")
	//opts.SetDefaultPublishHandler(msgPubHandler)

	opts.OnConnect = connectHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// get config data
	Cfg.getConf()
	connectToMqtt()
	<-c
}
