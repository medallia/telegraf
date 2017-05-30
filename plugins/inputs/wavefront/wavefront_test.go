package wavefront

import (
	"log"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/influxdata/telegraf/testutil"
)

const testServer = "localhost:1099"

var metrics = `docker.n.images 30 1496156870 engine_host="fib-r10-u05" source="fib-r10-u05"
test1.n.used.file.descriptors 26 1496156870 engine_host="fib-r10-u10" source="fib-r10-u10"
`

var individualMetrics = []string{
	`docker.n.images 30 1496156870 engine_host="fib-r10-u05" source="fib-r10-u05"`,
	`test1.n.used.file.descriptors 26 1496156870 engine_host="fib-r10-u10" source="fib-r10-u10"`,
}

var dockerFields = map[string]interface{}{
	"n.images": "30",
}

var dockerTags = map[string]string{
	"engine_host": "fib-r10-u05",
	"source":      "fib-r10-u05",
}

var test1Fields = map[string]interface{}{
	"n.used.file.descriptors": "26",
}

var test1Tags = map[string]string{
	"engine_host": "fib-r10-u10",
	"source":      "fib-r10-u10",
}

var metricStructs = []metric{
	metric{
		name:   "docker",
		fields: dockerFields,
		tags:   dockerTags,
		times: []time.Time{
			time.Unix(int64(1496156870), int64(0)),
		},
	},
	metric{
		name:   "test1",
		fields: test1Fields,
		tags:   test1Tags,
		times: []time.Time{
			time.Unix(int64(1496156870), int64(0)),
		},
	},
}

func sendMetric(metric string) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", testServer)
	if err != nil {
		log.Fatalf("ResolveTCPAddr failed: %v", err)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Fatalf("Could not dial wavefront input: %v", err)
	}

	if _, err = conn.Write([]byte(metric)); err != nil {
		log.Fatalf("Could not write metric to server: %v", err)
	}
}

func TestListen(t *testing.T) {
	w := &Wavefront{
		in:           make(chan string, 2),
		out:          make(chan metric, 2),
		done:         make(chan bool, 2),
		serverActive: make(chan bool, 1),
		Address:      testServer,
	}
	// Launch server
	go w.listen()

	// Wait for the server to go active
	<-w.serverActive

	// Dial the wavefront server and send the metrics
	sendMetric(metrics)
	// Make sure all metrics are as expected
	for _, metric := range individualMetrics {
		assert.Equal(t, metric, <-w.in)
	}
	// Gracefully shutdown server for next test
	w.done <- true
}

func TestParser(t *testing.T) {
	w := &Wavefront{
		in:           make(chan string, 2),
		out:          make(chan metric, 2),
		done:         make(chan bool, 2),
		serverActive: make(chan bool, 1),
		Address:      testServer,
	}
	for _, metric := range individualMetrics {
		w.in <- metric
	}
	go w.parser()

	for _, metric := range metricStructs {
		assert.Equal(t, metric, <-w.out)
	}
}

func TestGather(t *testing.T) {
	w := &Wavefront{
		in:           make(chan string, 2),
		out:          make(chan metric, 2),
		done:         make(chan bool, 2),
		serverActive: make(chan bool, 1),
		Address:      testServer,
	}

	var acc testutil.Accumulator
	go w.Gather(&acc)
	for _, metric := range metricStructs {
		w.out <- metric
	}

	acc.Wait(2)
	acc.AssertContainsTaggedFields(t, "docker", dockerFields, dockerTags)
	acc.AssertContainsTaggedFields(t, "test1", test1Fields, test1Tags)

}
