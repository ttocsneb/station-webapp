package database

import (
	"database/sql"
	"fmt"
	"time"
)

var DB *sql.DB = nil

func genStringJoins(table string, properties ...string) string {
	joins := ""
	for _, property := range properties {
		joins += fmt.Sprintf(
			"JOIN %[1]v %[2]v ON %[3]v.%[2]v_id = %[2]v.id\n",
			LOOKUP_STRINGS,
			property, table,
		)
	}
	return joins
}

func getReduced(db *sql.DB) (time.Time, error) {
	// Get the reduced value from db_info, or the earliest time in condition_entry
	row := db.QueryRow(`SELECT MAX(time) as time FROM (
		SELECT reduced as time FROM db_info WHERE ID = 1
		UNION
		SELECT MIN(time) FROM condition_entry);`)
	var val string
	err := row.Scan(&val)
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.Parse("2006-01-02 15:04:05.999999999Z07:00", val)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func setReduced(db *sql.DB, t time.Time) error {
	_, err := db.Exec(`UPDATE db_info SET reduced = ? WHERE ID = 1;`, t)
	return err
}
