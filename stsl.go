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

package stslgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "github.com/influxdata/influxdb1-client"
	timesrclient "github.com/influxdata/influxdb1-client/v2"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                      Datastructures for storing all the timeseries db specific information
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
type TimeSeriesDataGoClient interface {
	Close() error
	Query(timesrclient.Query) (*timesrclient.Response, error)
	Write(bp timesrclient.BatchPoints) error
}

type TimeSeriesClientData struct {
	Iclient            TimeSeriesDataGoClient // Connection to TimeSeriesDB
	timeSeriesDbName   string                 // TimeSeries DB to be used for this XAPP
	timeSeriesUserName string                 // Username for accessing the TimeSeries DB
	timeSeriesPassword string                 // Password for accessing the TimeSeries DB
}

type JsonRow map[string]interface{}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                     Constructor for TimeSeriesClientData
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
func NewTimeSeriesClientData(dbName, userName, passWord string) *TimeSeriesClientData {
	zerolog.SetGlobalLevel(zerolog.InfoLevel) //default logging, can be changed using SetLoggingLevel()
	return &TimeSeriesClientData{
		timeSeriesDbName:   dbName,
		timeSeriesUserName: userName,
		timeSeriesPassword: passWord,
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                     Methods for TimeSeriesClientData
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
func (timeserData *TimeSeriesClientData) CreateTimeSeriesConnection() (err error) {
	// TimeSeriesDB specific intialization
	hostname := os.Getenv("TIMESERIESDB_SERVICE_HOST")
	if hostname == "" {
		hostname = "localhost"
	}
	port := os.Getenv("TIMESERIESDB_SERVICE_PORT_HTTP")
	if port == "" {
		port = "8086"
	}
	log.Info().Msgf("Establishing connection with TimeSeriesDB hostname: %v, port: %v\n", hostname, port)
	(*timeserData).Iclient, err = timesrclient.NewHTTPClient(timesrclient.HTTPConfig{
		Addr:     fmt.Sprintf("http://%v:%v", hostname, port),
		Username: (*timeserData).timeSeriesUserName,
		Password: (*timeserData).timeSeriesPassword,
	})
	if err != nil {
		log.Error().Msgf("Error creating TimeSeriesDB Client: %v\n", err.Error())
	} else {
		log.Info().Msgf("TimeSeriesDB Client created successfully: %v\n", (*timeserData).Iclient)
		defer timeserData.Iclient.Close()
	}
	return err
}

// Creates a new database
func (timeserData *TimeSeriesClientData) CreateTimeSeriesDB() (err error) {
	q := timesrclient.NewQuery(fmt.Sprintf("CREATE DATABASE %v", (*timeserData).timeSeriesDbName), "", "")

	if response, err := (*timeserData).Iclient.Query(q); err == nil && response.Error() == nil {
		log.Info().Msgf("Sucessfully created DB %v\n", (*timeserData).timeSeriesDbName)
	} else {
		log.Error().Msgf("Failed to create DB %v with error %v\n", (*timeserData).timeSeriesDbName, err)
	}
	return err
}

// Creates a new database
func (timeserData *TimeSeriesClientData) CreateTimeSeriesDBWithRetentionPolicy(retentionPolicyName, duration string) (err error) {
	q := timesrclient.NewQuery(fmt.Sprintf("CREATE DATABASE %v WITH DURATION %v REPLICATION 1 SHARD DURATION %v NAME %v", (*timeserData).timeSeriesDbName, duration, duration, retentionPolicyName), "", "")

	if response, err := (*timeserData).Iclient.Query(q); err == nil && response.Error() == nil {
		log.Info().Msgf("Sucessfully created DB %v with retention policy %v\n", (*timeserData).timeSeriesDbName, retentionPolicyName)
	} else {
		log.Error().Msgf("Failed to create DB %v with retention policy %v with error %v\n", (*timeserData).timeSeriesDbName, retentionPolicyName, err)
	}
	return err
}

// Deletes a database
func (timeserData *TimeSeriesClientData) DeleteTimeSeriesDB() (err error) {
	q := timesrclient.NewQuery(fmt.Sprintf("DROP DATABASE %v", (*timeserData).timeSeriesDbName), "", "")

	if response, err := (*timeserData).Iclient.Query(q); err == nil && response.Error() == nil {
		log.Info().Msgf("Sucessfully deleted DB %v\n", (*timeserData).timeSeriesDbName)
	} else {
		log.Error().Msgf("Failed to delete DB %v with error %v\n", (*timeserData).timeSeriesDbName, err)
	}
	return err
}

// Deletes a table
func (timeserData *TimeSeriesClientData) DropMeasurement(measurement string) (err error) {
	q := timesrclient.NewQuery(fmt.Sprintf("DELETE FROM %v", measurement), (*timeserData).timeSeriesDbName, "")

	if response, err := (*timeserData).Iclient.Query(q); err == nil && response.Error() == nil {
		log.Info().Msgf("Sucessfully deleted measurement %v\n", measurement)
	} else {
		log.Error().Msgf("Failed to delete measurement %v with error %v\n", measurement, err)
	}
	return err
}

// Set operation to mimic traditional key-value pair setting.
// PS - This creates new row than updating existing one to demonstrate time series capability
func (timeserData *TimeSeriesClientData) Set(measurement, key string, value []byte) (err error) {
	// Create a new point batch
	bp, _ := timesrclient.NewBatchPoints(timesrclient.BatchPointsConfig{
		Database:  (*timeserData).timeSeriesDbName,
		Precision: "ns",
	})

	// Create a point and add to batch
	tags := map[string]string{}
	fields := map[string]interface{}{
		key: value,
	}
	pt, err := timesrclient.NewPoint(measurement, tags, fields, time.Now())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	bp.AddPoint(pt)
	// Write the batch
	timeserData.Iclient.Write(bp)
	log.Debug().Msgf("TimeSeriesDB Set: DB=%v Measurement=%v key=%v, value=%v err=%v\n", timeserData.timeSeriesDbName, measurement, key, value, err)
	return err
}

// Get operation to mimic traditional key-value pair get operation
func (timeserData *TimeSeriesClientData) Get(measurement, key string) (result interface{}, err error) {
	queryStr := fmt.Sprintf("SELECT %v FROM %v ORDER BY time DESC LIMIT 1", key, measurement)
	q := timesrclient.NewQuery(queryStr, timeserData.timeSeriesDbName, "")
	if response, err := timeserData.Iclient.Query(q); err == nil && response.Error() == nil {
		for _, v := range response.Results {
			for _, row := range v.Series {
				for _, value := range row.Values {
					fmt.Printf("Row: %v, Value: %v\n", row, value)
					result = value[1] // value[0] is time
				}
			}
		}
	}
	log.Debug().Msgf("TimeSeriesDB Get: DB=%v Measurement=%v key=%v, value=%v err=%v\n", timeserData.timeSeriesDbName, measurement, key, result, err)
	return result, err
}

// Generic query operation
func (timeserData *TimeSeriesClientData) Query(queryStr string) (resp *timesrclient.Response, err error) {
	q := timesrclient.NewQuery(queryStr, timeserData.timeSeriesDbName, "")
	response, err := timeserData.Iclient.Query(q)
	log.Debug().Msgf("TimeSeriesDB Query: DB=%v, QueryString=%v, Result=%v, err=%v\n", timeserData.timeSeriesDbName, queryStr, response, err)
	return response, err
}

// Generic write point operation
func (timeserData *TimeSeriesClientData) WritePoint(measurement string, tags map[string]string, fields map[string]interface{}) (err error) {
	// Create a new point batch
	bp, _ := timesrclient.NewBatchPoints(timesrclient.BatchPointsConfig{
		Database:  (*timeserData).timeSeriesDbName,
		Precision: "ns",
	})

	// Create a point and add to batch
	pt, err := timesrclient.NewPoint(measurement, tags, fields, time.Now())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	bp.AddPoint(pt)
	// Write the batch
	timeserData.Iclient.Write(bp)
	log.Debug().Msgf("\nTimeSeriesDB WritePoint: DB=%v Measurement=%v tags=%v, fields=%v, err=%v", timeserData.timeSeriesDbName, measurement, tags, fields, err)
	return err
}

// Function to flatten nested json
func (timeserData *TimeSeriesClientData) Flatten(nested map[string]interface{}, prefix string, IgnoreKeyList []string) (map[string]interface{}, error) {
	flatmap := make(map[string]interface{})

	err := _flatten(true, flatmap, nested, prefix, IgnoreKeyList)
	if err != nil {
		return nil, err
	}

	return flatmap, nil
}

// Insert 1 or more Json Rows as a single batch
func (timeserData *TimeSeriesClientData) InsertUnmarshalledJsonRows(measurement string, rows []JsonRow, ignoreKeyList []string) (err error) {
	tags := make(map[string]string)
	field := make(map[string]interface{})

	bp, err := timesrclient.NewBatchPoints(timesrclient.BatchPointsConfig{
		Database:  (*timeserData).timeSeriesDbName,
		Precision: "ns",
	})

	for _, data := range rows {
		flatjson, err := timeserData.Flatten(data, "", ignoreKeyList)
		if err != nil {
			log.Warn().Msgf("\n Not able to flatten json %s for:%v", err.Error(), data)
		}

		log.Info().Msgf("\n Data after flattening: %v", flatjson)

		for key, value := range flatjson {
			if value != nil {
				if reflect.ValueOf(value).Type().Kind() == reflect.Float64 {
					field[key] = value
				} else if reflect.ValueOf(value).Type().Kind() == reflect.String {
					field[key] = value
				} else if reflect.ValueOf(value).Type().Kind() == reflect.Bool {
					field[key] = value
				} else if reflect.ValueOf(value).Type().Kind() == reflect.Int {
					field[key] = value
				}
			}
		}
		// Create a point and add to batch
		pt, err := timesrclient.NewPoint(measurement, tags, field, time.Now())
		if err != nil {
			log.Error().Msgf("Error: %s", err.Error())
			return err
		}
		bp.AddPoint(pt)
	}
	// Write the batch
	err = timeserData.Iclient.Write(bp)
	return err
}

// Function to flatten array of nested json
func (timeserData *TimeSeriesClientData) UnmarshallJsonRows(jsonBuffer []byte) ([]JsonRow, error) {

	// We create an empty array
	jsonrow := []JsonRow{}

	// Unmarshal the json into it. this will use the struct tag
	err := json.Unmarshal(jsonBuffer, &jsonrow)
	if err != nil {
		return nil, err
	}
	// the array is now filled with each row of json as an array index
	return jsonrow, nil
}

// Inserts JSON rows as separate time points in the mentioned measurement
func (timeserData *TimeSeriesClientData) InsertJsonArray(measurement string, ignoreList []string, jsonBuffer []byte) (err error) {
	rows, err := timeserData.UnmarshallJsonRows(jsonBuffer)
	if err == nil && len(rows) > 0 {
		// We can call InsertUnmarshalledJsonRow but it will do write for each row
		// Instead, use batching if rows more than 1
		err = timeserData.InsertUnmarshalledJsonRows(measurement, rows, ignoreList)
	}
	return err
}

// Inserts json data as single row in the mentioned meausrement
// PS - Use only for single row data
func (timeserData *TimeSeriesClientData) InsertJson(measurement string, ignoreList []string, jsonBuffer []byte) (err error) {
	tags := make(map[string]string)
	field := make(map[string]interface{})
	data := make(map[string]interface{})

	err = json.Unmarshal(jsonBuffer, &data)
	if err != nil {
		log.Error().Msgf("\n Not able to Parse data %s", err.Error())
		return err
	}

	bp, err := timesrclient.NewBatchPoints(timesrclient.BatchPointsConfig{
		Database:  (*timeserData).timeSeriesDbName,
		Precision: "ns",
	})

	flatjson, err := timeserData.Flatten(data, "", ignoreList)
	if err != nil {
		log.Error().Msgf("\n Not able to flatten json %s for:%v", err.Error(), data)
		return err
	}

	log.Info().Msgf("\n Data after flattening: %v", flatjson)

	for key, value := range flatjson {
		if value != nil {
			if reflect.ValueOf(value).Type().Kind() == reflect.Float64 {
				field[key] = value
			} else if reflect.ValueOf(value).Type().Kind() == reflect.String {
				field[key] = value
			} else if reflect.ValueOf(value).Type().Kind() == reflect.Bool {
				field[key] = value
			} else if reflect.ValueOf(value).Type().Kind() == reflect.Int {
				field[key] = value
			}
		}
	}
	// Create a point and add to batch
	pt, err := timesrclient.NewPoint(measurement, tags, field, time.Now())
	if err != nil {
		log.Error().Msgf("Error: %s", err.Error())
		return err
	}
	bp.AddPoint(pt)
	// Write the batch
	err = timeserData.Iclient.Write(bp)
	return err
}

// Creates a new retention policy
func (timeserData *TimeSeriesClientData) CreateRetentionPolicy(retentionPolicyName, duration string, setDefault bool) (err error) {
	isDefault := ""
	if true == setDefault {
		isDefault = "DEFAULT"
	}
	q := timesrclient.NewQuery(fmt.Sprintf("CREATE RETENTION POLICY %v ON %v DURATION %v REPLICATION 1 SHARD DURATION %v %v", retentionPolicyName, (*timeserData).timeSeriesDbName, duration, duration, isDefault), (*timeserData).timeSeriesDbName, "")
	if response, err := (*timeserData).Iclient.Query(q); err == nil && response.Error() == nil {
		log.Info().Msgf("Sucessfully created retention policy %v\n", retentionPolicyName)
	} else {
		log.Error().Msgf("Failed to create retention policy %v with error %v\n", retentionPolicyName, err)
	}
	return err
}

// Updates an existing retention policy
func (timeserData *TimeSeriesClientData) UpdateRetentionPolicy(retentionPolicyName, duration string, setDefault bool) (err error) {
	isDefault := ""
	if true == setDefault {
		isDefault = "DEFAULT"
	}
	q := timesrclient.NewQuery(fmt.Sprintf("ALTER RETENTION POLICY %v ON %v DURATION %v SHARD DURATION %v %v", retentionPolicyName, (*timeserData).timeSeriesDbName, duration, duration, isDefault), (*timeserData).timeSeriesDbName, "")
	if response, err := (*timeserData).Iclient.Query(q); err == nil && response.Error() == nil {
		log.Info().Msgf("Sucessfully updatated retention policy %v\n", retentionPolicyName)
	} else {
		log.Error().Msgf("Failed to updatate retention policy %v with error %v\n", retentionPolicyName, err)
	}
	return err
}

// Deletes an existing retention policy
func (timeserData *TimeSeriesClientData) DeleteRetentionPolicy(retentionPolicyName string) (err error) {
	q := timesrclient.NewQuery(fmt.Sprintf("DROP RETENTION POLICY %v ON %v", retentionPolicyName, (*timeserData).timeSeriesDbName), (*timeserData).timeSeriesDbName, "")

	if response, err := (*timeserData).Iclient.Query(q); err == nil && response.Error() == nil {
		log.Info().Msgf("Sucessfully deleted retention policy %v\n", retentionPolicyName)
	} else {
		log.Error().Msgf("Failed to delete retention policy %v with error %v\n", retentionPolicyName, err)
	}
	return err
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                       Generic functions - Non methods
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
func _flatten(top bool, flatMap map[string]interface{}, nested interface{}, prefix string, ignorelist []string) error {
	var flag int

	assign := func(newKey string, v interface{}, ignoretag bool) error {
		if ignoretag {
			switch v.(type) {
			case map[string]interface{}, []interface{}:
				v, err := json.Marshal(&v)
				if err != nil {
					log.Error().Msgf("\n Not able to Marshal data for key:%s=%v", newKey, v)
					return err
				}
				flatMap[newKey] = string(v)
			default:
				flatMap[newKey] = v
			}

		} else {
			switch v.(type) {
			case map[string]interface{}, []interface{}:
				if err := _flatten(false, flatMap, v, newKey, ignorelist); err != nil {
					log.Error().Msgf("\n Not able to flatten data for key:%s=%v", newKey, v)
					return err
				}
			default:
				flatMap[newKey] = v
			}
		}
		return nil
	}

	switch nested.(type) {
	case map[string]interface{}:
		for k, v := range nested.(map[string]interface{}) {

			ok := _matchkey(ignorelist, k)

			if ok && prefix == "" {
				flag = 1
			} else if ok && prefix != "" {
				flag = 0
			} else {
				flag = -1
			}

			if flag == 1 {
				err := assign(k, v, true)
				if err != nil {
					return err
				}
			} else if flag == 0 {
				newKey := _createkey(top, prefix, k)
				err := assign(newKey, v, true)
				if err != nil {
					return err
				}
			} else {
				newKey := _createkey(top, prefix, k)
				err := assign(newKey, v, false)
				if err != nil {
					return err
				}
			}
		}
	case []interface{}:
		for i, v := range nested.([]interface{}) {
			switch v.(type) {
			case map[string]interface{}:
				for tag, value := range v.(map[string]interface{}) {
					ok := _matchkey(ignorelist, tag)
					if ok {
						subkey := strconv.Itoa(i) + "." + tag
						newKey := _createkey(top, prefix, subkey)
						err := assign(newKey, value, true)
						if err != nil {
							return err
						}
					} else {
						newKey := _createkey(top, prefix, strconv.Itoa(i))
						err := assign(newKey, v, false)
						if err != nil {
							return err
						}
					}
				}
			default:
				newKey := _createkey(top, prefix, strconv.Itoa(i))
				err := assign(newKey, v, false)
				if err != nil {
					return err
				}
			}

		}
	default:
		return errors.New("Not a valid input: map or slice")
	}

	return nil
}

func _createkey(top bool, prefix, subkey string) string {
	key := prefix

	if top {
		key += subkey
	} else {
		key += "." + subkey
	}

	return key
}

func _matchkey(ignorelist []string, value string) bool {

	for _, val := range ignorelist {
		if val == value {
			return true
		}
	}

	return false
}

func SetLoggingLevel(level string) {

	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
}
