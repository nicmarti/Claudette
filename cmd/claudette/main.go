package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"claudette/internal/graph"
	"claudette/internal/incremental"
	"claudette/internal/server"
	"claudette/internal/visualization"
)

const version = "1.0.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "claudette",
		Short: "Persistent incremental knowledge graph for code reviews",
		Run: func(cmd *cobra.Command, args []string) {
			printBanner()
		},
	}

	rootCmd.AddCommand(
		installCmd(),
		buildCmd(),
		updateCmd(),
		watchCmd(),
		statusCmd(),
		visualizeCmd(),
		serveCmd(),
		versionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func printBanner() {
	fmt.Printf(`
  ●──●──●
  │╲ │ ╱│       claudette  v%s
  ●──◆──●
  │╱ │ ╲│       Structural knowledge graph for
  ●──●──●       smarter code reviews

  Commands:
    install     Set up Claude Code integration
    build       Full graph build (parse all files)
    update      Incremental update (changed files only)
    watch       Auto-update on file changes
    status      Show graph statistics
    visualize   Generate interactive HTML graph
    serve       Start MCP server

  Run claudette <command> --help for details
`, version)
}

func installCmd() *cobra.Command {
	var repo string
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"init"},
		Short:   "Register MCP server with Claude Code (creates .mcp.json)",
		Run: func(cmd *cobra.Command, args []string) {
			repoRoot := repo
			if repoRoot == "" {
				repoRoot = incremental.FindRepoRoot("")
				if repoRoot == "" {
					cwd, _ := os.Getwd()
					repoRoot = cwd
				}
			}

			mcpPath := filepath.Join(repoRoot, ".mcp.json")
			mcpConfig := map[string]any{
				"mcpServers": map[string]any{
					"claudette": map[string]any{
						"command": "claudette",
						"args":   []string{"serve"},
					},
				},
			}

			// Merge into existing .mcp.json
			if data, err := os.ReadFile(mcpPath); err == nil {
				var existing map[string]any
				if json.Unmarshal(data, &existing) == nil {
					servers, _ := existing["mcpServers"].(map[string]any)
					if servers != nil {
						if _, ok := servers["claudette"]; ok {
							fmt.Printf("Already configured in %s\n", mcpPath)
							return
						}
						servers["claudette"] = mcpConfig["mcpServers"].(map[string]any)["claudette"]
						mcpConfig = existing
					}
				}
			}

			if dryRun {
				data, _ := json.MarshalIndent(mcpConfig, "", "  ")
				fmt.Printf("[dry-run] Would write to %s:\n%s\n\n[dry-run] No files were modified.\n", mcpPath, string(data))
				return
			}

			data, _ := json.MarshalIndent(mcpConfig, "", "  ")
			os.WriteFile(mcpPath, append(data, '\n'), 0o644)
			fmt.Printf("Created %s\n\nNext steps:\n  1. claudette build    # build the knowledge graph\n  2. Restart Claude Code        # to pick up the new MCP server\n", mcpPath)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "Repository root (auto-detected)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without writing")
	return cmd
}

func buildCmd() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Full graph build (re-parse all files)",
		Run: func(cmd *cobra.Command, args []string) {
			repoRoot := resolveRoot(repo)
			store := openStore(repoRoot)
			defer store.Close()

			result := incremental.FullBuild(repoRoot, store)
			fmt.Printf("Full build: %d files, %d nodes, %d edges\n",
				result.FilesParsed, result.TotalNodes, result.TotalEdges)
			if len(result.Errors) > 0 {
				fmt.Printf("Errors: %d\n", len(result.Errors))
			}
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "Repository root (auto-detected)")
	return cmd
}

func updateCmd() *cobra.Command {
	var repo, base string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Incremental update (only changed files)",
		Run: func(cmd *cobra.Command, args []string) {
			repoRoot := incremental.FindRepoRoot("")
			if repo != "" {
				repoRoot = repo
			}
			if repoRoot == "" {
				log.Fatal("Not in a git repository. 'update' requires git for diffing. Use 'build' for a full parse.")
			}

			store := openStore(repoRoot)
			defer store.Close()

			result := incremental.IncrementalUpdate(repoRoot, store, base, nil)
			fmt.Printf("Incremental: %d files updated, %d nodes, %d edges\n",
				result.FilesUpdated, result.TotalNodes, result.TotalEdges)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "Repository root (auto-detected)")
	cmd.Flags().StringVar(&base, "base", "HEAD~1", "Git diff base")
	return cmd
}

func watchCmd() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch for changes and auto-update",
		Run: func(cmd *cobra.Command, args []string) {
			repoRoot := resolveRoot(repo)
			store := openStore(repoRoot)
			defer store.Close()

			if err := incremental.Watch(repoRoot, store); err != nil {
				log.Fatalf("Watch error: %v", err)
			}
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "Repository root (auto-detected)")
	return cmd
}

func statusCmd() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show graph statistics",
		Run: func(cmd *cobra.Command, args []string) {
			repoRoot := resolveRoot(repo)
			store := openStore(repoRoot)
			defer store.Close()

			stats := store.GetStats()
			fmt.Printf("Nodes: %d\n", stats.TotalNodes)
			fmt.Printf("Edges: %d\n", stats.TotalEdges)
			fmt.Printf("Files: %d\n", stats.FilesCount)
			if len(stats.Languages) > 0 {
				fmt.Printf("Languages: ")
				for i, l := range stats.Languages {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Print(l)
				}
				fmt.Println()
			}
			if stats.LastUpdated != "" {
				fmt.Printf("Last updated: %s\n", stats.LastUpdated)
			} else {
				fmt.Println("Last updated: never")
			}
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "Repository root (auto-detected)")
	return cmd
}

func visualizeCmd() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "visualize",
		Short: "Generate interactive HTML graph visualization",
		Run: func(cmd *cobra.Command, args []string) {
			repoRoot := resolveRoot(repo)
			store := openStore(repoRoot)
			defer store.Close()

			htmlPath := filepath.Join(repoRoot, ".claudette", "graph.html")
			if err := visualization.GenerateHTML(store, htmlPath); err != nil {
				log.Fatalf("Visualization error: %v", err)
			}
			fmt.Printf("Visualization: %s\nOpen in browser to explore your codebase graph.\n", htmlPath)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "Repository root (auto-detected)")
	return cmd
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start MCP server (stdio transport)",
		Run: func(cmd *cobra.Command, args []string) {
			if err := server.Serve(); err != nil {
				log.Fatalf("Server error: %v", err)
			}
		},
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("claudette %s\n", version)
		},
	}
}

func resolveRoot(repo string) string {
	if repo != "" {
		return repo
	}
	return incremental.FindProjectRoot("")
}

func openStore(repoRoot string) *graph.GraphStore {
	dbPath := incremental.GetDBPath(repoRoot)
	store, err := graph.NewGraphStore(dbPath)
	if err != nil {
		log.Fatalf("Cannot open graph database: %v", err)
	}
	return store
}
