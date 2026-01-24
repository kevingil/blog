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

// ValidateTaskStatus validates task status
func ValidateTaskStatus(task *TaskStatus) error {
	if task == nil {
		return nil
	}
	
	if task.TaskID == "" {
		return fmt.Errorf("task ID is required")
	}
	
	validStatuses := map[string]bool{
		TaskStatusQueued:     true,
		TaskStatusInProgress: true,
		TaskStatusCompleted:  true,
		TaskStatusFailed:     true,
	}
	
	if task.Status != "" && !validStatuses[task.Status] {
		return fmt.Errorf("invalid task status: %s", task.Status)
	}
	
	if task.Progress < 0 || task.Progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100, got: %.2f", task.Progress)
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
	
	if err := ValidateTaskStatus(metadata.TaskStatus); err != nil {
		return fmt.Errorf("invalid task status: %w", err)
	}
	
	if err := ValidateUserAction(metadata.UserAction); err != nil {
		return fmt.Errorf("invalid user action: %w", err)
	}
	
	return nil
}

