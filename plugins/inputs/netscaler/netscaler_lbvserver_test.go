package netscaler

import (
  "testing"
  "encoding/json"
  "io/ioutil"
  "log"
	"github.com/influxdata/telegraf/testutil"
)


func TestLBVServerPublishMetrics(t *testing.T) {
  nsreseponse := new(LBVResponse)
  acc := new(testutil.Accumulator)

  json_source, err := ioutil.ReadFile("netscaler_lbvserver_response.json")
  if (err != nil) {
    log.Fatal("Could not find the mock json response file: %s", err.Error())
  }

  if err := json.Unmarshal(json_source, &nsreseponse); err != nil {
    log.Fatal("Could not parse json from resource: %s", err.Error())
  }

  nsreseponse.PublishMetrics(acc)
}
