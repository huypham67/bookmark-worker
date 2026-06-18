package bookmark

// BookmarkCSVRecord is a single bookmark row carried in an import batch. It
// mirrors the record bookmark-service enqueues.
type BookmarkCSVRecord struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

// BookmarkImportMessage is the job payload dequeued from the import queue.
type BookmarkImportMessage struct {
	JobID   string              `json:"job_id"`
	UserID  string              `json:"user_id"`
	Records []BookmarkCSVRecord `json:"records"`
}
