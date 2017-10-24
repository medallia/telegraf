package netscaler

import (
	"reflect"
	"time"

	"github.com/influxdata/telegraf"
)

/*
 * This response is included on every reply from Netscaler's stat API.
 */
type BaseJsonResponse struct {
	ErrorCode float64 `json:"errorcode"`
	Message   string  `json:"message"`
	Severity  string  `json:"severity"`
}

/*
 * Strcuture of each lbvserver response from '/lbvserver/' endpoint.
 * Fields are marked with `typeofmetric` to identify the type of value
 * it holds.
 */
type LBVServer struct {
	Name                         string  `json:"name" typeofmetric:"tag"`
	VSVRSurgeCount               float64 `json:"vsvrsurgecount,string" typeofmetric:"counter"`
	EstablishedConn              float64 `json:"establishedconn,string" typeofmetric:"gauge"`
	InactSvcs                    float64 `json:"inactsvcs,string" typeofmetric:"gauge"`
	VSLBHealth                   float64 `json:"vslbhealth,string" typeofmetric:"gauge"`
	State                        string  `json:"state" typeofmetric:"tag"`
	ActSvcs                      float64 `json:"actsvcs,string" typeofmetric:"gauge"`
	TottalHits                   float64 `json:"tothits,string" typeofmetric:"counter"`
	TotalRequests                float64 `json:"totalrequests,string" typeofmetric:"counter"`
	TotalResponses               float64 `json:"totalresponses,string" typeofmetric:"counter"`
	TotalRequestBytes            float64 `json:"totalrequestbytes,string" typeofmetric:"counter"`
	TotalResponseBytes           float64 `json:"totalresponsebytes,string" typeofmetric:"counter"`
	TotalPktsRecvd               float64 `json:"totalpktsrecvd,string" typeofmetric:"counter"`
	TotalPktsSent                float64 `json:"totalpktssent,string" typeofmetric:"counter"`
	CurrentClientConnections     float64 `json:"curclntconnections,string" typeofmetric:"gauge"`
	CurrentServerConnections     float64 `json:"cursrvrconnections,string" typeofmetric:"gauge"`
	SurgeCount                   float64 `json:"surgecount,string" typeofmetric:"counter"`
	SvcSurgeCount                float64 `json:"svcsurgecount,string" typeofmetric:"counter"`
	SpillOverThreshold           float64 `json:"sothreshold,string" typeofmetric:"gauge"`
	TotalSpillovers              float64 `json:"totspillovers,string" typeofmetric:"counter"`
	DeferredReqests              float64 `json:"deferredreq,string" typeofmetric:"counter"`
	InvalidRequestResponse       float64 `json:"invalidrequestresponse,string" typeofmetric:"counter"`
	InvalidRequestReponseDropped float64 `json:"invalidrequestresponsedropped,string" typeofmetric:"counter"`
	TotalVServerDownBackupHits   float64 `json:"totvserverdownbackuphits,string" typeofmetric:"counter"`
}

/*
 * Response from the '/lbvserver/' endpoint.
 */
type LBVResponse struct {
	BaseJsonResponse
	LBVservers []LBVServer `json:"lbvserver"`
}

/*
 * Implementation of the PublishMetrics interface for LBVServers.
 * Iterates on every 'lbvserver' and publishes the metrics to Telegraf.
 */
func (l *LBVResponse) PublishMetrics(acc telegraf.Accumulator) {
	for _, lbvh := range l.LBVservers {
		v := reflect.ValueOf(lbvh)
		counters := getMetrics(v, "counter")
		gauges := getMetrics(v, "gauge")
		tags := getTags(v)

		acc.AddCounter("netscaler.lbvserver", counters, tags, time.Now())
		acc.AddGauge("netscaler.lbvserver", gauges, tags, time.Now())
	}
}
