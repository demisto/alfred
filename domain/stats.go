package domain

import "time"

// Statistics holds message and detection statistics for a team
type Statistics struct {
	Team          string    `json:"team"`
	Timestamp     time.Time `json:"ts" db:"ts"`
	Messages      int64     `json:"messages"`
	FilesClean    int64     `json:"files_clean" db:"files_clean"`
	FilesDirty    int64     `json:"files_dirty" db:"files_dirty"`
	FilesUnknown  int64     `json:"files_unknown" db:"files_unknown"`
	URLsClean     int64     `json:"urls_clean" db:"urls_clean"`
	URLsDirty     int64     `json:"urls_dirty" db:"urls_dirty"`
	URLsUnknown   int64     `json:"urls_unknown" db:"urls_unknown"`
	HashesClean   int64     `json:"hashes_clean" db:"hashes_clean"`
	HashesDirty   int64     `json:"hashes_dirty" db:"hashes_dirty"`
	HashesUnknown int64     `json:"hashes_unknown" db:"hashes_unknown"`
	IPsClean      int64     `json:"ips_clean" db:"ips_clean"`
	IPsDirty      int64     `json:"ips_dirty" db:"ips_dirty"`
	IPsUnknown    int64     `json:"ips_unknown" db:"ips_unknown"`
}

// Reset all the counters
func (s *Statistics) Reset() {
	s.Messages = 0
	s.FilesClean = 0
	s.FilesDirty = 0
	s.FilesUnknown = 0
	s.URLsClean = 0
	s.URLsDirty = 0
	s.URLsUnknown = 0
	s.HashesClean = 0
	s.HashesDirty = 0
	s.HashesUnknown = 0
	s.IPsClean = 0
	s.IPsDirty = 0
	s.IPsUnknown = 0
}
