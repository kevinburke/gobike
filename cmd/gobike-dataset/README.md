# Usage

## Generate the CSV

```
gobike-dataset path/to/data/ /path/to/trips.csv
```

## Postgres

```sql
CREATE TABLE trips (
  duration INTEGER,
  start_time TIMESTAMP,
  end_time  TIMESTAMP,

  start_station_id        INTEGER,
  start_station_name      TEXT,
  start_station_latitude  FLOAT,
  start_station_longitude FLOAT,
  start_station_city      TEXT,

  end_station_id        INTEGER,
  end_station_name      TEXT,
  end_station_latitude  FLOAT,
  end_station_longitude FLOAT,
  end_station_city      TEXT,

  bike_id INTEGER,
  user_type TEXT,
  member_birth_year SMALLINT,
  member_gender TEXT,
  bike_share_for_all BOOLEAN
)
```

```sql
COPY trips FROM '/path/to/trips.csv' CSV HEADER;
```


