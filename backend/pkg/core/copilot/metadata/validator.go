// Package metadata provides structured types for message metadata
package metadata

import "fmt"

// ValidateArtifact validates artifact information
func ValidateArtifact(artifact *ArtifactInfo) error {
	if artifact == nil {
		return nil
	}

	if artifact.ID == "" {
		return fmt.Errorf("artifact ID is required")
	}

	if artifact.Type == "" {
		return fmt.Errorf("artifact type is required")
	}

	// Validate artifact type
	validTypes := map[string]bool{
		ArtifactTypeCodeEdit:          true,
		ArtifactTypeRewrite:           true,
		ArtifactTypeSuggestion:        true,
		ArtifactTypeContentGeneration: true,
		ArtifactTypeImagePrompt:       true,
	}

	if !validTypes[artifact.Type] {
		return fmt.Errorf("invalid artifact type: %s", artifact.Type)
	}

	// Validate status
	validStatuses := map[string]bool{
		ArtifactStatusPending:  true,
		ArtifactStatusAccepted: true,
		ArtifactStatusRejected: true,
		ArtifactStatusApplied:  true,
	}

	if artifact.Status != "" && !validStatuses[artifact.Status] {
		return fmt.Errorf("invalid artifact status: %s", artifact.Status)
	}

	return nil
}

// ValidateUserAction validates user action
func ValidateUserAction(action *UserAction) error {
	if action == nil {
		return nil
	}

	validActions := map[string]bool{
		UserActionAccept: true,
		UserActionReject: true,
		UserActionModify: true,
	}

	if action.Action == "" {
		return fmt.Errorf("action is required")
	}

	if !validActions[action.Action] {
		return fmt.Errorf("invalid action: %s", action.Action)
	}

	return nil
}

// ValidateMetaData validates the entire metadata structure
func ValidateMetaData(metadata *MessageMetaData) error {
	if metadata == nil {
		return nil
	}

	if err := ValidateArtifact(metadata.Artifact); err != nil {
		return fmt.Errorf("invalid artifact: %w", err)
	}

	if err := ValidateUserAction(metadata.UserAction); err != nil {
		return fmt.Errorf("invalid user action: %w", err)
	}

	return nil
}
