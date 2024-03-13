package database

import (
	"database/sql"
	"fmt"
	"strings"
)

const LOOKUP_STRINGS string = "lookup_strings"

func FetchLookupStrings(db *sql.DB, strs []string) (map[string]int, error) {
	placeholders := make([]string, len(strs))
	args := make([]any, len(strs))
	for i, str := range strs {
		placeholders[i] = "?"
		args[i] = str
	}
	query := fmt.Sprintf(
		"SELECT * FROM %v WHERE value IN (%v);",
		LOOKUP_STRINGS,
		strings.Join(placeholders, ", "),
	)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	strings := make(map[string]int)
	for rows.Next() {
		var id int
		var value string
		if err := rows.Scan(&id, &value); err != nil {
			return nil, err
		}
		strings[value] = id
	}

	return strings, nil
}

func InsertLookupStrings(db *sql.DB, strs []string) error {
	placeholders := make([]string, len(strs))
	args := make([]any, len(strs))
	for i, str := range strs {
		placeholders[i] = "(?)"
		args[i] = str
	}
	query := fmt.Sprintf(
		"INSERT INTO %v (value) VALUES %v;",
		LOOKUP_STRINGS,
		strings.Join(placeholders, ", "),
	)

	_, err := db.Exec(query, args...)
	return err
}

func GetOrInsertLookupStrings(db *sql.DB, strs []string) (map[string]int, error) {
	found, err := FetchLookupStrings(db, strs)
	if err != nil {
		return nil, err
	}

	to_create := []string{}
	for _, str := range strs {
		if _, exists := found[str]; !exists {
			to_create = append(to_create, str)
		}
	}

	if len(to_create) == 0 {
		return found, nil
	}

	if err := InsertLookupStrings(db, to_create); err != nil {
		return nil, err
	}
	created, err := FetchLookupStrings(db, to_create)
	if err != nil {
		return nil, err
	}
	for key, id := range created {
		found[key] = id
	}
	return found, nil
}
