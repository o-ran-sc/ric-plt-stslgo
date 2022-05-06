# Wrapper GO module to access the TimeSeriesDB (Currently Influx Db is used, but can be extended to other DBs in future). 
TimeSeriesDB means InfluxDb as of now for all purposes
This GO module uses the well defined v1.x GO client library SDK provided by the TimeSeriesDB team.
This provides APIs for mostly commonly used functionalities and makes accessing TimeSeriesDB easier. 
But, if needed, XAPP or any other pod can even use the v1.x GO client library SDK directly too.

## APIs
| API                                     | Description                                                                                                                                                                            |
|-----------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
|NewTimeSeriesClientData()                    | Constructor for type TimeSeriesClientData which is used to store connection to timeseriesDB, DB name, username and password.
|
|CreateTimeSeriesConnection()                 | Creates a connection to TimeSeriesDB.
|
|CreateTimeSeriesDB()                         | Creates the DB specified during the constructor of TimeSeriesClientData.
|
|CreateTimeSeriesDBWithRetentionPolicy()      | Creates the DB specified during the constructor of TimeSeriesClientData along with the new retention policy set as default for this database.
|
|DeleteTimeSeriesDB()                         | Deletes the DB specified during the constructor of TimeSeriesClientData.
|
|DropMeasurement()                        | Deletes the measurement specified as an arguement.
|
|CreateRetentionPolicy()                  | Creates a retention policy for a database.
|
|UpdateRetentionPolicy()                  | Updates the retention policy of a database.
|
|DeleteRetentionPolicy()                  | Deletes the retention policy of a database.
|
|Set()                                    | Mimics the traditional set operation of key-value pair. Inserts key-value pair into fieldset of TimeSeriesDB.
|
|Get()                                    | Mimics the traditional get operation of key-value pair. Gets the latest by time value of given key.
|
|Query()                                  | Generic query API for querying the TimeSeriesDB. Return type is Response structure of TimeSeriesDB GO library.
|
|WritePoint()                             | Generic write API to write a set of tags & fields to mentioned measurement/table in TimeSeriesDB.
|
|InsertJson()                             | Use to insert JSON object in mentioned measurement/table.
|
|InsertJsonArray()                        | Use to insert JSON array as individual rows in mentioned measurement/table. To be used only when top level JSON has array and not when array is nested inside one existing JSON. Eg. Not to be used for UeMetrics with multiple neighbor cells.
|
|Flatten()                                | Generic API to flatten JSON data. This will handle nested JSON as well and split it into individual columns.
|
