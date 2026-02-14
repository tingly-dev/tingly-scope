package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// PRD represents the Product Requirements Document
type PRD struct {
	Project     string      `json:"project"`
	BranchName  string      `json:"branchName"`
	Description string      `json:"description"`
	UserStories []UserStory `json:"userStories"`
}

// UserStory represents a single user story in the PRD
type UserStory struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
	Priority           int      `json:"priority"`
	Passes             bool     `json:"passes"`
	Notes              string   `json:"notes"`
}

// LoadPRD loads a PRD from a JSON file
func LoadPRD(path string) (*PRD, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read PRD file: %w", err)
	}

	var prd PRD
	if err := json.Unmarshal(data, &prd); err != nil {
		return nil, fmt.Errorf("failed to parse PRD JSON: %w", err)
	}

	return &prd, nil
}

// SavePRD saves a PRD to a JSON file
func SavePRD(path string, prd *PRD) error {
	data, err := json.MarshalIndent(prd, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal PRD: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// GetNextStory returns the highest priority story that hasn't passed yet
// Returns nil if all stories have passed
func (p *PRD) GetNextStory() *UserStory {
	// Filter stories that haven't passed
	var pending []UserStory
	for _, story := range p.UserStories {
		if !story.Passes {
			pending = append(pending, story)
		}
	}

	if len(pending) == 0 {
		return nil
	}

	// Sort by priority (ascending - lower number = higher priority)
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Priority < pending[j].Priority
	})

	return &pending[0]
}

// AllStoriesPassed returns true if all stories have passed
func (p *PRD) AllStoriesPassed() bool {
	for _, story := range p.UserStories {
		if !story.Passes {
			return false
		}
	}
	return true
}

// GetPassedCount returns the number of passed stories
func (p *PRD) GetPassedCount() int {
	count := 0
	for _, story := range p.UserStories {
		if story.Passes {
			count++
		}
	}
	return count
}

// GetTotalCount returns the total number of stories
func (p *PRD) GetTotalCount() int {
	return len(p.UserStories)
}

// MarkStoryPassed marks a story as passed by ID
func (p *PRD) MarkStoryPassed(storyID string) bool {
	for i := range p.UserStories {
		if p.UserStories[i].ID == storyID {
			p.UserStories[i].Passes = true
			return true
		}
	}
	return false
}

// FormatStoryList returns a formatted string of all stories with their status
func (p *PRD) FormatStoryList() string {
	result := "User Stories:\n"
	for _, story := range p.UserStories {
		status := "pending"
		if story.Passes {
			status = "completed"
		}
		result += fmt.Sprintf("  [%s] %s (Priority %d): %s\n",
			status, story.ID, story.Priority, story.Title)
	}
	return result
}
