package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// thread represents a video rendering task.
type Thread struct {
	ID                  string
	WorkStarted         bool
	WorkCompleted       bool
	SolutionProposed    bool
	VerificationStarted bool
}

// DB encapsulates the database connection.
type DB struct {
	conn *sql.DB
}

// Init initializes the SQLite database and creates the threads table.
func Init(databasePath string) (*DB, error) {
	// if the path doesn't exists, it might be that client wasn't yet initialized, so we don't create it
	_, err := os.Stat(databasePath)
	if errors.Is(err, fs.ErrNotExist) {
		return &DB{}, nil
	}

	db, err := sql.Open("sqlite3", filepath.Join(databasePath, "videoRendering.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	createTable := `
    CREATE TABLE IF NOT EXISTS threads (
        id TEXT PRIMARY KEY,
		work_started BOOLEAN,
		work_completed BOOLEAN,
		solution_proposed BOOLEAN,
		verification_started BOOLEAN
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
	insertQuery := `INSERT INTO threads (id, work_started, work_completed, solution_proposed, verification_started) VALUES (?, false, false, false, false)`
	_, err := db.conn.Exec(insertQuery, id)
	if err != nil {
		return fmt.Errorf("failed to insert thread: %w", err)
	}

	return nil
}

// Readthread retrieves a thread by ID.
func (db *DB) ReadThread(id string) (*Thread, error) {
	query := `SELECT id, work_started, work_completed, solution_proposed, verification_started  FROM threads WHERE id = ?`
	row := db.conn.QueryRow(query, id)

	var thread Thread
	if err := row.Scan(&thread.ID, &thread.WorkStarted, &thread.WorkCompleted, &thread.SolutionProposed, &thread.VerificationStarted); err != nil {
		if err == sql.ErrNoRows {
			// thead doesn't exists, so we insert it
			db.AddThread(id)
			return &Thread{ID: id, WorkStarted: false, WorkCompleted: false, SolutionProposed: false, VerificationStarted: false}, nil
		}
		return nil, fmt.Errorf("failed to read thread: %w", err)
	}

	return &thread, nil
}

// Updatethread updates a task's information.
func (db *DB) UpdateThread(id string, workStarted, workCompleted, solProposed, verificationStarted bool) error {
	updateQuery := `UPDATE threads SET work_started = ?, work_completed = ?, solution_proposed = ?, verification_started = ? WHERE id = ?`
	_, err := db.conn.Exec(updateQuery, workStarted, workCompleted, solProposed, verificationStarted, id)
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
