// Purpose: Define the single-session SQLite schema for conversation, execution observations and continuation summaries.
// Pattern: Repository schema.
// Usage: Applied to an unpublished database by Sessions.Create.
// Related: internal/storage/sqlite/store.go, internal/storage/sqlite/session.go, internal/storage/sqlite/runs.go.

package sqlite

const schemaVersion = 20

const schema = `
CREATE TABLE sessions (
 singleton INTEGER PRIMARY KEY DEFAULT 1 CHECK(singleton=1),
 id TEXT NOT NULL UNIQUE,
 project_root TEXT NOT NULL,
 display_root TEXT NOT NULL,
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL,
 selection BLOB,
 summary TEXT
);
CREATE TABLE payloads (
 id TEXT PRIMARY KEY,
 codec INTEGER NOT NULL,
 raw_bytes INTEGER NOT NULL,
 body BLOB NOT NULL
);
CREATE TABLE messages (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 session_id TEXT NOT NULL REFERENCES sessions(id),
 body BLOB NOT NULL,
 refs BLOB NOT NULL DEFAULT '{}',
 raw_bytes INTEGER NOT NULL,
 role TEXT NOT NULL,
 content_bytes INTEGER NOT NULL,
 created_at TEXT NOT NULL
);
CREATE INDEX session_messages ON messages(session_id, sequence);
CREATE TABLE tool_events (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 session_id TEXT NOT NULL REFERENCES sessions(id),
 body BLOB NOT NULL,
 refs BLOB NOT NULL DEFAULT '{}',
 raw_bytes INTEGER NOT NULL,
 kind TEXT NOT NULL,
 memory_role TEXT NOT NULL DEFAULT '',
 memory_text TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL
);
CREATE INDEX session_tool_events ON tool_events(session_id,sequence);
CREATE INDEX event_kind ON tool_events(session_id,kind,sequence);
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
 through_timeline INTEGER NOT NULL DEFAULT 0 CHECK(through_timeline>=0),
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
