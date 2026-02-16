package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

var initCommand = &cli.Command{
	Name:  "init",
	Usage: "Interactively create a tasks.json template",
	Description: `Creates a basic tasks.json template through interactive prompts.
After creation, you can edit the file to add more stories or details.`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workdir",
			Aliases: []string{"w"},
			Usage:   "Working directory",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output file path",
			Value:   "docs/loop/tasks.json",
		},
	},
	Action: func(c *cli.Context) error {
		workDir := c.String("workdir")
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		outputPath := c.String("output")

		scanner := bufio.NewScanner(os.Stdin)

		fmt.Println("🚀 Tingly-Loop Tasks Generator")
		fmt.Println("This will create a tasks.json template for your project.")
		fmt.Println()

		// Project name
		fmt.Print("Project name: ")
		scanner.Scan()
		project := scanner.Text()

		// Branch name
		defaultBranch := "feature/" + strings.ToLower(strings.ReplaceAll(project, " ", "-"))
		fmt.Printf("Branch name [%s]: ", defaultBranch)
		scanner.Scan()
		branch := scanner.Text()
		if branch == "" {
			branch = defaultBranch
		}

		// Description
		fmt.Print("Feature description (one line): ")
		scanner.Scan()
		description := scanner.Text()

		// Collect user stories
		var stories []UserStory
		fmt.Println("\n📝 Enter user stories (press Enter with empty input to finish):")
		fmt.Println("   Format: <title> | <description>")
		fmt.Println("   Example: Add login button | As a user, I want to see a login button")

		storyNum := 1
		for {
			fmt.Printf("\nStory %d (or press Enter to finish): ", storyNum)
			scanner.Scan()
			input := scanner.Text()

			if input == "" {
				break
			}

			// Parse input
			parts := strings.SplitN(input, "|", 2)
			title := strings.TrimSpace(parts[0])
			desc := ""
			if len(parts) > 1 {
				desc = strings.TrimSpace(parts[1])
			} else {
				desc = "As a user, I want " + strings.ToLower(title)
			}

			stories = append(stories, UserStory{
				ID:                 fmt.Sprintf("US-%03d", storyNum),
				Title:              title,
				Description:        desc,
				AcceptanceCriteria: []string{"Specific criterion 1", "Specific criterion 2", "Typecheck passes", "Tests pass"},
				Priority:           storyNum,
				Passes:             false,
				Notes:              "",
			})
			storyNum++
		}

		if len(stories) == 0 {
			// Add a default story if none provided
			stories = append(stories, UserStory{
				ID:                 "US-001",
				Title:              "Example story - replace this",
				Description:        "As a user, I want [feature] so that [benefit]",
				AcceptanceCriteria: []string{"Specific verifiable criterion", "Typecheck passes", "Tests pass"},
				Priority:           1,
				Passes:             false,
				Notes:              "",
			})
		}

		// Create tasks
		tasks := &Tasks{
			Project:     project,
			BranchName:  branch,
			Description: description,
			UserStories: stories,
		}

		// Save
		if err := SaveTasks(outputPath, tasks); err != nil {
			return fmt.Errorf("failed to save tasks: %w", err)
		}

		fmt.Printf("\n✅ Created %s with %d stories\n", outputPath, len(stories))
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit the file to refine acceptance criteria")
		fmt.Println("  2. Run 'tingly-loop run' to start the loop")

		return nil
	},
}

var generateCommand = &cli.Command{
	Name:  "generate",
	Usage: "Generate tasks.json from a feature description using AI",
	Description: `Uses an AI worker to generate a structured tasks.json from a natural language description.

The AI will:
- Break down the feature into small, manageable stories
- Order stories by dependency (schema → backend → UI)
- Add verifiable acceptance criteria

Example:
  tingly-loop generate "Add user authentication with email and password"
  tingly-loop generate "Create a dashboard showing sales metrics"`,
	ArgsUsage: "<feature description>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workdir",
			Aliases: []string{"w"},
			Usage:   "Working directory",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output file path",
			Value:   "docs/loop/tasks.json",
		},
		&cli.StringFlag{
			Name:    "project",
			Aliases: []string{"p"},
			Usage:   "Project name (default: directory name)",
		},
		&cli.StringFlag{
			Name:  "agent",
			Usage: "Agent to use for generation",
			Value: "claude",
		},
	},
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 1 {
			return fmt.Errorf("usage: tingly-loop generate <feature description>")
		}

		featureDesc := c.Args().First()

		workDir := c.String("workdir")
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		projectName := c.String("project")
		if projectName == "" {
			projectName = filepath.Base(workDir)
		}

		outputPath := c.String("output")

		// Build the generation prompt
		prompt := buildGeneratePrompt(featureDesc, projectName)

		// Create agent
		cfg := &Config{
			WorkDir:      workDir,
			AgentType:    c.String("agent"),
			Instructions: "", // We don't need loop instructions for generation
		}

		agent, err := CreateAgent(cfg)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		fmt.Printf("🤖 Generating tasks using %s agent...\n", agent.Name())
		fmt.Printf("Feature: %s\n\n", featureDesc)

		// Call agent
		output, err := agent.Execute(c.Context, prompt)
		if err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}

		fmt.Println(output)

		// Try to extract and save JSON from output
		if err := extractAndSaveTasks(output, outputPath); err != nil {
			fmt.Printf("\n⚠️  Could not automatically extract tasks.json from output.\n")
			fmt.Printf("Please review the output above and create tasks.json manually.\n")
			return err
		}

		fmt.Printf("\n✅ Created %s\n", outputPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Review and edit the generated tasks")
		fmt.Println("  2. Run 'tingly-loop run' to start the loop")

		return nil
	},
}

func buildGeneratePrompt(featureDesc, projectName string) string {
	return fmt.Sprintf(`Generate a tasks.json file for the following feature:

Project: %s
Feature: %s

Requirements:
1. Break down the feature into 3-7 small, manageable user stories
2. Each story must be completable in one iteration (one context window)
3. Order stories by dependency: database → backend → UI
4. Each acceptance criterion must be verifiable (not vague)
5. Always include "Typecheck passes" in acceptance criteria
6. For UI stories, include "Verify in browser" criterion

Output ONLY valid JSON in this exact format:
{
  "project": "%s",
  "branchName": "feature/[kebab-case-feature-name]",
  "description": "[one-line description]",
  "userStories": [
    {
      "id": "US-001",
      "title": "[short title]",
      "description": "As a [user], I want [feature] so that [benefit]",
      "acceptanceCriteria": [
        "[specific criterion]",
        "Typecheck passes"
      ],
      "priority": 1,
      "passes": false,
      "notes": ""
    }
  ]
}

Do not include any text before or after the JSON.`, projectName, featureDesc, projectName)
}

func extractAndSaveTasks(output, outputPath string) error {
	// Find JSON in output
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || end < start {
		return fmt.Errorf("no valid JSON found in output")
	}

	jsonStr := output[start : end+1]

	// Validate it's valid tasks
	var tasks Tasks
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
		return fmt.Errorf("invalid tasks JSON: %w", err)
	}

	// Save
	return SaveTasks(outputPath, &tasks)
}
