# Datasets

In addition to server, we've made the Ford GoBike datasets available online.

## BigQuery

All trip data lives in the
[ford_gobike](https://bigquery.cloud.google.com/table/kjc-datasets:ford_gobike.trips)
dataset, which is availably publicly.

### Trips per week

```sql
SELECT
  DATE_TRUNC(DATE(start_time), WEEK) as week,
  COUNT(*) as trips
FROM `bay-area-public-data.ford_gobike.trips`
GROUP BY 1
ORDER BY 1
```

### Unique bikes per week

```sql
SELECT
  DATE_TRUNC(DATE(start_time), WEEK) as week,
  COUNT(distinct bike_id) as bikes
FROM `bay-area-public-data.ford_gobike.trips`
GROUP BY 1
ORDER BY 1
```

### Average trips per bike per week

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
