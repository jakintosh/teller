package session

import (
	"encoding/json"
	"fmt"
	"os"

	"git.sr.ht/~jakintosh/teller/internal/core"
)

const sessionFileName = ".teller-session.tmp"

// saveBatch serializes the current batch to a temporary file.
func SaveBatch(batch []core.Transaction) error {
	if len(batch) == 0 {
		// Don't save empty batches
		return DeleteSession()
	}

	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	err = os.WriteFile(sessionFileName, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// loadBatch deserializes a batch from a temporary file.
func LoadBatch() ([]core.Transaction, error) {
	data, err := os.ReadFile(sessionFileName)
	if err != nil {
		if os.IsNotExist(err) {
			// No session file exists, return empty batch
			return []core.Transaction{}, nil
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var batch []core.Transaction
	err = json.Unmarshal(data, &batch)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return batch, nil
}

// HasSession returns true if a session file exists.
func HasSession() bool {
	_, err := os.Stat(sessionFileName)
	return err == nil
}

// DeleteSession removes the session file.
func DeleteSession() error {
	err := os.Remove(sessionFileName)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}
