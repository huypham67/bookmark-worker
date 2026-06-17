package bookmark

// BookmarkCSVRecord is a single bookmark row carried in an import batch. It
// mirrors the record bookmark-service enqueues.
type BookmarkCSVRecord struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

// BookmarkImportMessage is the message format consumed from the import queue: a
// batch of bookmark records for a specific import job.
type BookmarkImportMessage struct {
	JobID   string              `json:"job_id"`  // Unique identifier for the import job
	UserID  string              `json:"user_id"` // ID of the user who initiated the import
	Records []BookmarkCSVRecord `json:"records"` // Batch of bookmark records to be imported
}
