package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/trailblaze/claude-replay/internal/session"
)

var listCmd = &cobra.Command{
	Use:   "list [project]",
	Short: "List projects or sessions (non-interactive)",
	Long:  "List all projects, or sessions within a project. Useful for scripting.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if gitMode {
			return listGitSessions()
		}
		if len(args) == 0 {
			return listProjects()
		}
		return listSessions(args[0])
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func listProjects() error {
	projects, err := source.ListProjects()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPATH\tSESSIONS\tLAST USED")
	for _, p := range projects {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			p.Name,
			p.Path,
			p.Sessions,
			p.LastUsed.Format("2006-01-02 15:04"),
		)
	}
	return w.Flush()
}

func listSessions(projectQuery string) error {
	projects, err := source.ListProjects()
	if err != nil {
		return err
	}

	// Find matching project
	var matched *struct {
		dirPath string
		name    string
	}
	for _, p := range projects {
		if p.Name == projectQuery || p.DirName == projectQuery || p.Path == projectQuery {
			matched = &struct {
				dirPath string
				name    string
			}{dirPath: p.DirPath, name: p.Name}
			break
		}
	}
	if matched == nil {
		return fmt.Errorf("project not found: %s", projectQuery)
	}

	sessions, err := source.ListSessions(matched.dirPath)
	if err != nil {
		return err
	}

	return printSessionTable(sessions)
}

func listGitSessions() error {
	projects, err := source.ListProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return fmt.Errorf("no sessions found on claude-sessions branch")
	}

	sessions, err := source.ListSessions(projects[0].DirPath)
	if err != nil {
		return err
	}

	return printSessionTable(sessions)
}

func printSessionTable(sessions []session.SessionInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tID\tMODEL\tTURNS\tDATE\tSIZE")
	for _, s := range sessions {
		slug := s.Slug
		if slug == "" {
			slug = "-"
		}
		id := s.ID
		if len(id) > 8 {
			id = id[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			slug,
			id,
			s.Model,
			s.TurnCount,
			s.LastTime.Format("2006-01-02 15:04"),
			formatBytes(s.FileSize),
		)
	}
	return w.Flush()
}

func formatBytes(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.0fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
