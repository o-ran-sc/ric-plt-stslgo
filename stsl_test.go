//
// Copyright 2022 Parallel Wireless
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

//  This source code is part of the near-RT RIC (RAN Intelligent Controller)
//  platform project (RICP).
package stslgo_test

import (
	"encoding/json"
	"fmt"
	"stslgo"
	"testing"

	_ "github.com/influxdata/influxdb1-client"
	"github.com/influxdata/influxdb1-client/models"
	timesrclient "github.com/influxdata/influxdb1-client/v2"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                     Mock Client And Its Methods
//                   Mock client structure implements the timesrclient.Iclient interface
//                   and mocks responses instead of using the TimeSeriesDB provided GO library.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
type MockClient struct{}

func (c *MockClient) Close() error {
	return nil
}

// Dynamic function for queryResponse so that based on the test case different outputs can be simulated
var queryResp func(q timesrclient.Query) (*timesrclient.Response, error)

func (c *MockClient) Query(q timesrclient.Query) (*timesrclient.Response, error) {
	return queryResp(q)
}

func (c *MockClient) Write(bp timesrclient.BatchPoints) error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                    Test & utility functions for the stslgo GO module
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Change to false when testing without authentication (default authentication is enabled for TimeSeriesDB)
var auth bool = true

// Setup the test environment using the mock interface
func setupIclientTest(timeserData *stslgo.TimeSeriesClientData) {
	var iclientIntf stslgo.TimeSeriesDataGoClient
	iclientIntf = &MockClient{}
	(*timeserData).Iclient = iclientIntf
}

// Setup the test environment using the TimeSeriesdb interface
func setupIclient(timeserData *stslgo.TimeSeriesClientData) {
	_ = timeserData.CreateTimeSeriesConnection()
}

// Setup the test environment for each test case
func setup() (timeserData *stslgo.TimeSeriesClientData, err error) {
	queryResp = func(q timesrclient.Query) (*timesrclient.Response, error) {
		result := timesrclient.Result{}
		resp := timesrclient.Response{}
		resp.Results = append(resp.Results, result)
		return &resp, nil
	}

	// Allocate and initialize the TimeSeriesClientData structure for TimeSeriesDB access
	timeserData = stslgo.NewTimeSeriesClientData("testdb", "testuser", "testpasswd")

	// IMP - Replace with setupIclient when running TimeSeriesDB is to be used instead of mock
	setupIclientTest(timeserData)

	if false == auth {
		// Create testdb
		err = timeserData.CreateTimeSeriesDB()
		if err != nil {
			return nil, err
		}

		err = timeserData.CreateRetentionPolicy("testdbrp", "2h", true)
		if err != nil {
			return nil, err
		}
	}
	return timeserData, err
}

// Setup the test environment for each test case
func setupWithRetentionPolicy() (timeserData *stslgo.TimeSeriesClientData, err error) {
	queryResp = func(q timesrclient.Query) (*timesrclient.Response, error) {
		result := timesrclient.Result{}
		resp := timesrclient.Response{}
		resp.Results = append(resp.Results, result)
		return &resp, nil
	}

	// Allocate and initialize the TimeSeriesClientData structure for TimeSeriesDB access
	timeserData = stslgo.NewTimeSeriesClientData("test2db", "test2user", "test2passwd")

	// IMP - Replace with setupIclient when running TimeSeriesDB is to be used instead of mock
	setupIclientTest(timeserData)

	if false == auth {
		// Create testdb
		err = timeserData.CreateTimeSeriesDBWithRetentionPolicy("test2rp", "1h")
		if err != nil {
			return nil, err
		}
	}
	return timeserData, err
}

// Test function to test basic get/set functions on TimeSeriesDB
func TestTimeSeriesDbGetSet(t *testing.T) {
	timeserData, err := setup()
	if err != nil {
		fmt.Println("Error in setup", err)
	}

	tableName := "SetGetTable"
	val := "3"
	newval, err := json.Marshal(&val)
	err = timeserData.Set(tableName, "a", newval)
	if err != nil {
		fmt.Printf("Unable to set data with error %v\n", err)
	}

	val = "2"
	newval, err = json.Marshal(&val)
	err = timeserData.Set(tableName, "a", newval)
	if err != nil {
		fmt.Printf("Unable to set data with error %v\n", err)
	}

	queryResp = func(q timesrclient.Query) (*timesrclient.Response, error) {
		result := timesrclient.Result{}

		var values [][]interface{}
		var value []interface{}
		value = append(value, "2021-08-20T05:47:46.275224998Z", "2")
		values = append(values, value)

		row := models.Row{Name: "SetGetTable", Columns: []string{"time", "a"}, Partial: false, Values: values}
		result.Series = append(result.Series, row)
		result.Err = ""

		resp := timesrclient.Response{}
		resp.Results = append(resp.Results, result)
		resp.Err = ""
		return &resp, nil
	}

	result, err := timeserData.Get(tableName, "a")
	if err != nil {
		fmt.Printf("Unable to get data with error %v\n", err)
		return
	}

	switch mytype := result.(type) {
	default:
		fmt.Printf("My type is %T and value %v\n", mytype, mytype)
	}

	err = timeserData.DropMeasurement(tableName)
	if err != nil {
		fmt.Printf("Unable to delete measurement with error %v\n", err)
		return
	}
}

// Test function for testing flattening and inserting of a json array as individual time points
func TestTimeSeriesDbJsonArrayFlatten(t *testing.T) {
	timeserData, err := setup()
	if err != nil {
		fmt.Println("Error in setup", err)
		return
	}

	if false == auth {
		// Alter the testDB retention policy
		err = timeserData.UpdateRetentionPolicy("testdbrp", "1h", true)
		if err != nil {
			fmt.Println("Error in updating retention policy", err)
			return
		}
	}

	// Array of two rows
	neighborCells := []byte(`[{"CID": "310-680-200-555001", "Cell-RF": {"rsp": -90, "rsrq": -13, "rsSinr": -2.5}}, {"CID": "310-680-200-555003", "Cell-RF": {"rsp": -140, "rsrq": -17, "rsSinr": -6}}]`)
	ignoreKeyList := []string{}

	err = timeserData.InsertJsonArray("FlattenJsonArrayTable", ignoreKeyList, neighborCells)
	if err != nil {
		fmt.Printf("\n Failed to flatten and insert the json array with error %s", err.Error())
	}
}

func TestTimeSeriesDbFlatten(t *testing.T) {
	timeserData, err := setupWithRetentionPolicy()
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

	err = timeserData.InsertJson("FlattenTable", ignoreKeyListMsg, msg)
	if err != nil {
		fmt.Printf("\n Failed to flatten and insert the json array with error %s", err.Error())
	}
}
