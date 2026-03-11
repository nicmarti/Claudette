// Package embeddings provides SQLite blob storage for vector embeddings.
// ML model support is deferred; only keyword search is available in the Go version.
package embeddings

import (
	"database/sql"
	"encoding/binary"
	"math"

	_ "github.com/mattn/go-sqlite3"
)

const embeddingsSchema = `
CREATE TABLE IF NOT EXISTS embeddings (
    qualified_name TEXT PRIMARY KEY,
    vector BLOB NOT NULL,
    text_hash TEXT NOT NULL
);
`

// EmbeddingStore manages vector embeddings for graph nodes in SQLite.
type EmbeddingStore struct {
	db        *sql.DB
	Available bool
}

// NewEmbeddingStore creates a new embedding store.
func NewEmbeddingStore(dbPath string) (*EmbeddingStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(embeddingsSchema)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &EmbeddingStore{db: db, Available: false}, nil
}

// Close closes the database connection.
func (s *EmbeddingStore) Close() error {
	return s.db.Close()
}

// Count returns the number of stored embeddings.
func (s *EmbeddingStore) Count() int {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	return count
}

// EncodeVector encodes a float32 vector as a compact binary blob.
func EncodeVector(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// DecodeVector decodes a binary blob to a float32 vector.
func DecodeVector(blob []byte) []float32 {
	n := len(blob) / 4
	vec := make([]float32, n)
	for i := range n {
		vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return vec
}

// CosineSimilarity computes cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (normA * normB)
}
