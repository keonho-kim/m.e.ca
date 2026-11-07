// Purpose: Define the single-session SQLite schema for conversation, execution observations and continuation summaries.
// Pattern: Repository schema.
// Usage: Applied to an unpublished database by Sessions.Create.
// Related: internal/storage/sqlite/store.go, internal/storage/sqlite/session.go, internal/storage/sqlite/runs.go.

package sqlite

const schemaVersion = 27

const schema = `
CREATE TABLE team_members (
 ordinal INTEGER PRIMARY KEY CHECK(ordinal>=0),
 agent_id TEXT NOT NULL UNIQUE,
 session_id TEXT NOT NULL UNIQUE,
 summary TEXT NOT NULL DEFAULT ''
);
CREATE TABLE sessions (
 singleton INTEGER PRIMARY KEY DEFAULT 1 CHECK(singleton=1),
 id TEXT NOT NULL UNIQUE,
 project_root TEXT NOT NULL,
 display_root TEXT NOT NULL,
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL,
 team_enabled INTEGER NOT NULL DEFAULT 0,
 context_cursor INTEGER NOT NULL DEFAULT 0,
 selection BLOB,
 summary TEXT
);
CREATE TABLE payloads (
 id INTEGER PRIMARY KEY,
 digest BLOB NOT NULL UNIQUE CHECK(length(digest)=32),
 codec INTEGER NOT NULL,
 raw_bytes INTEGER NOT NULL,
 body BLOB NOT NULL
);
CREATE TABLE public_contexts (
 sequence INTEGER PRIMARY KEY REFERENCES timeline(sequence),
 prefix BLOB NOT NULL,
 codec INTEGER NOT NULL,
 raw_bytes INTEGER NOT NULL,
 digest BLOB NOT NULL CHECK(length(digest)=32)
);
CREATE TABLE messages (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 body BLOB NOT NULL,
 refs BLOB NOT NULL DEFAULT '{}',
 raw_bytes INTEGER NOT NULL,
 role TEXT NOT NULL,
 content_bytes INTEGER NOT NULL,
 created_at TEXT NOT NULL
);
CREATE TABLE tool_events (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 body BLOB NOT NULL,
 refs BLOB NOT NULL DEFAULT '{}',
 raw_bytes INTEGER NOT NULL,
 kind TEXT NOT NULL,
 memory_role TEXT NOT NULL DEFAULT '',
 memory_text TEXT NOT NULL DEFAULT '',
 memory_bytes INTEGER NOT NULL DEFAULT 0,
 memory_source INTEGER NOT NULL DEFAULT 0 CHECK(memory_source>=0),
 display_args BLOB NOT NULL DEFAULT '{}',
 created_at TEXT NOT NULL
);
CREATE INDEX event_kind ON tool_events(kind,sequence);
CREATE TABLE run_states (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 run_id TEXT NOT NULL,
 body BLOB NOT NULL,
 codec INTEGER NOT NULL,
 digest TEXT NOT NULL,
 raw_bytes INTEGER NOT NULL,
 UNIQUE(run_id)
);
CREATE TABLE file_selections (
 path TEXT NOT NULL,
 PRIMARY KEY(path)
);
CREATE TABLE timeline (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 kind TEXT NOT NULL CHECK(kind IN ('message','event')),
 source_sequence INTEGER NOT NULL,
 UNIQUE(kind,source_sequence)
);
CREATE TABLE conversation_tokens (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 singleton INTEGER NOT NULL UNIQUE DEFAULT 1 CHECK(singleton=1),
 amount INTEGER NOT NULL CHECK(amount>=0),
 through_timeline INTEGER NOT NULL DEFAULT 0 CHECK(through_timeline>=0),
 estimated INTEGER NOT NULL CHECK(estimated IN (0,1)),
 kind TEXT NOT NULL
);
CREATE TABLE memories (
 singleton INTEGER PRIMARY KEY DEFAULT 1 CHECK(singleton=1),
 revision INTEGER NOT NULL,
 through_timeline INTEGER NOT NULL,
 through_usage INTEGER NOT NULL,
 body BLOB NOT NULL,
	 codec INTEGER NOT NULL,
	 digest TEXT NOT NULL,
	 raw_bytes INTEGER NOT NULL,
	 created_at TEXT NOT NULL
);
`
