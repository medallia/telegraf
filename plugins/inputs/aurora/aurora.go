package aurora

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type Aurora struct {
	Timeout    int
	Master     string
	HttpPrefix string
	Numeric    bool
}

var sampleConfig = `
  ## Timeout, in ms.
  timeout = 100
  ## Aurora Master
  master = "localhost:8081"
  ## Http Prefix
  prefix = "http"
  ## Numeric values only
  numeric = true
`

// SampleConfig returns a sample configuration block
func (a *Aurora) SampleConfig() string {
	return sampleConfig
}

// Description just returns a short description of the Mesos plugin
func (a *Aurora) Description() string {
	return "Telegraf plugin for gathering metrics from N Apache Aurora Masters"
}

func (a *Aurora) SetDefaults() {
	if a.Timeout == 0 {
		log.Println("I! [aurora] Missing timeout value, setting default value (100ms)")
		a.Timeout = 1000
	} else if a.HttpPrefix == "" {
		log.Println("I! [aurora] Missing http prefix value, setting default value (http)")
		a.HttpPrefix = "http"
	}
}

func convertToNumeric(value string) (interface{}, bool) {
	var err error
	var val interface{}
	if val, err = strconv.ParseFloat(value, 64); err == nil {
		return val, true
	}
	if val, err = strconv.ParseBool(value); err != nil {
		return val.(bool), false
	}
	return val, true
}

func isJobMetric(key string) bool {
	// Regex for matching job specific tasks
	re := regexp.MustCompile("^sla_(.*?)/(.*?)/.*")
	return re.MatchString(key)
}

func parseJobSpecificMetric(key string, value interface{}) (map[string]interface{}, map[string]string) {
	// cut off the sla_
	key = key[4:]
	slashSplit := strings.Split(key, "/")
	role := slashSplit[0]
	env := slashSplit[1]
	underscoreIdx := strings.Index(slashSplit[2], "_")
	job := slashSplit[2][:underscoreIdx]
	metric := slashSplit[2][underscoreIdx+1:]

	fields := make(map[string]interface{})
	fields[metric] = value

	tags := make(map[string]string)
	tags["role"] = role
	tags["env"] = env
	tags["job"] = job
	return fields, tags
}

// Gather() metrics from given list of Aurora Masters
func (a *Aurora) Gather(acc telegraf.Accumulator) error {
	a.SetDefaults()

	client := http.Client{
		Timeout: time.Duration(a.Timeout) * time.Second,
	}
	url := fmt.Sprintf("%s://%s/vars", a.HttpPrefix, a.Master)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if isJobMetric(line) {
			splitIdx := strings.Index(line, " ")
			if splitIdx == -1 {
				continue
			}
			key := line[:splitIdx]
			value := line[splitIdx+1:]
			// If numeric is true and the metric is not numeric then ignore
			numeric, isNumeric := convertToNumeric(value)
			if a.Numeric && !isNumeric {
				continue
			}
			log.Printf("Key: %v. Numeric: %v", key, numeric)
			fields, tags := parseJobSpecificMetric(key, numeric)
			// Per job there are different tags so need to add a field per line
			acc.AddFields("aurora", fields, tags)
		}
	}
	return nil
}

func init() {
	inputs.Add("aurora", func() telegraf.Input {
		return &Aurora{}
	})
}
