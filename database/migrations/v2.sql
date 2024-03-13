CREATE TABLE db_info (
    id INTEGER PRIMARY KEY,
    version INTEGER,
    reduced DATETIME
);

CREATE TABLE lookup_strings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    value TEXT 
);

CREATE TABLE condition_entry (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    time DATETIME
);

CREATE TABLE sensor_value (
    entry_id INTEGER,
    name_id INTEGER,
    value FLOAT,
    PRIMARY KEY (entry_id, name_id),
    CONSTRAINT FK_entry FOREIGN KEY (entry_id) REFERENCES condition_entry(id),
    CONSTRAINT FK_name FOREIGN KEY (name_id) REFERENCES condition_entry(id)
);

INSERT INTO db_info (id, version)
    VALUES (1, 2);
