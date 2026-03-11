package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schemaDDL = `
CREATE TABLE IF NOT EXISTS nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    qualified_name TEXT NOT NULL UNIQUE,
    file_path TEXT NOT NULL,
    line_start INTEGER,
    line_end INTEGER,
    language TEXT,
    parent_name TEXT,
    params TEXT,
    return_type TEXT,
    modifiers TEXT,
    is_test INTEGER DEFAULT 0,
    file_hash TEXT,
    extra TEXT DEFAULT '{}',
    updated_at REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    source_qualified TEXT NOT NULL,
    target_qualified TEXT NOT NULL,
    file_path TEXT NOT NULL,
    line INTEGER DEFAULT 0,
    extra TEXT DEFAULT '{}',
    updated_at REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_nodes_file ON nodes(file_path);
CREATE INDEX IF NOT EXISTS idx_nodes_kind ON nodes(kind);
CREATE INDEX IF NOT EXISTS idx_nodes_qualified ON nodes(qualified_name);
CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_qualified);
CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_qualified);
CREATE INDEX IF NOT EXISTS idx_edges_kind ON edges(kind);
CREATE INDEX IF NOT EXISTS idx_edges_file ON edges(file_path);
`

// GraphStore is a SQLite-backed code knowledge graph.
type GraphStore struct {
	db *sql.DB
}

