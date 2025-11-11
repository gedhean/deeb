// Package main provides a generator tool to create rules metadata from sql-reviewer's schema.yaml
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Command-line flags
	schemaPath := flag.String("schema", "", "Path to sql-reviewer config/schema.yaml (required)")
	outputGoPath := flag.String("output-go", "", "Path to output Go file (rules_metadata.go)")
	outputTSPath := flag.String("output-ts", "", "Path to output TypeScript file (generated-types.ts)")
	flag.Parse()

	if *schemaPath == "" {
		flag.Usage()
		log.Fatal("Error: -schema flag is required")
	}

	// Resolve paths
	schemaPath = expandPath(schemaPath)

	// Default output paths
	if *outputGoPath == "" {
		*outputGoPath = "./backend/internal/review/rules_metadata.go"
	}
	if *outputTSPath == "" {
		*outputTSPath = "./frontend/src/lib/api/generated-types.ts"
	}

	log.Printf("Parsing schema from: %s", *schemaPath)

	// Parse schema.yaml
	rules, err := ParseSchema(*schemaPath)
	if err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
	}

	log.Printf("Parsed %d rules from schema", len(rules))

	// Generate Go code
	log.Printf("Generating Go code to: %s", *outputGoPath)
	if err := GenerateGoMetadata(rules, *outputGoPath); err != nil {
		log.Fatalf("Failed to generate Go code: %v", err)
	}
	log.Printf("Successfully generated Go metadata")

	// Generate TypeScript types
	log.Printf("Generating TypeScript types to: %s", *outputTSPath)
	if err := GenerateTypeScriptTypes(rules, *outputTSPath); err != nil {
		log.Fatalf("Failed to generate TypeScript types: %v", err)
	}
	log.Printf("Successfully generated TypeScript types")

	log.Println("✅ Rule metadata generation complete!")
}

// expandPath expands ~ and relative paths to absolute paths
func expandPath(path *string) *string {
	p := *path
	if p == "" {
		return path
	}

	// Expand ~
	if p[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, p[1:])
		}
	}

	// Make absolute
	abs, err := filepath.Abs(p)
	if err == nil {
		p = abs
	}

	return &p
}
