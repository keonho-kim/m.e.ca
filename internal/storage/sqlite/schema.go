// Purpose: Define the current SQLite schema for conversations, Plans and compaction evidence.
// Pattern: Repository schema.
// Usage: Applied atomically by Store.initializeSchema.
// Related: internal/storage/sqlite/store.go, internal/storage/sqlite/session.go, internal/storage/sqlite/plans.go.

package sqlite

const schemaVersion = 4

const schema = `
CREATE TABLE sessions (
 id TEXT PRIMARY KEY,
 project_root TEXT NOT NULL,
 display_root TEXT NOT NULL,
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL,
 selection BLOB
);
CREATE INDEX recent_sessions ON sessions(project_root, updated_at DESC);
CREATE TABLE messages (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 session_id TEXT NOT NULL REFERENCES sessions(id),
 body TEXT NOT NULL,
 created_at TEXT NOT NULL
);
CREATE INDEX session_messages ON messages(session_id, sequence);
CREATE TABLE plans (
 id TEXT NOT NULL,
 session_id TEXT NOT NULL REFERENCES sessions(id),
 revision INTEGER NOT NULL CHECK(revision > 0),
 body TEXT NOT NULL,
 updated_at TEXT NOT NULL,
 PRIMARY KEY(session_id, id, revision)
);
CREATE INDEX latest_plans ON plans(session_id, updated_at DESC, revision DESC);
CREATE TABLE project_settings (
 project_root TEXT PRIMARY KEY,
 permission TEXT NOT NULL DEFAULT '',
 scope TEXT NOT NULL DEFAULT ''
);
CREATE TABLE tool_events (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 session_id TEXT NOT NULL REFERENCES sessions(id),
 body BLOB NOT NULL,
 created_at TEXT NOT NULL
);
CREATE INDEX session_tool_events ON tool_events(session_id,sequence);
CREATE TABLE file_selections (
 session_id TEXT NOT NULL REFERENCES sessions(id),
 path TEXT NOT NULL,
 PRIMARY KEY(session_id,path)
);
CREATE TABLE timeline (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 session_id TEXT NOT NULL REFERENCES sessions(id),
 kind TEXT NOT NULL CHECK(kind IN ('message','event')),
 source_sequence INTEGER NOT NULL,
 UNIQUE(kind,source_sequence)
);
CREATE INDEX session_timeline ON timeline(session_id,sequence);
CREATE TABLE conversation_tokens (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 session_id TEXT NOT NULL REFERENCES sessions(id),
 amount INTEGER NOT NULL CHECK(amount>=0),
 estimated INTEGER NOT NULL CHECK(estimated IN (0,1)),
 kind TEXT NOT NULL
);
CREATE INDEX session_tokens ON conversation_tokens(session_id,sequence);
CREATE TABLE memories (
 session_id TEXT NOT NULL REFERENCES sessions(id),
 revision INTEGER NOT NULL,
 through_timeline INTEGER NOT NULL,
 through_usage INTEGER NOT NULL,
 body BLOB NOT NULL,
 created_at TEXT NOT NULL,
 PRIMARY KEY(session_id,revision)
);
`
