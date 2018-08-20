# GoBike Data

This project visualizes data about the Ford GoBike network. The data is
available here: https://bikeshare.science.

## Trip Data

Trip data is downloaded from https://www.fordgobike.com/system-data and should
be placed in the `data` directory. Once downloaded, the directory should look
like this:

```
$ ll data
total 1309768
drwxr-xr-x  11 kevin  staff   352B Aug 14 01:08 .
drwxr-xr-x  22 kevin  staff   704B Aug 19 20:09 ..
-rw-r--r--@  1 kevin  staff   112M Aug 12 02:44 2017-fordgobike-tripdata.csv
-rw-r--r--@  1 kevin  staff    19M Aug 12 02:44 201801-fordgobike-tripdata.csv
-rw-r--r--@  1 kevin  staff    22M Aug 12 02:44 201802-fordgobike-tripdata.csv
-rw-r--r--@  1 kevin  staff    23M Aug 12 02:44 201803-fordgobike-tripdata.csv
-rw-r--r--@  1 kevin  staff    27M Aug 12 02:44 201804-fordgobike-tripdata.csv
-rw-r--r--@  1 kevin  staff    36M Jun  8 08:08 201805-fordgobike-tripdata.csv
-rw-r--r--@  1 kevin  staff    40M Jul 16 11:40 201806-fordgobike-tripdata.csv
-rw-r--r--@  1 kevin  staff    40M Aug  7 12:01 201807-fordgobike-tripdata.csv
```

This is a prerequisite for building the site.

## Static Site

All of the pages are static pages that are checked in to Git. Run `make site` to
regenerate the HTML pages.

## Testing

Run `make test` to run the test suite.

## Polygons

The polygons are kind of a pain. Use `geojsonlint` to check whether your
polygons are okay. They need to be in a particular order.

Run the `rewind` script to rewind the polygon order.

### Datasets

In addition to server, we've made the Ford GoBike datasets available online.

#### BigQuery

All trip data lives in the
[ford_gobike](https://bigquery.cloud.google.com/table/kjc-datasets:ford_gobike.trips)
dataset, which is available publicly.

##### Trips per week

```sql
SELECT
  DATE_TRUNC(DATE(start_time), WEEK) as week,
  COUNT(*) as trips
FROM `bay-area-public-data.ford_gobike.trips`
GROUP BY 1
ORDER BY 1
```

##### Unique bikes per week

```sql
SELECT
  DATE_TRUNC(DATE(start_time), WEEK) as week,
  COUNT(distinct bike_id) as bikes
FROM `bay-area-public-data.ford_gobike.trips`
GROUP BY 1
ORDER BY 1
```

##### Average trips per bike per week

```sql
WITH bike_trips AS (
SELECT
  DATE_TRUNC(DATE(start_time), WEEK) as week,
  bike_id,
  count(*) as trips
FROM `bay-area-public-data.ford_gobike.trips`
GROUP BY 1, 2
ORDER BY 1
)

SELECT week, avg(trips)
FROM bike_trips
GROUP BY 1
```
