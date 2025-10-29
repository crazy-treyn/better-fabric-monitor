package utils

import "fmt"

// GenerateFabricURL creates a deep link to Microsoft Fabric for a job run
// Returns an empty string if the item type is not supported or required fields are missing
// For notebooks, uses livyID if available, otherwise falls back to jobRunID (which may not work)
func GenerateFabricURL(workspaceID, itemID, itemType, jobRunID string, livyID *string) string {
	// Return empty if any required field is missing
	if workspaceID == "" || jobRunID == "" {
		return ""
	}

	switch itemType {
	case "DataPipeline":
		// Pipeline URL requires itemID as well
		if itemID == "" {
			return ""
		}
		return fmt.Sprintf(
			"https://app.powerbi.com/workloads/data-pipeline/monitoring/workspaces/%s/pipelines/%s/%s?experience=fabric-developer",
			workspaceID, itemID, jobRunID,
		)
	case "Notebook":
		// Notebook URL requires itemID (notebookId) and livyID
		if itemID == "" {
			return ""
		}
		// Use livyID if available for correct URL
		if livyID != nil && *livyID != "" {
			return fmt.Sprintf(
				"https://app.powerbi.com/workloads/de-ds/sparkmonitor/%s/%s?experience=fabric-developer",
				itemID, *livyID,
			)
		}
		// Fall back to jobRunID (may not work, but better than no link)
		// To get correct links, run SyncNotebookSessions() to populate livyID
		return fmt.Sprintf(
			"https://app.powerbi.com/workloads/de-ds/sparkmonitor/%s/%s?experience=fabric-developer",
			itemID, jobRunID,
		)
	default:
		// Unsupported item type
		return ""
	}
}
