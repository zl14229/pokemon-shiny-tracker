package database

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

func TestSQLLoggingFiltersBoundPersonalValues(t *testing.T) {
	filter, ok := NewConfig().Logger.(gorm.ParamsFilter)
	if !ok {
		t.Fatal("ORM logger cannot filter bound parameters")
	}
	const query = "SELECT ?"
	sql, params := filter.ParamsFilter(context.Background(), query, "private-test-value")
	if sql != query || len(params) != 0 {
		t.Fatal("ORM logger exposed bound values instead of preserving placeholders")
	}
}
