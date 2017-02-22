package aurora

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/influxdata/telegraf/testutil"
)

var masterServer *httptest.Server

var referenceMetrics = map[string]interface{} {
	"assigner_launch_failures": "0",
	"cron_job_triggers": "240",
	"sla_cluster_mtta_ms": "18",
	"sla_disk_small_mttr_ms": "1029",
	"job_uptime_50.00_sec": "689291",
	"sla_cpu_small_mtta_ms": "17",
}

//sla_role/prod/jobname_

func getRawMetrics() string {
	return `assigner_launch_failures 0
cron_job_triggers 240
sla_cluster_mtta_ms 18
sla_disk_small_mttr_ms 1029
sla_role1/prod1/jobname1_job_uptime_50.00_sec 689291
sla_cpu_small_mtta_ms 17
jvm_prop_java.endorsed.dirs /usr/lib/jvm/java-8-openjdk-amd64/jre/lib/endorsed
sla_role2/prod2/jobname2_job_uptime_50.00_sec 99`
}

func TestMain(m *testing.M) {
	metrics := getRawMetrics()

	masterRouter := http.NewServeMux()
	masterRouter.HandleFunc("/vars", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, metrics)
	})
	masterServer = httptest.NewServer(masterRouter)

	rc := m.Run()

	masterServer.Close()
	os.Exit(rc)
}

func TestAuroraMaster(t *testing.T) {
	var acc testutil.Accumulator
	acc.SetDebug(true)

	m := Aurora{
		Master: masterServer.Listener.Addr().String(),
		Timeout: 10,
		HttpPrefix: "http",
		Numeric: true,
	}

	err := m.Gather(&acc)
	if err != nil {
		t.Errorf(err.Error())
	}

	acc.AssertContainsFields(t, "aurora", referenceMetrics)
	fmt.Printf("\n\n")
	acc.AssertContainsFields(t, "aurora", referenceMetrics)

	// acc.AssertContainsFields(t, "mesos", masterMetrics)
}

// func TestMasterFilter(t *testing.T) {
// 	m := Mesos{
// 		MasterCols: []string{
// 			"resources", "master", "registrar",
// 		},
// 	}
// 	b := []string{
// 		"system", "agents", "frameworks",
// 		"messages", "evqueue", "tasks",
// 	}

// 	m.filterMetrics(MASTER, &masterMetrics)

// 	for _, v := range b {
// 		for _, x := range getMetrics(MASTER, v) {
// 			if _, ok := masterMetrics[x]; ok {
// 				t.Errorf("Found key %s, it should be gone.", x)
// 			}
// 		}
// 	}
// 	for _, v := range m.MasterCols {
// 		for _, x := range getMetrics(MASTER, v) {
// 			if _, ok := masterMetrics[x]; !ok {
// 				t.Errorf("Didn't find key %s, it should present.", x)
// 			}
// 		}
// 	}
// }

// func TestMesosSlave(t *testing.T) {
// 	var acc testutil.Accumulator

// 	m := Mesos{
// 		Masters: []string{},
// 		Slaves:  []string{slaveTestServer.Listener.Addr().String()},
// 		// SlaveTasks: true,
// 		Timeout: 10,
// 	}

// 	err := m.Gather(&acc)

// 	if err != nil {
// 		t.Errorf(err.Error())
// 	}

// 	acc.AssertContainsFields(t, "mesos", slaveMetrics)

// 	// expectedFields := make(map[string]interface{}, len(slaveTaskMetrics["statistics"].(map[string]interface{}))+1)
// 	// for k, v := range slaveTaskMetrics["statistics"].(map[string]interface{}) {
// 	// 	expectedFields[k] = v
// 	// }
// 	// expectedFields["executor_id"] = slaveTaskMetrics["executor_id"]

// 	// acc.AssertContainsTaggedFields(
// 	// 	t,
// 	// 	"mesos_tasks",
// 	// 	expectedFields,
// 	// 	map[string]string{"server": "127.0.0.1", "framework_id": slaveTaskMetrics["framework_id"].(string)})
// }

// func TestSlaveFilter(t *testing.T) {
// 	m := Mesos{
// 		SlaveCols: []string{
// 			"resources", "agent", "tasks",
// 		},
// 	}
// 	b := []string{
// 		"system", "executors", "messages",
// 	}

// 	m.filterMetrics(SLAVE, &slaveMetrics)

// 	for _, v := range b {
// 		for _, x := range getMetrics(SLAVE, v) {
// 			if _, ok := slaveMetrics[x]; ok {
// 				t.Errorf("Found key %s, it should be gone.", x)
// 			}
// 		}
// 	}
// 	for _, v := range m.MasterCols {
// 		for _, x := range getMetrics(SLAVE, v) {
// 			if _, ok := slaveMetrics[x]; !ok {
// 				t.Errorf("Didn't find key %s, it should present.", x)
// 			}
// 		}
// 	}
// }
