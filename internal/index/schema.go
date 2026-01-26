package index

const SchemaVersion = 1

const schemaSQL = `
-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY
);

-- Indexed files
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE NOT NULL,
    content_hash TEXT,
    encoding TEXT DEFAULT 'utf-8',
    language TEXT,
    status TEXT DEFAULT 'pending',
    error_message TEXT,
    indexed_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);
CREATE INDEX IF NOT EXISTS idx_files_status ON files(status);
CREATE INDEX IF NOT EXISTS idx_files_language ON files(language);

-- Symbols extracted from files
CREATE TABLE IF NOT EXISTS symbols (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    signature TEXT,
    line_start INTEGER NOT NULL,
    line_end INTEGER,
    column_start INTEGER,
    column_end INTEGER,
    visibility TEXT,
    documentation TEXT,
    is_exported INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);

-- FTS5 for fast symbol search
CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
    name, signature, documentation,
    content=symbols,
    content_rowid=id
);

-- Triggers to keep FTS5 in sync
CREATE TRIGGER IF NOT EXISTS symbols_ai AFTER INSERT ON symbols BEGIN
    INSERT INTO symbols_fts(rowid, name, signature, documentation)
    VALUES (NEW.id, NEW.name, NEW.signature, NEW.documentation);
END;

CREATE TRIGGER IF NOT EXISTS symbols_ad AFTER DELETE ON symbols BEGIN
    INSERT INTO symbols_fts(symbols_fts, rowid, name, signature, documentation)
    VALUES ('delete', OLD.id, OLD.name, OLD.signature, OLD.documentation);
END;

CREATE TRIGGER IF NOT EXISTS symbols_au AFTER UPDATE ON symbols BEGIN
    INSERT INTO symbols_fts(symbols_fts, rowid, name, signature, documentation)
    VALUES ('delete', OLD.id, OLD.name, OLD.signature, OLD.documentation);
    INSERT INTO symbols_fts(rowid, name, signature, documentation)
    VALUES (NEW.id, NEW.name, NEW.signature, NEW.documentation);
END;

-- Symbol references (usages, imports, etc)
CREATE TABLE IF NOT EXISTS symbol_refs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    line INTEGER NOT NULL,
    column INTEGER,
    kind TEXT NOT NULL,
    context TEXT
);

CREATE INDEX IF NOT EXISTS idx_refs_symbol ON symbol_refs(symbol_id);
CREATE INDEX IF NOT EXISTS idx_refs_file ON symbol_refs(file_id);
`

func GetSchema() string {
	return schemaSQL
}

func GetSchemaVersion() int {
	return SchemaVersion
}
