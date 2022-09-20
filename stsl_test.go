//
// Copyright 2022 Parallel Wireless
// Copyright 2022 Samsung Electronics Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// This source code is part of the near-RT RIC (RAN Intelligent Controller)
// platform project (RICP).
package stslgo

import (
	"fmt"
	"os"
	"testing"
	"time"
)

var (
	serverURL string
	authToken string
	orgName   string
	dbName    string
)

func getEnvValue(key, defVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	} else {
		return defVal
	}
}

func init() {
	serverURL = getEnvValue("TIMESERIESDB_SERVICE_HOST", "http://localhost:8086")
	authToken = getEnvValue("TIMESERIESDB_SERVICE_TOKEN", "my-token")
	orgName = getEnvValue("TIMESERIESDB_SERVICE_ORG_NAME", "influxdata")
	dbName = getEnvValue("TIMESERIESDB_DB_NAME ", "default")
}

// Setup the test environment for each test case
func setup(t *testing.T) (tsCli *TimeSeriesClientData, err error) {
	// Allocate and initialize the TimeSeriesClient structure for TimeSeriesDB access
	tsCli = NewTimeSeriesClientData(dbName)

	// Setup the test environment using the TimeSeriesdb interface
	err = tsCli.CreateTimeSeriesConnection()
	if err != nil {
		fmt.Println("Error in connection", err)
	}

	// Create testdb
	tsCli.CreateTimeSeriesDB()
	return tsCli, err
}

func TestTimeSeriesDbCreate(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	err = tsCli.CreateTimeSeriesDBWithRetentionPolicy("24h")
	if err != nil {
		fmt.Println("Error in create DB", err)
	}
}
func TestTimeSeriesDbDelete(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	err = tsCli.DeleteTimeSeriesDB()
	if err != nil {
		fmt.Println("Error in delete DB", err)
	}
}

func TestTimeSeriesDbUpdate(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	err = tsCli.UpdateTimeSeriesDBRetentionPolicy("")
	if err != nil {
		fmt.Println("Error in delete DB", err)
	}
}

func TestTimeSeriesDbWritePoint(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	err = tsCli.WritePoint("testMeasurement",
		map[string]string{
			"tagKey1": "tagVal_a",
		},
		map[string]interface{}{
			"fieldKey1": 3,
		})
	if err != nil {
		fmt.Println("Error in delete DB", err)
	}
}

func TestTimeSeriesDbUpdateRetentionPolicy(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	RP := tsCli.timeSeriesDB.RetentionPolicy
	if RP == "" {
		RP = "infinite"
	}
	fmt.Println("from : ", RP)
	err = tsCli.UpdateTimeSeriesDBRetentionPolicy("24h")
	if err != nil {
		fmt.Println("Error in delete DB", err)
	}

	RP = tsCli.timeSeriesDB.RetentionPolicy
	if RP == "" {
		RP = "infinite"
	}
	fmt.Println("to : ", RP)
}

func TestTimeSeriesDbDropMeasurement(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	err = tsCli.DropMeasurement("testMeasurement")
	if err != nil {
		fmt.Println("Error in delete DB", err)
	}
}

func TestTimeSeriesSetAndGet(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	tableName := "FloatSetGetTable"
	val := 2.0
	err = tsCli.Set(tableName, "a", val)
	if err != nil {
		fmt.Printf("Unable to set data with error %v", err)
	}

	time.Sleep(1 * time.Second)

	result, err := tsCli.Get(tableName, "a")
	if err != nil {
		fmt.Printf("Unable to get data with error %v", err)
		return
	}
	fmt.Printf("Result = %v, type = %T \n", result, result)

	newFloatVal := 33.3
	err = tsCli.Set(tableName, "a", newFloatVal)
	if err != nil {
		fmt.Printf("Unable to set data with error %v", err)
	}

	time.Sleep(1 * time.Second)

	result, err = tsCli.Get(tableName, "a")
	if err != nil {
		fmt.Printf("Unable to get data with error %v", err)
		return
	}
	fmt.Printf("Result = %v, type = %T \n", result, result)
}

func TestTimeSeriesGenericQuery(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	err = tsCli.WritePoint("testMeasurement",
		map[string]string{
			"tagKey1": "tagVal_a",
		},
		map[string]interface{}{
			"fieldKey1": 3,
		})
	if err != nil {
		fmt.Println("Error in delete DB", err)
	}

	measurement := "testMeasurement"
	flux := fmt.Sprintf(`
	from(bucket: "%s") 
	|> range(start: -24h) 
	|> filter(fn: (r) => r._measurement == "%s")
	`, dbName, measurement)

	resp, err := tsCli.Query(flux)
	if err == nil {
		// Iterate over query response
		for resp.Next() {
			// Access data
			fmt.Printf("value: %v\n", resp.Record().Value())
		}
		// check for an error
		if resp.Err() != nil {
			fmt.Printf("query parsing error: %s\n", resp.Err().Error())
		}
	} else {
		fmt.Printf("Unable to query data with error %v\n", err)
	}
}

func TestTimeSeriesDbJsonArrayFlatten(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
		return
	}

	// Array of two rows
	neighborCells := []byte(`[{"CID": "310-680-200-555001", "Cell-RF": {"rsp": -90, "rsrq": -13, "rsSinr": -2.5}}, {"CID": "310-680-200-555003", "Cell-RF": {"rsp": -140, "rsrq": -17, "rsSinr": -6}}]`)
	ignoreKeyList := []string{}

	err = tsCli.InsertJsonArray("FlattenJsonArrayTable", ignoreKeyList, neighborCells)
	if err != nil {
		fmt.Printf("\n Failed to flatten and insert the json array with error %s", err.Error())
	}
}

func TestTimeSeriesDbFlatten(t *testing.T) {
	tsCli, err := setup(t)
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	// Single row
	msg := []byte(`{"intdata": [10,24,43,56,45,78],
                      "floatdata": [56.67, 45.68, 78.12],
                      "nested_data": {
                          "key1": "string_data",
                          "key2": [45, 56],
                          "key3": [60.8, 45.78]
                      }}`)
	ignoreKeyListMsg := []string{"floatdata", "key2"}

	err = tsCli.InsertJson("FlattenTable", ignoreKeyListMsg, msg)
	if err != nil {
		fmt.Printf("\n Failed to flatten and insert the json array with error %s", err.Error())
	}
}

func TestRPIntToString(t *testing.T) {
	rp := "3w4d12m30s"
	rpi, err := rpStringToInt64(rp)
	if err != nil {
		fmt.Println(err)
	}
	rps := rpInt64ToString(rpi)
	fmt.Printf("rp : %s, rpi : %d, rps : %s \n", rp, rpi, rps)
}
