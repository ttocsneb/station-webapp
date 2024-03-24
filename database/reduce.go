package database

import (
	"database/sql"
	"math"
	"time"
)

func value(_ int, val float64) float64 {
	return val
}

func min_sensor(conditions []Condition, key string) (int, float64) {
	min_value := conditions[0].Sensors[key]
	index := 0
	for i, cond := range conditions {
		val := cond.Sensors[key]
		if val < min_value {
			min_value = val
			index = i
		}
	}
	return index, min_value
}

func max_sensor(conditions []Condition, key string) (int, float64) {
	max_value := conditions[0].Sensors[key]
	index := 0
	for i, cond := range conditions {
		val := cond.Sensors[key]
		if val > max_value {
			max_value = val
			index = i
		}
	}
	return index, max_value
}

func average(pairs []Pair) float64 {
	t0 := pairs[0].Time

	time_sum := 0.0
	value_sum := 0.0

	for _, pair := range pairs {
		t := pair.Time.Sub(t0).Abs().Seconds()
		time_sum += t
		value_sum += t * pair.Value
	}

	return value_sum / time_sum
}

func averageAngles(pairs []Pair) float64 {
	xs := make([]Pair, len(pairs))
	ys := make([]Pair, len(pairs))
	for i, pair := range pairs {
		rad := pair.Value * math.Pi / 180
		xs[i] = Pair{
			Time:  pair.Time,
			Value: math.Cos(rad),
		}
		ys[i] = Pair{
			Time:  pair.Time,
			Value: math.Sin(rad),
		}
	}

	x := average(xs)
	y := average(ys)
	return math.Atan2(y, x)
}

var averageMap = map[string]AveragingFunc{
	"winddir":        averageAngles,
	"winddir-avg2m":  averageAngles,
	"winddir-avg10m": averageAngles,
	"windgustdir-2m": averageAngles,
}

func getAverager(name string) AveragingFunc {
	if name == "winddir" || name == "winddir-avg2m" || name == "winddir-avg10m" ||
		name == "windgustdir-2m" {
		return averageAngles
	}
	return average
}

func reduceConditionsRange(db *sql.DB, begin time.Time, end time.Time) (int, error) {
	conditions, err := FetchConditions(db, `WHERE time BETWEEN ? AND ? ORDER BY time`, begin, end)
	if err != nil {
		return 0, err
	}
	if len(conditions) <= 1 {
		return len(conditions), nil
	}

	// Average out all the fields, then get the min/max of each noteworthy field
	new_condition := AverageConditions(conditions, conditions[len(conditions)-1].Time, getAverager)
	new_condition.Sensors["temp-min"] = value(min_sensor(conditions, "temp"))
	new_condition.Sensors["temp-max"] = value(max_sensor(conditions, "temp"))
	new_condition.Sensors["dewpoint-min"] = value(min_sensor(conditions, "dewpoint"))
	new_condition.Sensors["dewpoint-max"] = value(max_sensor(conditions, "dewpoint"))
	new_condition.Sensors["humidity-min"] = value(min_sensor(conditions, "humidity"))
	new_condition.Sensors["humidity-max"] = value(max_sensor(conditions, "humidity"))
	new_condition.Sensors["barom-min"] = value(min_sensor(conditions, "barom"))
	new_condition.Sensors["barom-max"] = value(max_sensor(conditions, "barom"))
	new_condition.Sensors["uv-min"] = value(min_sensor(conditions, "uv"))
	new_condition.Sensors["uv-max"] = value(max_sensor(conditions, "uv"))
	new_condition.Sensors["dailyrain"] = value(max_sensor(conditions, "dailyrain"))
	i, val := max_sensor(conditions, "windgustspd-2m")
	new_condition.Sensors["windgustspd-2m"] = val
	new_condition.Sensors["windgustdir-2m"] = conditions[i].Sensors["windgustdir-2m"]

	err = new_condition.InsertDb(db)
	if err != nil {
		return 0, err
	}

	err = DeleteConditions(db, conditions)
	if err != nil {
		return 0, err
	}

	return len(conditions), nil
}

func ReduceConditions(db *sql.DB) error {
	year, month, day := time.Now().Local().Date()
	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.Local)

	stop_time, err := getReduced(db)
	if err != nil {
		return err
	}

	// Past one week, there should only be one sample per hour.
	end_time := midnight.Add(-time.Hour * 24 * 7)
	new_reduced := end_time
	for end_time.Compare(stop_time) > 0 {
		start_time := end_time.Add(-time.Hour)
		_, err := reduceConditionsRange(db, start_time, end_time)
		if err != nil {
			return err
		}
		end_time = start_time
	}

	err = setReduced(db, new_reduced)

	return err
}

func IsTimeToReduce(db *sql.DB) (bool, error) {
	t, err := getReduced(db)
	if err != nil {
		return false, err
	}
	year, month, day := t.Date()
	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	to_reduce := midnight.Add(time.Hour * 24 * 8)

	now := time.Now().Local()

	return to_reduce.Before(now), nil

}

// temp: C
// dewpoint: C
// humidity: %
// barom: hPa
// dailyrain: in
// rain-1h: in
// uv: UV Index
// windspd: km/h
// winddir: deg
// windspd-avg2m: km/h
// winddir-avg2m: deg
// windspd-avg10m: km/h
// winddir-avg10m: deg
// windgustspd-2m: km/h
// windgustdir-2m: deg
