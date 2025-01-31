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
	SubmitionStarted    bool
}

type Worker struct {
	Address    string
	Registered bool
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

	createTables := `
    CREATE TABLE IF NOT EXISTS threads (
        id TEXT PRIMARY KEY,
		work_started BOOLEAN,
		work_completed BOOLEAN,
		solution_proposed BOOLEAN,
		verification_started BOOLEAN,
		submition_started BOOLEAN
	);
	CREATE TABLE IF NOT EXISTS workers (
		address TEXT PRIMARY KEY,
		registered BOOLEAN
	);
    `

	if _, err := db.Exec(createTables); err != nil {
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
	insertQuery := `INSERT INTO threads (id, work_started, work_completed, solution_proposed, verification_started, submition_started) VALUES (?, false, false, false, false, false)`
	_, err := db.conn.Exec(insertQuery, id)
	if err != nil {
		return fmt.Errorf("failed to insert thread: %w", err)
	}

	return nil
}

// Readthread retrieves a thread by ID.
func (db *DB) ReadThread(id string) (*Thread, error) {
	query := `SELECT id, work_started, work_completed, solution_proposed, verification_started, submition_started  FROM threads WHERE id = ?`
	row := db.conn.QueryRow(query, id)

	var thread Thread
	if err := row.Scan(&thread.ID, &thread.WorkStarted, &thread.WorkCompleted, &thread.SolutionProposed, &thread.VerificationStarted, &thread.SubmitionStarted); err != nil {
		if err == sql.ErrNoRows {
			// thead doesn't exists, so we insert it
			db.AddThread(id)
			return &Thread{ID: id, WorkStarted: false, WorkCompleted: false, SolutionProposed: false, VerificationStarted: false, SubmitionStarted: false}, nil
		}
		return nil, fmt.Errorf("failed to read thread: %w", err)
	}

	return &thread, nil
}

// Updatethread updates a task's information.
func (db *DB) UpdateThread(id string, workStarted, workCompleted, solProposed, verificationStarted bool, submitionStarted bool) error {
	updateQuery := `UPDATE threads SET work_started = ?, work_completed = ?, solution_proposed = ?, verification_started = ? , submition_started = ? WHERE id = ?`
	_, err := db.conn.Exec(updateQuery, workStarted, workCompleted, solProposed, verificationStarted, submitionStarted, id)
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

// Createthread inserts a new thread into the database.
func (db *DB) Addworker(address string) error {
	insertQuery := `INSERT INTO workers (address, registered) VALUES (?, true)`
	_, err := db.conn.Exec(insertQuery, address)
	if err != nil {
		return fmt.Errorf("failed to insert worker: %w", err)
	}

	return nil
}

// Readthread retrieves a thread by ID.
func (db *DB) IsWorkerRegistered(address string) (bool, error) {
	query := `SELECT address, registered  FROM workers WHERE address = ?`
	row := db.conn.QueryRow(query, address)

	var worker Worker
	if err := row.Scan(&worker.Address, &worker.Registered); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to read thread: %w", err)
	}

	return true, nil
}

func (db *DB) DeleteWorker(address string) error {
	deleteQuery := `DELETE FROM workers WHERE address = ?`
	_, err := db.conn.Exec(deleteQuery, address)
	if err != nil {
		return fmt.Errorf("failed to delete worker: %w", err)
	}
	return nil
}
