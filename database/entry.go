package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Condition struct {
	Id      int
	Time    time.Time
	Sensors map[string]float64
}

func NewCondition(time time.Time) Condition {
	return Condition{
		Time:    time,
		Sensors: make(map[string]float64),
	}
}

func (self *Condition) InsertDb(db *sql.DB) error {
	string_list := []string{}
	for key := range self.Sensors {
		string_list = append(string_list, key)
	}

	lookup, err := GetOrInsertLookupStrings(db, string_list)
	if err != nil {
		return err
	}

	query := `INSERT INTO condition_entry (time) VALUES (?);`
	result, err := db.Exec(query, self.Time)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	self.Id = int(id)

	entries := []string{}
	args := []any{}
	for name, value := range self.Sensors {
		entries = append(entries, "(?, ?, ?)")
		args = append(args, id, lookup[name], value)
	}

	query = fmt.Sprintf(
		`INSERT INTO sensor_value 
			(entry_id, name_id, value)
		VALUES 
			%v;`,
		strings.Join(entries, ",\n"),
	)
	_, err = db.Exec(query, args...)

	return err
}

func fetchSensorsFromEntry(db *sql.DB, id int) (map[string]float64, error) {
	query := fmt.Sprintf(
		`SELECT name.value, sensor_value.value
		FROM sensor_value
		%v
		WHERE entry_id = ?
		ORDER BY name_id ASC`,
		genStringJoins("sensor_value", "name"),
	)

	rows, err := db.Query(query, id)
	if err != nil {
		return nil, err
	}

	sensors := make(map[string]float64)

	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		sensors[name] = value
	}

	return sensors, nil
}

func FetchCondition(db *sql.DB, condition string, arg ...any) (Condition, error) {
	query := fmt.Sprintf(
		`SELECT condition_entry.id, time FROM condition_entry %v LIMIT 1;`,
		condition,
	)
	row := db.QueryRow(query, arg...)
	var id int
	var time time.Time
	err := row.Scan(&id, &time)
	if err != nil {
		return Condition{}, err
	}

	sensors, err := fetchSensorsFromEntry(db, id)
	if err != nil {
		return Condition{}, err
	}

	return Condition{
		Id:      id,
		Time:    time,
		Sensors: sensors,
	}, nil
}

func FetchConditions(db *sql.DB, condition string, args ...any) ([]Condition, error) {
	query := fmt.Sprintf(
		`SELECT id, time FROM condition_entry %v;`,
		condition,
	)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	entries := []Condition{}
	for rows.Next() {
		var id int
		var time time.Time
		err := rows.Scan(&id, &time)
		if err != nil {
			return nil, err
		}

		sensors, err := fetchSensorsFromEntry(db, id)
		if err != nil {
			return nil, err
		}

		entries = append(entries, Condition{
			Id:      id,
			Time:    time,
			Sensors: sensors,
		})
	}

	return entries, nil
}

func FetchLatestCondition(db *sql.DB) (Condition, error) {
	return FetchCondition(db, "ORDER BY time DESC")
}

type Pair struct {
	Time  time.Time
	Value float64
}
type AveragingFunc func([]Pair) float64

func AverageConditions(
	conditions []Condition,
	new_time time.Time,
	average func(string) AveragingFunc) Condition {
	sensors := make(map[string][]Pair)

	for _, condition := range conditions {
		for name, value := range condition.Sensors {
			pairs, exists := sensors[name]
			if !exists {
				pairs = []Pair{}
			}
			pairs = append(pairs, Pair{
				Time:  condition.Time,
				Value: value,
			})

			sensors[name] = pairs
		}
	}

	averaged := Condition{
		Time:    new_time,
		Sensors: make(map[string]float64),
	}

	for name, pairs := range sensors {
		averager := average(name)
		averaged.Sensors[name] = averager(pairs)
	}

	return averaged
}

func (self *Condition) DeleteDB(db *sql.DB) error {
	query := `DELETE FROM sensor_value WHERE entry_id = ?;`
	_, err := db.Exec(query, self.Id)
	if err != nil {
		return err
	}

	query = `DELETE FROM condition_entry WHERE id = ?;`
	_, err = db.Exec(query, self.Id)
	if err != nil {
		return err
	}

	return nil
}

func DeleteConditions(db *sql.DB, conditions []Condition) error {
	args := make([]any, len(conditions))
	for i, condition := range conditions {
		args[i] = condition.Id
	}
	query := fmt.Sprintf(
		`DELETE FROM sensor_value WHERE entry_id IN (%v);`,
		"?"+strings.Repeat(", ?", len(conditions)-1),
	)
	_, err := db.Exec(query, args...)
	if err != nil {
		return err
	}
	query = fmt.Sprintf(
		`DELETE FROM condition_entry WHERE id IN (%v);`,
		"?"+strings.Repeat(", ?", len(conditions)-1),
	)
	_, err = db.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}
