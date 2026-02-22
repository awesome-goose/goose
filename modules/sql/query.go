package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/props"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrNotFound is returned when a record is not found
var ErrNotFound = errors.New("record not found")

// validIdentifier matches safe SQL identifiers (alphanumeric, underscores, and dots)
var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)*$`)

type Query struct {
	db  *Db       `inject:""`
	log types.Log `inject:""`
}

func (r *Query) With(db *Db) *Query {
	clone := *r
	clone.db = db
	return &clone
}

// Tx executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
func (r *Query) Tx(fn func(*Query) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&Query{db: &Db{DB: tx, log: r.db.log}, log: r.log})
	})
}

// Exists checks if any record matches the query
func (r *Query) Exists(table string, query any, args ...any) (bool, error) {
	var count int64
	err := r.db.Table(table).Where(query, args...).Limit(1).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Count returns the number of records matching the query
func (r *Query) Count(table string, query any, args ...any) (int64, error) {
	var count int64
	err := r.db.Table(table).Where(query, args...).Count(&count).Error
	return count, err
}

// Sum returns the sum of a column for records matching the query
func (r *Query) Sum(table string, column string, query any, args ...any) (int64, error) {
	if !validIdentifier.MatchString(column) {
		return 0, fmt.Errorf("invalid column name: %s", column)
	}

	var result sql.NullInt64
	err := r.db.Table(table).Where(query, args...).Select("COALESCE(SUM(" + column + "), 0)").Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.Int64, nil
}

// Avg returns the average of a column for records matching the query
func (r *Query) Avg(table string, column string, query any, args ...any) (float64, error) {
	if !validIdentifier.MatchString(column) {
		return 0, fmt.Errorf("invalid column name: %s", column)
	}

	var result sql.NullFloat64
	err := r.db.Table(table).Where(query, args...).Select("COALESCE(AVG(" + column + "), 0)").Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.Float64, nil
}

// Min returns the minimum value of a column for records matching the query
func (r *Query) Min(table string, column string, query any, args ...any) (int64, error) {
	if !validIdentifier.MatchString(column) {
		return 0, fmt.Errorf("invalid column name: %s", column)
	}

	var result sql.NullInt64
	err := r.db.Table(table).Where(query, args...).Select("MIN(" + column + ")").Scan(&result).Error
	if err != nil {
		return 0, err
	}
	if !result.Valid {
		return 0, ErrNotFound
	}
	return result.Int64, nil
}

// Max returns the maximum value of a column for records matching the query
func (r *Query) Max(table string, column string, query any, args ...any) (int64, error) {
	if !validIdentifier.MatchString(column) {
		return 0, fmt.Errorf("invalid column name: %s", column)
	}

	var result sql.NullInt64
	err := r.db.Table(table).Where(query, args...).Select("MAX(" + column + ")").Scan(&result).Error
	if err != nil {
		return 0, err
	}
	if !result.Valid {
		return 0, ErrNotFound
	}
	return result.Int64, nil
}

// Insert creates new records in the table
func (r *Query) Insert(table string, records ...props.Props) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.Table(table).Create(records).Error
}

// Upsert inserts or updates records based on conflict columns
func (r *Query) Upsert(table string, conflictColumns []string, records ...props.Props) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.Table(table).Clauses(clause.OnConflict{
		Columns:   r.toColumns(conflictColumns),
		UpdateAll: true,
	}).Create(records).Error
}

// Update modifies records matching the query
func (r *Query) Update(table string, changes props.Props, query any, args ...any) (int64, error) {
	result := r.db.Table(table).Where(query, args...).Updates(map[string]any(changes))
	return result.RowsAffected, result.Error
}

// First retrieves the first record matching the query (ordered by sort)
func (r *Query) First(table string, columns []string, sort string, query any, args ...any) (props.Props, error) {
	record := make(map[string]any)
	q := r.db.Table(table)

	if len(columns) > 0 {
		q = q.Select(columns)
	}

	if query != nil {
		q = q.Where(query, args...)
	}

	if sort != "" {
		q = q.Order(sort)
	}

	err := q.Limit(1).Scan(&record).Error
	if err != nil {
		return nil, err
	}
	if len(record) == 0 {
		return nil, ErrNotFound
	}
	return props.Props(record), nil
}

// Last retrieves the last record matching the query (reverse of sort order)
func (r *Query) Last(table string, columns []string, sort string, query any, args ...any) (props.Props, error) {
	// Reverse the sort order for Last
	if sort == "" {
		sort = "id DESC" // Default fallback
	}
	return r.First(table, columns, r.reverseSort(sort), query, args...)
}

// Find retrieves all records matching the query
func (r *Query) Some(table string, columns []string, sort string, query any, args ...any) ([]props.Props, error) {
	return r.Find(table, columns, sort, -1, -1, query, args...)
}

// Page retrieves records with pagination and returns total count
func (r *Query) Page(table string, columns []string, sort string, page int, perPage int, query any, args ...any) ([]props.Props, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	offset := (page - 1) * perPage
	limit := perPage

	return r.Find(table, columns, sort, limit, offset, query, args...)
}

// PageInfo contains pagination metadata
type PageInfo struct {
	Records    []props.Props
	TotalCount int64
	Page       int
	PerPage    int
	TotalPages int64
	HasNext    bool
	HasPrev    bool
}

// PageWithCount retrieves records with pagination and returns total count for proper pagination UIs
func (r *Query) PageWithCount(table string, columns []string, sort string, page int, perPage int, query any, args ...any) (*PageInfo, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	// Get total count
	var totalCount int64
	countQuery := r.db.Table(table)
	if query != nil {
		countQuery = countQuery.Where(query, args...)
	}
	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Get records
	offset := (page - 1) * perPage
	records, err := r.Find(table, columns, sort, perPage, offset, query, args...)
	if err != nil {
		return nil, err
	}

	totalPages := (totalCount + int64(perPage) - 1) / int64(perPage)

	return &PageInfo{
		Records:    records,
		TotalCount: totalCount,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasNext:    int64(page) < totalPages,
		HasPrev:    page > 1,
	}, nil
}

// All retrieves all records from the table
func (r *Query) All(table string, columns []string, sort string) ([]props.Props, error) {
	return r.Find(table, columns, sort, -1, -1, nil)
}

// List retrieves records with pagination support
func (r *Query) Find(table string, columns []string, sort string, limit int, offset int, query any, args ...any) ([]props.Props, error) {
	var records []map[string]any
	q := r.db.Table(table)

	if len(columns) > 0 {
		q = q.Select(columns)
	}

	if query != nil {
		q = q.Where(query, args...)
	}

	if sort != "" {
		q = q.Order(sort)
	}

	if limit > 0 {
		q = q.Limit(limit)
	}

	if offset > 0 {
		q = q.Offset(offset)
	}

	err := q.Scan(&records).Error
	if err != nil {
		return nil, err
	}

	// Convert []map[string]any to []props.Props
	result := make([]props.Props, len(records))
	for i, rec := range records {
		result[i] = props.Props(rec)
	}
	return result, nil
}

// Delete removes records matching the query
func (r *Query) Delete(table string, query any, args ...any) (int64, error) {
	if query == nil {
		return 0, errors.New("delete requires a where clause; use DeleteAll for deleting all records")
	}
	result := r.db.Table(table).Where(query, args...).Delete(nil)
	return result.RowsAffected, result.Error
}

// Raw executes a raw SQL query and returns results
func (r *Query) Raw(sql string, args ...any) ([]props.Props, error) {
	var records []map[string]any
	err := r.db.Raw(sql, args...).Scan(&records).Error
	if err != nil {
		return nil, err
	}

	result := make([]props.Props, len(records))
	for i, rec := range records {
		result[i] = props.Props(rec)
	}
	return result, nil
}

// Exec executes a raw SQL statement (for INSERT, UPDATE, DELETE)
func (r *Query) Exec(sql string, args ...any) (int64, error) {
	result := r.db.Exec(sql, args...)
	return result.RowsAffected, result.Error
}

func (r *Query) toColumns(cols []string) []clause.Column {
	columns := make([]clause.Column, len(cols))
	for i, col := range cols {
		columns[i] = clause.Column{Name: col}
	}
	return columns
}

func (r *Query) reverseSort(sort string) string {
	// Simple reversal: toggle ASC/DESC
	if len(sort) >= 4 && sort[len(sort)-4:] == " ASC" {
		return sort[:len(sort)-4] + " DESC"
	}
	if len(sort) >= 5 && sort[len(sort)-5:] == " DESC" {
		return sort[:len(sort)-5] + " ASC"
	}
	// No explicit order, assume ASC, reverse to DESC
	return sort + " DESC"
}
