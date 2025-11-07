package discover

// ManifestEntry describes the result for a single dependency.
type ManifestEntry struct {
	Name         string    `json:"name"`
	Ecosystem    Ecosystem `json:"ecosystem"`
	Version      string    `json:"version"`
	Repository   string    `json:"repository"`
	Reference    string    `json:"reference"`
	OutputPath   string    `json:"outputPath,omitempty"`
	Status       Status    `json:"status"`
	Reason       string    `json:"reason,omitempty"`
	DocFileCount int       `json:"docFileCount,omitempty"`
}

// Summary aggregates manifest entries for reporting.
type Summary struct {
	Entries []ManifestEntry `json:"entries"`
}

// Count returns the number of entries matching the provided status.
func (summary Summary) Count(status Status) int {
	total := 0
	for _, entry := range summary.Entries {
		if entry.Status == status {
			total++
		}
	}
	return total
}

// EcosystemTotals returns the number of dependencies per ecosystem.
func (summary Summary) EcosystemTotals() map[Ecosystem]int {
	totals := map[Ecosystem]int{}
	for _, entry := range summary.Entries {
		totals[entry.Ecosystem]++
	}
	return totals
}
