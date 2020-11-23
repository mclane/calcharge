package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"time"

	"github.com/apognu/gocal"
	"gopkg.in/yaml.v2"
)

type Config struct {
	IcalServer   string `yaml:"icalserver"`
	IcalPort     string `yaml:"icalport"`
	User         string `yaml:"user"`
	CalendarFile string `yaml:"calendarfile"`

	Capa   int `yaml:"capa"`
	MaxCur int `yaml:"maxcur"`
}

var Cfg Config

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

func main() {
	// get config data
	Cfg.getConf()

	// read calendar data from server
	// connect to server
	url := "http://" + Cfg.IcalServer + ":" + Cfg.IcalPort + "/" + Cfg.User + "/" + Cfg.CalendarFile + ".ics/"
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// define timeslot
	start, end := time.Now(), time.Now().Add(24*time.Hour)

	// parse server response
	cal := gocal.NewParser(resp.Body)
	cal.Start, cal.End = &start, &end
	err = cal.Parse()
	if err != nil {
		panic(err)
	}

	var strt time.Time
	// print result
	for _, evnt := range cal.Events {
		fmt.Printf("%s on %s\n", evnt.Summary, evnt.Start)
		strt = *evnt.Start
	}

	// extract target soc and target time
	tsoc := 100

	if err != nil {
		panic(err)
	}

	// define charge start time
	// get actual soc
	asoc := 60

	// calculate charging time @ max charge current
	ct := 10.0 * float64(Cfg.Capa*(tsoc-asoc)) / 230.0 / float64(Cfg.MaxCur)
	fmt.Printf("Ladezeit: %f\n", ct)

	//calculate start time
	tstart := strt.Add(-time.Hour * time.Duration(ct))
	_, ctm := math.Modf(ct)
	tstart = tstart.Add(-time.Minute * time.Duration(60*ctm))
	fmt.Printf("Start: %v\n", tstart)

}
