package job

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JobInstructions represents the instructions for processing an uploaded file
type JobInstructions struct {
	FilePath     string      `json:"file_path"`     // Path to the temp folder containing the file
	OriginalFile string      `json:"original_file"` // Original filename
	Hash         string      `json:"hash"`          // SHA256 hash
	Job          combinedJob `json:"job"`           // The parsed job details
}

// WriteInstructions writes the job instructions to instructions.json in the given directory
func WriteInstructions(dir string, instr JobInstructions) error {
	path := filepath.Join(dir, "instructions.json")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create instructions file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(instr); err != nil {
		return fmt.Errorf("failed to encode instructions: %w", err)
	}
	return nil
}

// ReadInstructions reads job instructions from instructions.json in the given directory
func ReadInstructions(dir string) (JobInstructions, error) {
	path := filepath.Join(dir, "instructions.json")
	file, err := os.Open(path)
	if err != nil {
		return JobInstructions{}, fmt.Errorf("failed to open instructions file: %w", err)
	}
	defer file.Close()

	var instr JobInstructions
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&instr); err != nil {
		return JobInstructions{}, fmt.Errorf("failed to decode instructions: %w", err)
	}
	return instr, nil
}
