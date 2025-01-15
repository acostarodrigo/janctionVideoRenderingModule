package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// thread represents a video rendering task.
type Thread struct {
	ID                  string
	WorkStarted         bool
	SolutionProposed    bool
	VerificationStarted bool
}

// DB encapsulates the database connection.
type DB struct {
	conn *sql.DB
}

// Init initializes the SQLite database and creates the threads table.
func Init(databasePath string) (*DB, error) {
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	createTable := `
    CREATE TABLE IF NOT EXISTS threads (
        id TEXT PRIMARY KEY,
		work_started BOOL,
		solution_proposed BOOL,
		verification_started BOOL
    );`

	if _, err := db.Exec(createTable); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &DB{conn: db}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Createthread inserts a new thread into the database.
func (db *DB) AddThread(id string) error {
	insertQuery := `INSERT INTO threads (id, work_started, solution_proposed, verification_started) VALUES (?, false, false, false)`
	_, err := db.conn.Exec(insertQuery, id)
	if err != nil {
		return fmt.Errorf("failed to insert thread: %w", err)
	}

	return nil
}

// Readthread retrieves a thread by ID.
func (db *DB) ReadThread(id string) (*Thread, error) {
	query := `SELECT * FROM threads WHERE id = ?`
	row := db.conn.QueryRow(query, id)

	var thread Thread
	if err := row.Scan(&thread.ID, &thread.WorkStarted, &thread.SolutionProposed, &thread.VerificationStarted); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read thread: %w", err)
	}

	return &thread, nil
}

// Updatethread updates a task's information.
func (db *DB) UpdateThread(id string, workStarted, solProposed, verificationStarted bool) error {
	updateQuery := `UPDATE threads SET work_started = ?, solution_proposed = ?, verification_started = ? WHERE id = ?`
	_, err := db.conn.Exec(updateQuery, id, workStarted, solProposed, verificationStarted)
	if err != nil {
		return fmt.Errorf("failed to update thread: %w", err)
	}
	return nil
}

// Deletethread deletes a thread by ID.
func (db *DB) DeleteThread(id string) error {
	deleteQuery := `DELETE FROM threads WHERE id = ?`
	_, err := db.conn.Exec(deleteQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}
	return nil
}
