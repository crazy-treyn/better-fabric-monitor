package utils

import "fmt"

// GenerateFabricURL creates a deep link to Microsoft Fabric for a job run
// Returns an empty string if the item type is not supported
func GenerateFabricURL(workspaceID, itemID, itemType, jobRunID string) string {
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
			"https://app.powerbi.com/workloads/data-pipeline/monitoring/workspaces/%s/pipelines/%s/%s?experience=power-bi",
			workspaceID, itemID, jobRunID,
		)
	case "Notebook":
		// Notebook URL doesn't include itemID in path
		return fmt.Sprintf(
			"https://app.powerbi.com/workloads/de-ds/sparkmonitor/%s/%s?experience=power-bi",
			workspaceID, jobRunID,
		)
	default:
		// Unsupported item type
		return ""
	}
}
