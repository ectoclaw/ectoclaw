package bridge

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var reDailyLog = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.md$`)

var (
	reFrontmatter = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)
	reFrontKey    = regexp.MustCompile(`(?m)^(\w+)\s*:\s*(.+)$`)
)

// IsBootstrap reports whether BOOTSTRAP.md exists in the workspace,
// indicating that first-run onboarding has not yet completed.
func IsBootstrap(workDir string) bool {
	_, err := os.Stat(filepath.Join(workDir, "BOOTSTRAP.md"))
	return err == nil
}

// AssembleSystemPrompt reads workspace files and assembles a system prompt for new sessions.
// All files are optional; missing files are silently skipped.
func AssembleSystemPrompt(workDir string) (string, error) {
	var chunks []string

	// Bootstrap file takes priority — included first so it overrides normal behaviour on first run.
	if bootstrap, err := os.ReadFile(filepath.Join(workDir, "BOOTSTRAP.md")); err == nil {
		if s := strings.TrimSpace(string(bootstrap)); s != "" {
			chunks = append(chunks, s)
		}
	}

	// Static workspace files in order.
	for _, name := range []string{"SYSTEM.md", "SOUL.md", "IDENTITY.md", "USER.md", "MEMORY.md"} {
		content, err := os.ReadFile(filepath.Join(workDir, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if s := strings.TrimSpace(string(content)); s != "" {
			chunks = append(chunks, s)
		}
	}

	// Two most recent daily log files from <workDir>/memory/YYYY-MM-DD.md.
	memoryDir := filepath.Join(workDir, "memory")
	entries, err := os.ReadDir(memoryDir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	var dailyFiles []string
	for _, e := range entries {
		if !e.IsDir() && reDailyLog.MatchString(e.Name()) {
			dailyFiles = append(dailyFiles, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dailyFiles)))
	if len(dailyFiles) > 2 {
		dailyFiles = dailyFiles[:2]
	}
	for _, name := range dailyFiles {
		content, err := os.ReadFile(filepath.Join(memoryDir, name))
		if err != nil {
			continue
		}
		if s := strings.TrimSpace(string(content)); s != "" {
			chunks = append(chunks, s)
		}
	}

	// Skills — inject name, description, and path so the agent can load the full content when needed.
	if section := assembleSkillsSection(workDir); section != "" {
		chunks = append(chunks, section)
	}

	// Hardcoded instructions for file delivery — tied to the [FILE:] / [IMAGE:] marker
	// format parsed by output.go. Always appended last so they're never overridden.
	chunks = append(chunks, fileDeliveryInstructions)

	return strings.Join(chunks, "\n\n"), nil
}

// assembleSkillsSection scans workDir/skills/*/SKILL.md, parses name and description from YAML
// frontmatter, and returns a markdown section listing each skill with its absolute path.
// Skills with missing name or description are silently skipped. Returns "" if none are found.
func assembleSkillsSection(workDir string) string {
	pattern := filepath.Join(workDir, "skills", "*", "SKILL.md")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)

	var lines []string
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		fm := reFrontmatter.FindSubmatch(data)
		if fm == nil {
			continue
		}
		fields := make(map[string]string)
		for _, kv := range reFrontKey.FindAllSubmatch(fm[1], -1) {
			fields[string(kv[1])] = strings.TrimSpace(string(kv[2]))
		}
		name, desc := fields["name"], fields["description"]
		if name == "" || desc == "" {
			continue
		}
		lines = append(lines, "- **"+name+"** (`"+path+"`): "+desc)
	}
	if len(lines) == 0 {
		return ""
	}
	return "## Available Skills\n\nWhen a skill is relevant, read its full file before using it.\n\n" +
		strings.Join(lines, "\n")
}

// fileDeliveryInstructions tells the AI how to send files back through the chat channel.
// The markers are parsed by ParseOutput — keep this in sync with output.go.
const fileDeliveryInstructions = `## Sending Files

To send a file or image to the user, include a marker in your response:
- [FILE: /absolute/path/to/file] — delivers any file type
- [IMAGE: /absolute/path/to/image.png] — delivers an image inline

Place all markers at the very end of your message — no text should follow them. The markers will be stripped from the visible text and the files delivered alongside it. You can include multiple markers.

When downloading or generating files, always save them to the system temp directory (use $TMPDIR, or /tmp as a fallback) — never inside the workspace folder.

To receive a file from the user, ask them to share it — the path will be included in their message.`
