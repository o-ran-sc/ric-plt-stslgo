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

//  This source code is part of the near-RT RIC (RAN Intelligent Controller)
//  platform project (RICP).

package stslgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/domain"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
//	Datastructures for storing all the timeseries db specific information
//
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
type TimeSeriesClientData struct {
	iClient           influxdb2.Client // Connection to TimeSeriesDB
	timeSeriesOrgName string           // The organization including TimeSeriesDB
	timeSeriesDB      TimeSeriesDB     // TimeSeriesDB to be used for this XAPP
}

type TimeSeriesDB struct {
	Name            string
	RetentionPolicy string
	CreatedTime     time.Time
}

type JsonRow map[string]interface{}

const (
	TIMESERIESDB_DEFAULT_SERVICE_ORG_NAME = "influxdata"
	TIMESERIESDB_DEFAULT_DB_NAME          = "default"
	TIMESERIESDB_DEFAULT_RETENTION_POLICY = ""
	TIMESERIESDB_DEFAULT_SERVICE_HOST     = "http://127.0.0.1:8086"
)

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
//	Constructor for TimeSeriesClientData
//
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
func NewTimeSeriesClientData(dbName string) *TimeSeriesClientData {
	zerolog.SetGlobalLevel(zerolog.InfoLevel) //default logging, can be changed using SetLoggingLevel()
	if dbName == "" {
		dbName = TIMESERIESDB_DEFAULT_DB_NAME
	}

	orgName := os.Getenv("TIMESERIESDB_SERVICE_ORG_NAME")
	if orgName == "" {
		orgName = TIMESERIESDB_DEFAULT_SERVICE_ORG_NAME
	}

	timeserData := &TimeSeriesClientData{
		timeSeriesOrgName: orgName,
		timeSeriesDB: TimeSeriesDB{
			Name:            dbName,
			RetentionPolicy: TIMESERIESDB_DEFAULT_RETENTION_POLICY,
		},
	}

	log.Info().Msgf("TimeSeriesDB Client created successfully: %+v\n", timeserData)
	return timeserData
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
//	Methods for TimeSeriesClientData
//
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
func (tscd *TimeSeriesClientData) CreateTimeSeriesConnection() (err error) {
	host := os.Getenv("TIMESERIESDB_SERVICE_HOST")
	if host == "" {
		host = TIMESERIESDB_DEFAULT_SERVICE_HOST
	}
	token := os.Getenv("TIMESERIESDB_SERVICE_TOKEN")

	log.Info().Msgf("Establishing connection with TimeSeriesDB host: %v\n", host)
	(*tscd).iClient = influxdb2.NewClient(host, token)
	defer tscd.iClient.Close()

	health, err := (*tscd).iClient.Health(context.Background())

	if err != nil || health.Status != domain.HealthCheckStatusPass {
		log.Error().Msgf("Error checking TimeSeriesDB Client health: %+v\n", err.Error())
		return
	}

	log.Info().Msgf("TimeSeriesDB Client connected successfully: %+v\n", (*tscd).iClient)
	return
}

// Creates a new database
func (tscd *TimeSeriesClientData) CreateTimeSeriesDB() (err error) {
	// Empty retention policy makes the database with 0s duration, which means infinite retention
	return tscd.CreateTimeSeriesDBWithRetentionPolicy("")
}

func (tscd *TimeSeriesClientData) CreateTimeSeriesDBWithRetentionPolicy(retentionPolicy string) (err error) {
	orgName := (*tscd).timeSeriesOrgName
	bucketName := (*tscd).timeSeriesDB.Name
	bucketsAPI := (*tscd).iClient.BucketsAPI()

	orgAPI := tscd.iClient.OrganizationsAPI()
	org, err := orgAPI.FindOrganizationByName(context.Background(), orgName)
	if err != nil {
		log.Error().Msgf("Failed to find organization %v with error: %v\n", orgName, err)
		return
	}

	bucket, err := bucketsAPI.FindBucketByName(context.Background(), bucketName)
	if bucket != nil {
		log.Debug().Msgf("TimeSeriesDB with name %v already exists", bucketName)

		tscd.timeSeriesDB.RetentionPolicy = rpInt64ToString(bucket.RetentionRules[0].EverySeconds)
		tscd.timeSeriesDB.CreatedTime = *bucket.CreatedAt
		return
	}

	duration, err := rpStringToInt64(retentionPolicy)
	if err != nil {
		log.Error().Msgf("Failed to convert retention policy %v to duration with error: %v\n", retentionPolicy, err)
		return
	}

	bucket, err = bucketsAPI.CreateBucketWithName(context.Background(), org, bucketName, domain.RetentionRule{
		EverySeconds: duration,
	})

	if err != nil {
		log.Error().Msgf("Failed to create TimeSeriesDB %v with error: %v\n", bucketName, err)
	}

	tscd.timeSeriesDB.RetentionPolicy = retentionPolicy
	tscd.timeSeriesDB.CreatedTime = *bucket.CreatedAt
	log.Info().Msgf("Sucessfully created TimeSeriesDB with name %v, at %v\n", bucketName, tscd.timeSeriesDB.CreatedTime)
	return
}

// Deletes a database
func (tscd *TimeSeriesClientData) DeleteTimeSeriesDB() (err error) {
	bucketName := (*tscd).timeSeriesDB.Name
	bucketsAPI := (*tscd).iClient.BucketsAPI()
	bucket, err := bucketsAPI.FindBucketByName(context.Background(), bucketName)
	if bucket == nil {
		log.Error().Msgf("Failed to find TimeSeriesDB with name %v", bucketName)
		return
	}

	err = bucketsAPI.DeleteBucket(context.Background(), bucket)
	if err != nil {
		log.Error().Msgf("Failed to delete TimeSeriesDB with name %v", bucketName)
		return
	}

	tscd.timeSeriesDB.Name = ""
	tscd.timeSeriesDB.RetentionPolicy = ""
	log.Info().Msgf("Sucessfully deleted TimeSeriesDB with name %v\n", bucketName)
	return
}

// Updates the database's retention policy
func (tscd *TimeSeriesClientData) UpdateTimeSeriesDBRetentionPolicy(newRetentionPolicy string) (err error) {
	bucketName := (*tscd).timeSeriesDB.Name
	bucketsAPI := (*tscd).iClient.BucketsAPI()
	bucket, err := bucketsAPI.FindBucketByName(context.Background(), bucketName)
	if bucket == nil {
		log.Error().Msgf("Failed to find TimeSeriesDB with name %v", bucketName)
		return
	}

	duration, err := rpStringToInt64(newRetentionPolicy)
	if err != nil {
		log.Error().Msgf("Failed to convert retention policy %v to duration with error: %v\n", newRetentionPolicy, err)
		return
	}

	bucket.RetentionRules[0].EverySeconds = duration

	// default shard group duration value
	var shardGroupDuration string
	if _60d, _ := rpStringToInt64("60d"); duration > _60d || duration == 0 {
		shardGroupDuration = "1w"
	} else if _2d, _ := rpStringToInt64("2d"); duration > _2d {
		shardGroupDuration = "1d"
	} else {
		shardGroupDuration = "1h"
	}

	shardGroupDurationSeconds, _ := rpStringToInt64(shardGroupDuration)
	bucket.RetentionRules[0].ShardGroupDurationSeconds = &shardGroupDurationSeconds
	_, err = bucketsAPI.UpdateBucket(context.Background(), bucket)
	if err != nil {
		log.Error().Msgf("Failed to updated TimeSeriesDB with name %v", bucketName)
		return
	}

	tscd.timeSeriesDB.RetentionPolicy = newRetentionPolicy
	log.Info().Msgf("Sucessfully updated TimeSeriesDB with name %v's retention policy to %vsec\n", bucketName, duration)
	return
}

// Deletes a table
func (tscd *TimeSeriesClientData) DropMeasurement(measurement string) (err error) {
	orgName := (*tscd).timeSeriesOrgName
	bucketName := (*tscd).timeSeriesDB.Name

	ctx := context.Background()
	startTime := tscd.timeSeriesDB.CreatedTime
	stopTime := time.Now()
	predicate := fmt.Sprintf("_measurement=%s", measurement)
	deleteAPI := (*tscd).iClient.DeleteAPI()

	err = deleteAPI.DeleteWithName(ctx, orgName, bucketName, startTime, stopTime, predicate)
	if err != nil {
		log.Error().Msgf("Failed to drop TimeSeriesDB's measurement with name %v", measurement)
	}

	log.Info().Msgf("Sucessfully drop %v's measurement with name %v\n", bucketName, measurement)
	return
}

// // Set operation to mimic traditional key-value pair setting.
// // PS - This creates new row than updating existing one to demonstrate time series capability
func (tscd *TimeSeriesClientData) Set(measurement, key string, value interface{}) (err error) {
	tags := map[string]string{}
	fields := map[string]interface{}{
		key: value,
	}
	return tscd.WritePoint(measurement, tags, fields)
}

// Get operation to mimic traditional key-value pair get operation
func (tscd *TimeSeriesClientData) Get(measurement, key string) (result interface{}, err error) {
	bucketName := tscd.timeSeriesDB.Name
	// Get query all data since DB created.
	startRange := time.Since(tscd.timeSeriesDB.CreatedTime).Truncate(time.Second) + (5 * time.Second)

	fluxQueryStr := fmt.Sprintf(`
	from(bucket: "%s")
    |> range(start: -%s)
    |> filter(fn: (r) => r._measurement == "%s" and r._field == "%s")
	|> last()
	`, bucketName, startRange, measurement, key)

	resp, err := tscd.Query(fluxQueryStr)
	if err == nil {
		for resp.Next() {
			result = resp.Record().Value()
			log.Debug().Msgf("value: %v\n", result)
		}
		if resp.Err() != nil {
			log.Error().Msgf("query parsing error: %s\n", resp.Err().Error())
		}
	} else {
		log.Error().Msgf("Unable to query data with error %v\n", err)
	}
	return result, nil
}

// Generic query operation wtih flux
func (tscd *TimeSeriesClientData) Query(fluxQueryStr string) (resp *api.QueryTableResult, err error) {
	orgName := (*tscd).timeSeriesOrgName

	queryAPI := (*tscd).iClient.QueryAPI(orgName)
	if queryAPI == nil {
		log.Error().Msgf("Failed to get queryAPI")
		return nil, errors.New("cannot get writeAPI")
	}

	resp, err = queryAPI.Query(context.Background(), fluxQueryStr)
	log.Info().Msgf("TimeSeriesDB Query: DB=%v, QueryString=%s, Result=%v, err=%v\n", tscd.timeSeriesDB.Name, fluxQueryStr, resp, err)
	return
}

// Generic write point operation. In influxDBv2, batch writing is implemented inside of writeAPI.WritePoint()
func (tscd *TimeSeriesClientData) WritePoint(measurement string, tags map[string]string, fields map[string]interface{}) (err error) {
	orgName := (*tscd).timeSeriesOrgName
	bucketName := (*tscd).timeSeriesDB.Name
	writeAPI := (*tscd).iClient.WriteAPI(orgName, bucketName)
	if writeAPI == nil {
		log.Error().Msgf("Failed to get writeAPI")
		return errors.New("cannot get writeAPI")
	}

	defer writeAPI.Flush()

	errorsCh := writeAPI.Errors()
	go func() {
		for err := range errorsCh {
			log.Error().Msgf("Failed to write with error: %v", err)
		}
	}()

	point := influxdb2.NewPoint(measurement,
		tags,
		fields,
		time.Now())
	writeAPI.WritePoint(point)
	log.Debug().Msgf("\nTimeSeriesDB WritePoint: DB=%v Measurement=%v tags=%v, fields=%v, err=%v", tscd.timeSeriesDB.Name, measurement, tags, fields, err)

	return nil
}

// Function to flatten nested json
func (tscd *TimeSeriesClientData) Flatten(nested map[string]interface{}, prefix string, IgnoreKeyList []string) (flatmap map[string]interface{}, err error) {
	flatmap = make(map[string]interface{})

	err = _flatten(true, flatmap, nested, prefix, IgnoreKeyList)
	if err != nil {
		return
	}

	return
}

// Insert 1 or more Json Rows
func (tscd *TimeSeriesClientData) InsertUnmarshalledJsonRows(measurement string, rows []JsonRow, ignoreKeyList []string) (err error) {
	tags := make(map[string]string)
	fields := make(map[string]interface{})

	for _, data := range rows {
		flatjson, err := tscd.Flatten(data, "", ignoreKeyList)
		if err != nil {
			log.Warn().Msgf("\n Not able to flatten json %s for:%v", err.Error(), data)
		}

		log.Info().Msgf("\n Data after flattening: %v", flatjson)

		for key, value := range flatjson {
			if value != nil {
				if reflect.ValueOf(value).Type().Kind() == reflect.Float64 {
					fields[key] = value
				} else if reflect.ValueOf(value).Type().Kind() == reflect.String {
					fields[key] = value
				} else if reflect.ValueOf(value).Type().Kind() == reflect.Bool {
					fields[key] = value
				} else if reflect.ValueOf(value).Type().Kind() == reflect.Int {
					fields[key] = value
				}
			}
		}
		err = tscd.WritePoint(measurement, tags, fields)
		if err != nil {
			log.Error().Msgf("Failed to InsertUnmarshalledJsonRows cause error : %v", err)
		}
	}
	return
}

// Function to flatten array of nested json
func (tscd *TimeSeriesClientData) UnmarshallJsonRows(jsonBuffer []byte) (jsonrow []JsonRow, err error) {

	// We create an empty array
	jsonrow = []JsonRow{}

	// Unmarshal the json into it. this will use the struct tag
	err = json.Unmarshal(jsonBuffer, &jsonrow)

	// the array is now filled with each row of json as an array index
	return
}

// Inserts JSON rows as separate time points in the mentioned measurement
func (tscd *TimeSeriesClientData) InsertJsonArray(measurement string, ignoreList []string, jsonBuffer []byte) (err error) {
	rows, err := tscd.UnmarshallJsonRows(jsonBuffer)
	if err == nil && len(rows) > 0 {
		// We can call InsertUnmarshalledJsonRow but it will do write for each row
		// Instead, use batching if rows more than 1
		err = tscd.InsertUnmarshalledJsonRows(measurement, rows, ignoreList)
	}
	return err
}

// Inserts json data as single row in the mentioned meausrement
// PS - Use only for single row data
func (tscd *TimeSeriesClientData) InsertJson(measurement string, ignoreList []string, jsonBuffer []byte) (err error) {
	tags := make(map[string]string)
	fields := make(map[string]interface{})
	data := make(map[string]interface{})

	err = json.Unmarshal(jsonBuffer, &data)
	if err != nil {
		log.Error().Msgf("\n Not able to Parse data %s", err.Error())
		return err
	}

	flatjson, err := tscd.Flatten(data, "", ignoreList)
	if err != nil {
		log.Error().Msgf("\n Not able to flatten json %s for:%v", err.Error(), data)
		return err
	}

	log.Info().Msgf("\n Data after flattening: %v", flatjson)

	for key, value := range flatjson {
		if value != nil {
			if reflect.ValueOf(value).Type().Kind() == reflect.Float64 {
				fields[key] = value
			} else if reflect.ValueOf(value).Type().Kind() == reflect.String {
				fields[key] = value
			} else if reflect.ValueOf(value).Type().Kind() == reflect.Bool {
				fields[key] = value
			} else if reflect.ValueOf(value).Type().Kind() == reflect.Int {
				fields[key] = value
			}
		}
	}

	return tscd.WritePoint(measurement, tags, fields)
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
//	Generic functions - Non methods
//
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
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

	switch reflect.TypeOf(nested).Kind() {
	case reflect.Map:
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
	case reflect.Slice:
		for i, v := range nested.([]interface{}) {
			switch reflect.TypeOf(v).Kind() {
			case reflect.Map:
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
		return errors.New("not a valid input: map or slice")
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

func rpInt64ToString(duration int64) string {
	if duration == 0 {
		return ""
	}

	type timeUnit struct {
		unit  byte
		asSec int64
	}

	wdhms := [5]timeUnit{
		{'w', 7 * 24 * 60 * 60},
		{'d', 24 * 60 * 60},
		{'h', 60 * 60},
		{'m', 60},
		{'s', 1},
	}

	var buf strings.Builder

	for _, tu := range wdhms {
		p := duration / tu.asSec
		duration = duration % tu.asSec
		if p != 0 {
			buf.WriteString(strconv.FormatInt(p, 10))
			buf.WriteByte(tu.unit)
		}
	}

	return buf.String()
}

func rpStringToInt64(retentionPolicy string) (duration int64, err error) {
	if retentionPolicy == "" {
		return 0, nil
	}
	var buf strings.Builder
	for _, c := range retentionPolicy {
		if c < '0' || c > '9' {
			val, _ := strconv.ParseInt(buf.String(), 10, 64)
			switch c {
			case 'w':
				duration += val * 7 * 24 * 60 * 60
			case 'd':
				duration += val * 24 * 60 * 60
			case 'h':
				duration += val * 60 * 60
			case 'm':
				duration += val * 60
			case 's':
				duration += val
			default:
				return 0, errors.New("unit of retention policy time duration supports only 'w', 'd', 'h', 'm', 's'")
			}
			buf.Reset()
		} else {
			buf.WriteRune(c)
		}
	}
	return
}
