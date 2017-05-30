package wavefront

import (
	"bufio"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

const (
	defaultAllowPending = 10000
)

type metric struct {
	name   string
	fields map[string]interface{}
	tags   map[string]string
	times  []time.Time
}

type Wavefront struct {
	Address string `toml:"address"`

	AllowedPendingMetrics int `toml:"allowed_pending"`

	// tracks the number of dropped metrics.
	drops int

	// Channel for all incoming wavefront packets
	in  chan string
	out chan metric

	done chan bool

	serverActive chan bool // For testing purposes to determine if it's listening
}

var sampleConfig = `
  ## Address
  address = 10.169.255.100:2878
`

// SampleConfig returns a sample configuration block
func (w *Wavefront) SampleConfig() string {
	return sampleConfig
}

// Description just returns a short description of the Mesos plugin
func (w *Wavefront) Description() string {
	return "Telegraf input plugin for gathering metrics from sources using wavefront format"
}

func (w *Wavefront) SetDefaults() {
	if w.Address == "" {
		log.Println("I! [wavefront] Missing address value, setting default value (10.169.255.100)")
		w.Address = "10.169.255.100"
	}
}

func (w *Wavefront) parser() error {
	for {
		select {
		case metricLine := <-w.in:
			// Convert all multiple spaces and tabs to
			r, _ := regexp.Compile("[ |\t]+")
			metricLine = strings.TrimSpace(string(r.ReplaceAll([]byte(metricLine), []byte(" "))))
			split := splitOnSpacesNotInQuotes(metricLine)
			// Scrub invalid values
			if split[1] == "nan" || split[1] == "Infinity" || split[1] == "null" || split[1] == "NaN" {
				continue
			}

			var metricTimes []time.Time
			tagIdx := 3 // Assumes there is a timestamp on the metric
			unixSeconds, err := strconv.ParseInt(split[2], 10, 64)
			// If it cannot be parsed then it is assumed that there is no timestamp and it is a tag instead
			if err != nil {
				tagIdx = 2
			} else {
				metricTimes = append(metricTimes, time.Unix(unixSeconds, int64(0)))
			}

			tags := make(map[string]string)
			for _, tagStr := range split[tagIdx:] {
				tagStr = strings.Replace(tagStr, "\"", "", -1)
				tagIdx := strings.Index(tagStr, "=")
				if tagIdx == -1 {
					log.Printf("I! Malformed tag on metric: %v\n", tagStr)
					continue
				}
				tags[tagStr[:tagIdx]] = tagStr[tagIdx+1:]
			}

			splitName := strings.Split(split[0], ".")
			if len(splitName) < 2 {
				log.Printf("I! Metric name is not namespaced. Skipping... %v\n", split[0])
				continue
			}

			value, isNumeric := convertToNumeric(split[1])
			// This would imply an invalid wavefront metric because they do not handle anything but strings
			if !isNumeric {
				continue
			}

			w.out <- metric{
				name: splitName[0],
				fields: map[string]interface{}{
					strings.Join(splitName[1:], "."): value,
				},
				tags:  tags,
				times: metricTimes,
			}
		}
	}
}

// Gather() metrics
func (w *Wavefront) Gather(acc telegraf.Accumulator) error {
LOOP:
	for {
		select {
		case m := <-w.out:
			acc.AddFields(m.name, m.fields, m.tags, m.times...)
		default:
			break LOOP
		}
	}
	return nil
}

func (w *Wavefront) Start(_ telegraf.Accumulator) error {
	log.Printf("I! Started the wavefront service on %s\n", w.Address)
	// Start the UDP listener
	go w.listen()
	// Start the line parser
	go w.parser()
	return nil
}

func (w *Wavefront) Stop() {
	w.done <- true
}

func init() {
	inputs.Add("wavefront", func() telegraf.Input {
		return &Wavefront{
			AllowedPendingMetrics: defaultAllowPending,
			in:           make(chan string, defaultAllowPending),
			out:          make(chan metric, defaultAllowPending),
			done:         make(chan bool, 10),
			serverActive: make(chan bool, 1),
			Address:      "10.169.100.100",
		}
	})
}

func (w *Wavefront) handleClient(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		wfMetric, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		select {
		case w.in <- strings.TrimSpace(wfMetric):
		default:
			w.drops++
			if w.drops != 0 {
				log.Printf("I! has dropped this many metrics: %v\n", w.drops)
			}
		}
	}
}

func (w *Wavefront) listen() {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", w.Address)
	if err != nil {
		panic(err)
	}
	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		panic(err)
	}

	// Close the listener when the application closes.
	defer l.Close()
	log.Printf("I! Listening on %v\n", w.Address)
	w.serverActive <- true
	acceptChan := make(chan bool, 1)
	acceptChan <- true
LISTENER:
	for {
		select {
		case <-acceptChan:
			// Listen for an incoming connection.
			conn, err := l.Accept()
			if err != nil {
				log.Printf("E! Error accepting new metrics: %s\n", err.Error())
			}
			go w.handleClient(conn)
			acceptChan <- true
		case <-w.done:
			log.Printf("I! Stopping listener\n")
			break LISTENER
		}
	}
}

// Converts string values taken from aurora vars to numeric values for wavefront
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

func splitOnSpacesNotInQuotes(wfString string) []string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)

		}
	}
	return strings.FieldsFunc(wfString, f)
}
