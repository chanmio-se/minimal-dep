package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"minimal-dep/internal/modules/tasks/domain"
)

var fakeState struct {
	mu   sync.Mutex
	done bool
}

func TestSQLRepository(t *testing.T) {
	sql.Register("tasks-repository-test", fakeDriver{})
	db, err := sql.Open("tasks-repository-test", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repo := NewSQL(db)
	ctx := context.Background()

	task, err := domain.NewTask("t1", "u1", "buy milk")
	if err != nil {
		t.Fatal(err)
	}
	task.CreatedAt = time.Now().UTC()
	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tasks, err := repo.ListByOwner(ctx, "u1")
	if err != nil || len(tasks) != 1 || tasks[0].Title != "buy milk" {
		t.Fatalf("ListByOwner() = %+v, %v", tasks, err)
	}

	found, err := repo.FindByID(ctx, "t1")
	if err != nil || found.ID != "t1" {
		t.Fatalf("FindByID() = %+v, %v", found, err)
	}
	if err := repo.Save(ctx, found.Complete()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	found, err = repo.FindByID(ctx, "t1")
	if err != nil || !found.Done {
		t.Fatalf("FindByID() after Save() = %+v, %v", found, err)
	}
}

// fakeDriver is the smallest database/sql driver needed to exercise the SQL
// repository queries without adding an external dependency.
type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return &fakeConn{}, nil }

type fakeConn struct{}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeStmt{query: query}, nil
}
func (c *fakeConn) Close() error              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error) { return fakeTx{}, nil }

type fakeTx struct{}

func (fakeTx) Commit() error   { return nil }
func (fakeTx) Rollback() error { return nil }

type fakeStmt struct{ query string }

func (s *fakeStmt) Close() error  { return nil }
func (s *fakeStmt) NumInput() int { return -1 }
func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	if strings.Contains(s.query, "UPDATE tasks") {
		done, _ := args[1].(bool)
		fakeState.mu.Lock()
		fakeState.done = done
		fakeState.mu.Unlock()
	}
	return driver.RowsAffected(1), nil
}
func (s *fakeStmt) Query([]driver.Value) (driver.Rows, error) {
	fakeState.mu.Lock()
	done := fakeState.done
	fakeState.mu.Unlock()
	value := int64(0)
	if done {
		value = 1
	}
	rows := &fakeRows{columns: []string{"id", "owner_id", "title", "done", "created_at"}}
	switch {
	case strings.Contains(s.query, "WHERE owner_id"):
		rows.data = [][]driver.Value{{"t1", "u1", "buy milk", value, time.Now()}}
	case strings.Contains(s.query, "WHERE id"):
		rows.data = [][]driver.Value{{"t1", "u1", "buy milk", value, time.Now()}}
	}
	return rows, nil
}

type fakeRows struct {
	columns []string
	data    [][]driver.Value
	index   int
}

func (r *fakeRows) Columns() []string { return r.columns }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.index >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.index])
	r.index++
	return nil
}
