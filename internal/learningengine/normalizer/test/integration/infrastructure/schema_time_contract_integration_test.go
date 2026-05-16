//go:build integration

package infrastructure_test

import (
	"context"
	"testing"
)

func TestSchemaTimeContractForAnalyticsAndLearning(t *testing.T) {
	t.Parallel()

	db := testDB(t)

	rows, err := db.Pool.Query(context.Background(), `
		select table_schema, table_name, column_name, data_type
		from information_schema.columns
		where table_schema in ('analytics', 'learning')
		  and (
		    data_type in ('timestamp without time zone', 'date')
		    or column_name in ('timezone', 'time_zone')
		    or (column_name like '%\_at' escape '\' and data_type <> 'timestamp with time zone')
		  )
		order by table_schema, table_name, column_name
	`)
	if err != nil {
		t.Fatalf("query schema time contract: %v", err)
	}
	defer rows.Close()

	var violations []struct {
		schema   string
		table    string
		column   string
		dataType string
	}
	for rows.Next() {
		var violation struct {
			schema   string
			table    string
			column   string
			dataType string
		}
		if err := rows.Scan(&violation.schema, &violation.table, &violation.column, &violation.dataType); err != nil {
			t.Fatalf("scan schema time contract: %v", err)
		}
		violations = append(violations, violation)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate schema time contract: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("time schema violations = %+v, want none", violations)
	}
}
