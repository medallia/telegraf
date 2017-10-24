package netscaler

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

/*
 * Configuration sample for this plugin.
 */
var sampleConfig = `
  ## IP address of netscaler LB
  # host = "127.0.0.1"
  ## Username
  # username = "admin"
  ## Password
  # password = "admin"
  ## Use https for connection
  # https = false
  ## Path of the Nitro API "stat" endpoint
  # ApiPath = "/nitro/v1/stat"
`

var tr = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

var httpClient = &http.Client{
	Timeout:   time.Second * 3,
	Transport: tr,
}

/*
 * This interface is satisfied by every struct resulting from a
 * request to Netscaler's API.
 */
type NSResponse interface {
	PublishMetrics(acc telegraf.Accumulator)
}

/*
 * Main struct used by Telegraf for this plugin.
 */
type Netscaler struct {
	Host     string
	Username string
	Password string
	Https    bool
	ApiPath  string
}

func (n *Netscaler) Description() string {
	return "Telegraf plugin to gather metrics from netscaler load balancers"
}

func (n *Netscaler) SampleConfig() string {
	return sampleConfig
}

/*
 * Connects to a Netscaler and fetches the data.
 * It receives the path to the resource, constructs the URL and
 * does the connection.
 */
func (n *Netscaler) fetchResource(url_path string) ([]byte, error) {
	var http_prefix string
	var body []byte
	var err error = nil

	if n.Https {
		http_prefix = "https://"
	} else {
		http_prefix = "http://"
	}
	url := http_prefix + n.Host + n.ApiPath + url_path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return body, err
	}
	// Headers required by Netscaler's API interface
	req.Header.Add("X-NITRO-USER", n.Username)
	req.Header.Add("X-NITRO-PASS", n.Password)
	req.Header.Add("Content-type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal("Could not connect to the host:", err)
		return body, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return body, err
	}

	body, err = ioutil.ReadAll(resp.Body)
	return body, err
}

// Dispatches metrics to Telegraf
func (n *Netscaler) Gather(acc telegraf.Accumulator) error {
	resources := make(map[string]interface {
		PublishMetrics(acc telegraf.Accumulator)
	})
	resources["/lbvserver/"] = new(LBVResponse)

	for resource, resStruct := range resources {
		// Connect to Netscaler and gather the result from the resource
		body_raw, err := n.fetchResource(resource)
		if err != nil {
			log.Fatal("Coud not get results from resource %s: %s",
				resource, err.Error())
			return err
		}
		if err := json.Unmarshal(body_raw, &resStruct); err != nil {
			log.Fatalf("Could not parse json from resource %s: %s",
				resource, err.Error())
			return err
		}
		// Call the implementation of PublishMetrics in the interface.
		// Here is where metrics are actually dispatched.
		resStruct.PublishMetrics(acc)
	}
	return nil
}

/*
 * Get the tags from a struct.
 * The struct must tag the Field as `typeofmetric:"tag"` to be recognized
 * as a tag for the metric.
 * @Param: v: a reflect.Value type from the interface of the struct.
 * @Returns: A map of the type [string]string with the name and value
 *           of the tag.
 */
func getTags(v reflect.Value) map[string]string {
	tags := make(map[string]string)
	const tag string = "tag"

	for i := 0; i < v.NumField(); i++ {
		if ctag, ok := v.Type().Field(i).Tag.Lookup("typeofmetric"); ok {
			if ctag == tag {
				fieldName := v.Type().Field(i).Name
				fieldValue := reflect.Indirect(v).Field(i).Interface().(string)
				tags[fieldName] = fieldValue
			}
		}
	}
	return tags
}

/*
 * Get the tags from a struct.
 * The struct must tag the Field as `typeofmetric:"tag"` to be recognized
 * as a tag for the metric.
 * @Param: v: A reflect.Value type from the interface of the struct.
 * @Param: metricTagType: A string indicating how the Filed has been tagged
                          in the struct as `typeofmetric:<string>`.
 * @Returns: A map of the type [string]string with the name and value
 *           of the tag.
*/
func getMetrics(v reflect.Value, metricTagType string) map[string]interface{} {
	m := make(map[string]interface{})
	for i := 0; i < v.NumField(); i++ {
		if ctag, ok := v.Type().Field(i).Tag.Lookup("typeofmetric"); ok {
			if ctag == metricTagType {
				fieldName := v.Type().Field(i).Name
				fieldValue := reflect.Indirect(v).Field(i).Interface()
				m[fieldName] = fieldValue
			}
		}
	}
	return m
}

// Initializes the plugin
func init() {
	inputs.Add("netscaler", func() telegraf.Input {
		return &Netscaler{
			Host:     "127.0.0.1",
			Username: "admin",
			Password: "admin",
			Https:    false,
			ApiPath:  "/nitro/v1/stat",
		}
	})
}
