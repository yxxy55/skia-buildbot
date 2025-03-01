package sql

// Generated by //go/sql/exporter/
// DO NOT EDIT

const Schema = `CREATE TABLE IF NOT EXISTS Executions (
  execution_id UUID NOT NULL DEFAULT gen_random_uuid(),
  quest_type STRING NOT NULL,
  status STRING,
  creation_time TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
  started_time TIMESTAMPTZ,
  completed_time TIMESTAMPTZ,
  arguments JSONB,
  properties JSONB
);
`

var Executions = []string{
	"execution_id",
	"quest_type",
	"status",
	"creation_time",
	"started_time",
	"completed_time",
	"arguments",
	"properties",
}
