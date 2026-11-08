package storage

var reliabilitySchema = []string{
	`CREATE TABLE IF NOT EXISTS reliability_series (id TEXT PRIMARY KEY, project TEXT NOT NULL, payload TEXT NOT NULL, first_at TEXT NOT NULL, last_at TEXT NOT NULL)`,
	`CREATE INDEX IF NOT EXISTS reliability_series_project ON reliability_series (project,id)`,
	`CREATE TABLE IF NOT EXISTS reliability_authorities (series TEXT NOT NULL, source TEXT NOT NULL, PRIMARY KEY(series,source))`,
	`CREATE TABLE IF NOT EXISTS reliability_observations (id TEXT PRIMARY KEY, series TEXT NOT NULL, at TEXT NOT NULL, created_at TEXT NOT NULL, payload TEXT NOT NULL)`,
	`CREATE INDEX IF NOT EXISTS reliability_observations_time ON reliability_observations (series,at,id)`,
	`CREATE INDEX IF NOT EXISTS reliability_observations_logical ON reliability_observations (series,created_at,at)`,
	`CREATE TABLE IF NOT EXISTS reliability_jobs (series TEXT PRIMARY KEY, since_at TEXT NOT NULL, until_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS reliability_policies (series TEXT NOT NULL, max_age BIGINT NOT NULL, PRIMARY KEY(series,max_age))`,
	`CREATE TABLE IF NOT EXISTS reliability_summaries (series TEXT NOT NULL, max_age BIGINT NOT NULL, resolution TEXT NOT NULL, start_at TEXT NOT NULL, payload TEXT NOT NULL, compact TEXT NOT NULL, PRIMARY KEY(series,max_age,resolution,start_at))`,
}