// NewGraphStore opens or creates a graph database at the given path.
// Use ":memory:" for an in-memory database (useful for tests).
func NewGraphStore(dbPath string) (*GraphStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_timeout=30000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writers

	s := &GraphStore{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *GraphStore) initSchema() error {
	_, err := s.db.Exec(schemaDDL)
	return err
}

// Close closes the database connection.
func (s *GraphStore) Close() error {
	return s.db.Close()
}

// MakeQualified creates a qualified name for a node.
func MakeQualified(node *NodeInfo) string {
	if node.Kind == "File" {
		return node.FilePath
	}
	if node.ParentName != "" {
		return fmt.Sprintf("%s::%s.%s", node.FilePath, node.ParentName, node.Name)
	}
	return fmt.Sprintf("%s::%s", node.FilePath, node.Name)
}

// UpsertNode inserts or updates a node. Returns the node ID.
func (s *GraphStore) UpsertNode(node *NodeInfo, fileHash string) (int64, error) {
	now := float64(time.Now().UnixMilli()) / 1000.0
	qualified := MakeQualified(node)
	extra := "{}"
	if node.Extra != nil {
		b, _ := json.Marshal(node.Extra)
		extra = string(b)
	}

	isTest := 0
	if node.IsTest {
		isTest = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO nodes
		(kind, name, qualified_name, file_path, line_start, line_end,
		 language, parent_name, params, return_type, modifiers, is_test,
		 file_hash, extra, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(qualified_name) DO UPDATE SET
		  kind=excluded.kind, name=excluded.name,
		  file_path=excluded.file_path, line_start=excluded.line_start,
		  line_end=excluded.line_end, language=excluded.language,
		  parent_name=excluded.parent_name, params=excluded.params,
		  return_type=excluded.return_type, modifiers=excluded.modifiers,
		  is_test=excluded.is_test, file_hash=excluded.file_hash,
		  extra=excluded.extra, updated_at=excluded.updated_at`,
		node.Kind, node.Name, qualified, node.FilePath,
		node.LineStart, node.LineEnd, node.Language,
		node.ParentName, node.Params, node.ReturnType,
		node.Modifiers, isTest, fileHash,
		extra, now,
	)
	if err != nil {
		return 0, fmt.Errorf("upsert node: %w", err)
	}

	var id int64
	err = s.db.QueryRow("SELECT id FROM nodes WHERE qualified_name = ?", qualified).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get node id: %w", err)
	}
	return id, nil
}

// UpsertEdge inserts or updates an edge. Returns the edge ID.
func (s *GraphStore) UpsertEdge(edge *EdgeInfo) (int64, error) {
	now := float64(time.Now().UnixMilli()) / 1000.0
	extra := "{}"
	if edge.Extra != nil {
		b, _ := json.Marshal(edge.Extra)
		extra = string(b)
	}

	var existingID int64
	err := s.db.QueryRow(
		`SELECT id FROM edges WHERE kind=? AND source_qualified=? AND target_qualified=? AND file_path=?`,
		edge.Kind, edge.Source, edge.Target, edge.FilePath,
	).Scan(&existingID)

	if err == nil {
		_, err = s.db.Exec("UPDATE edges SET line=?, extra=?, updated_at=? WHERE id=?",
			edge.Line, extra, now, existingID)
		if err != nil {
			return 0, fmt.Errorf("update edge: %w", err)
		}
		return existingID, nil
	}

	res, err := s.db.Exec(`
		INSERT INTO edges
		(kind, source_qualified, target_qualified, file_path, line, extra, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		edge.Kind, edge.Source, edge.Target, edge.FilePath, edge.Line, extra, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert edge: %w", err)
	}
	return res.LastInsertId()
}

// RemoveFileData removes all nodes and edges associated with a file.
func (s *GraphStore) RemoveFileData(filePath string) error {
	_, err := s.db.Exec("DELETE FROM nodes WHERE file_path = ?", filePath)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM edges WHERE file_path = ?", filePath)
	return err
}

// StoreFileNodesEdges atomically replaces all data for a file.
func (s *GraphStore) StoreFileNodesEdges(filePath string, nodes []NodeInfo, edges []EdgeInfo, fileHash string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM nodes WHERE file_path = ?", filePath)
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM edges WHERE file_path = ?", filePath)
	if err != nil {
		return err
	}

	now := float64(time.Now().UnixMilli()) / 1000.0

	for _, node := range nodes {
		qualified := MakeQualified(&node)
		extra := "{}"
		if node.Extra != nil {
			b, _ := json.Marshal(node.Extra)
			extra = string(b)
		}
		isTest := 0
		if node.IsTest {
			isTest = 1
		}
		_, err = tx.Exec(`
			INSERT INTO nodes
			(kind, name, qualified_name, file_path, line_start, line_end,
			 language, parent_name, params, return_type, modifiers, is_test,
			 file_hash, extra, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(qualified_name) DO UPDATE SET
			  kind=excluded.kind, name=excluded.name,
			  file_path=excluded.file_path, line_start=excluded.line_start,
			  line_end=excluded.line_end, language=excluded.language,
			  parent_name=excluded.parent_name, params=excluded.params,
			  return_type=excluded.return_type, modifiers=excluded.modifiers,
			  is_test=excluded.is_test, file_hash=excluded.file_hash,
			  extra=excluded.extra, updated_at=excluded.updated_at`,
			node.Kind, node.Name, qualified, node.FilePath,
			node.LineStart, node.LineEnd, node.Language,
			node.ParentName, node.Params, node.ReturnType,
			node.Modifiers, isTest, fileHash,
			extra, now,
		)
		if err != nil {
			return fmt.Errorf("insert node %s: %w", node.Name, err)
		}
	}

	for _, edge := range edges {
		extra := "{}"
		if edge.Extra != nil {
			b, _ := json.Marshal(edge.Extra)
			extra = string(b)
		}
		_, err = tx.Exec(`
			INSERT INTO edges
			(kind, source_qualified, target_qualified, file_path, line, extra, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			edge.Kind, edge.Source, edge.Target, edge.FilePath, edge.Line, extra, now,
		)
		if err != nil {
			return fmt.Errorf("insert edge %s->%s: %w", edge.Source, edge.Target, err)
		}
	}

	return tx.Commit()
}

// SetMetadata stores a key-value pair in the metadata table.
func (s *GraphStore) SetMetadata(key, value string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)", key, value)
	return err
}

// GetMetadata retrieves a value from the metadata table.
func (s *GraphStore) GetMetadata(key string) (string, bool) {
	var value string
	err := s.db.QueryRow("SELECT value FROM metadata WHERE key=?", key).Scan(&value)
	if err != nil {
		return "", false
	}
	return value, true
}

// GetNode returns a node by qualified name, or nil if not found.
func (s *GraphStore) GetNode(qualifiedName string) *GraphNode {
	row := s.db.QueryRow("SELECT * FROM nodes WHERE qualified_name = ?", qualifiedName)
	return s.scanNode(row)
}

// GetNodesByFile returns all nodes in a given file.
func (s *GraphStore) GetNodesByFile(filePath string) []*GraphNode {
	rows, err := s.db.Query("SELECT * FROM nodes WHERE file_path = ?", filePath)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return s.scanNodes(rows)
}

// GetEdgesBySource returns all edges originating from a given qualified name.
func (s *GraphStore) GetEdgesBySource(qualifiedName string) []*GraphEdge {
	rows, err := s.db.Query("SELECT * FROM edges WHERE source_qualified = ?", qualifiedName)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return s.scanEdges(rows)
}

// GetEdgesByTarget returns all edges targeting a given qualified name.
func (s *GraphStore) GetEdgesByTarget(qualifiedName string) []*GraphEdge {
	rows, err := s.db.Query("SELECT * FROM edges WHERE target_qualified = ?", qualifiedName)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return s.scanEdges(rows)
}

// GetAllFiles returns all distinct file paths that have a File node.
func (s *GraphStore) GetAllFiles() []string {
	rows, err := s.db.Query("SELECT DISTINCT file_path FROM nodes WHERE kind = 'File'")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var files []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err == nil {
			files = append(files, f)
		}
	}
	return files
}

// SearchNodes performs keyword search across node names.
func (s *GraphStore) SearchNodes(query string, limit int) []*GraphNode {
	pattern := "%" + query + "%"
	rows, err := s.db.Query(
		"SELECT * FROM nodes WHERE name LIKE ? OR qualified_name LIKE ? LIMIT ?",
		pattern, pattern, limit,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return s.scanNodes(rows)
}

// GetAllEdges returns all edges in the graph.
func (s *GraphStore) GetAllEdges() []*GraphEdge {
	rows, err := s.db.Query("SELECT * FROM edges")
	if err != nil {
		return nil
	}
	defer rows.Close()
	return s.scanEdges(rows)
}

// GetEdgesAmong returns edges where both source and target are in the given set.
func (s *GraphStore) GetEdgesAmong(qualifiedNames map[string]bool) []*GraphEdge {
	if len(qualifiedNames) == 0 {
		return nil
	}
	qns := make([]string, 0, len(qualifiedNames))
	for qn := range qualifiedNames {
		qns = append(qns, qn)
	}
	placeholders := strings.Repeat("?,", len(qns))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]any, 0, len(qns)*2)
	for _, qn := range qns {
		args = append(args, qn)
	}
	for _, qn := range qns {
		args = append(args, qn)
	}

	query := fmt.Sprintf(
		"SELECT * FROM edges WHERE source_qualified IN (%s) AND target_qualified IN (%s)",
		placeholders, placeholders,
	)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return s.scanEdges(rows)
}

// GetStats returns aggregate statistics about the graph.
func (s *GraphStore) GetStats() *GraphStats {
	stats := &GraphStats{
		NodesByKind: make(map[string]int),
		EdgesByKind: make(map[string]int),
	}

	s.db.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&stats.TotalNodes)
	s.db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&stats.TotalEdges)

	rows, err := s.db.Query("SELECT kind, COUNT(*) as cnt FROM nodes GROUP BY kind")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var kind string
			var cnt int
			if rows.Scan(&kind, &cnt) == nil {
				stats.NodesByKind[kind] = cnt
			}
		}
	}

	rows2, err := s.db.Query("SELECT kind, COUNT(*) as cnt FROM edges GROUP BY kind")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var kind string
			var cnt int
			if rows2.Scan(&kind, &cnt) == nil {
				stats.EdgesByKind[kind] = cnt
			}
		}
	}

	rows3, err := s.db.Query("SELECT DISTINCT language FROM nodes WHERE language IS NOT NULL AND language != ''")
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var lang string
			if rows3.Scan(&lang) == nil {
				stats.Languages = append(stats.Languages, lang)
			}
		}
	}

	s.db.QueryRow("SELECT COUNT(*) FROM nodes WHERE kind = 'File'").Scan(&stats.FilesCount)

	if val, ok := s.GetMetadata("last_updated"); ok {
		stats.LastUpdated = val
	}

	return stats
}

// scan helpers

func (s *GraphStore) scanNode(row *sql.Row) *GraphNode {
	n := &GraphNode{}
	var isTest int
	var extraStr, parentName, params, returnType, modifiers, language, fileHash sql.NullString
	var lineStart, lineEnd sql.NullInt64
	err := row.Scan(
		&n.ID, &n.Kind, &n.Name, &n.QualifiedName, &n.FilePath,
		&lineStart, &lineEnd, &language, &parentName, &params,
		&returnType, &modifiers, &isTest, &fileHash, &extraStr, new(float64),
	)
	if err != nil {
		return nil
	}
	n.IsTest = isTest != 0
	n.Language = language.String
	n.ParentName = parentName.String
	n.Params = params.String
	n.ReturnType = returnType.String
	n.FileHash = fileHash.String
	if lineStart.Valid {
		n.LineStart = int(lineStart.Int64)
	}
	if lineEnd.Valid {
		n.LineEnd = int(lineEnd.Int64)
	}
	if extraStr.Valid && extraStr.String != "" {
		json.Unmarshal([]byte(extraStr.String), &n.Extra)
	}
	return n
}

func (s *GraphStore) scanNodes(rows *sql.Rows) []*GraphNode {
	var nodes []*GraphNode
	for rows.Next() {
		n := &GraphNode{}
		var isTest int
		var extraStr, parentName, params, returnType, modifiers, language, fileHash sql.NullString
		var lineStart, lineEnd sql.NullInt64
		err := rows.Scan(
			&n.ID, &n.Kind, &n.Name, &n.QualifiedName, &n.FilePath,
			&lineStart, &lineEnd, &language, &parentName, &params,
			&returnType, &modifiers, &isTest, &fileHash, &extraStr, new(float64),
		)
		if err != nil {
			continue
		}
		n.IsTest = isTest != 0
		n.Language = language.String
		n.ParentName = parentName.String
		n.Params = params.String
		n.ReturnType = returnType.String
		n.FileHash = fileHash.String
		if lineStart.Valid {
			n.LineStart = int(lineStart.Int64)
		}
		if lineEnd.Valid {
			n.LineEnd = int(lineEnd.Int64)
		}
		if extraStr.Valid && extraStr.String != "" {
			json.Unmarshal([]byte(extraStr.String), &n.Extra)
		}
		nodes = append(nodes, n)
	}
	return nodes
}

func (s *GraphStore) scanEdges(rows *sql.Rows) []*GraphEdge {
	var edges []*GraphEdge
	for rows.Next() {
		e := &GraphEdge{}
		var extraStr sql.NullString
		err := rows.Scan(
			&e.ID, &e.Kind, &e.SourceQualified, &e.TargetQualified,
			&e.FilePath, &e.Line, &extraStr, new(float64),
		)
		if err != nil {
			continue
		}
		if extraStr.Valid && extraStr.String != "" {
			json.Unmarshal([]byte(extraStr.String), &e.Extra)
		}
		edges = append(edges, e)
	}
	return edges
}
