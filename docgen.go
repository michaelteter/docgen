// go build -o docgen main.go

package main

import (
    "bufio"
    "bytes"
    "flag"
    "fmt"
    "io/fs"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"
)

var (
    filesList    = flag.Bool("files-list", false, "Print included files only")
    graphicTree  = flag.Bool("graphic-tree", false, "Print symbol-formatted tree")
    plainTree    = flag.Bool("plain-tree", false, "Print space-indented tree")
    outputPath   = flag.String("output", "docs/project_doc.txt", "Path to write full documentation output")
    maxSizeBytes = int64(2 * 1024 * 1024) // 2MB

    // Default values if .docgen_ignore doesn't exist
    defaultAlwaysInclude = []string{
        ".gitignore",
        "Makefile",
        "requirements.txt",
    }
    defaultAlsoExclude = []string{
        ".git/",
        ".DS_Store",
        "__pycache__/",
        "node_modules/",
        "*.pyc",
        "*.log",
        "*.tmp",
        "*.swp",
        "*.bak",
        "*.out",
    }
)

type Config struct {
    AlwaysInclude []string
    AlsoExclude   []string
}

func main() {
    flag.Parse()
    root := "."

    config := loadConfig()
    allFiles := collectFiles(root)
    ignored := getIgnoredPaths(allFiles, config.AlsoExclude)
    included := filterIncluded(allFiles, ignored, config.AlwaysInclude)

    switch {
    case *filesList:
        for _, f := range included {
            fmt.Println(f)
        }
    case *graphicTree:
        printTree(included, true)
    case *plainTree:
        printTree(included, false)
    default:
        var buf bytes.Buffer
        buf.WriteString(".\n")
        buf.WriteString(treeString(included, true))
        buf.WriteString("\n")
        for _, f := range included {
            buf.WriteString(fmt.Sprintf(">>>> FILE CONTENTS: %s %s\n", f, strings.Repeat("=", 80-len(f)-22)))
            content, err := os.ReadFile(f)
            if err != nil {
                buf.WriteString(fmt.Sprintf("ERROR reading %s: %v\n\n", f, err))
                continue
            }
            buf.Write(content)
            buf.WriteString("\n\n")
        }
        os.MkdirAll(filepath.Dir(*outputPath), 0755)
        os.WriteFile(*outputPath, buf.Bytes(), 0644)
        fmt.Printf("Documentation written to %s\n", *outputPath)
    }
}

func loadConfig() Config {
    config := Config{
        AlwaysInclude: defaultAlwaysInclude,
        AlsoExclude:   defaultAlsoExclude,
    }

    file, err := os.Open(".docgen_ignore")
    if err != nil {
        // File doesn't exist, use defaults
        return config
    }
    defer file.Close()

    // Clear defaults since we're reading from file
    config.AlwaysInclude = []string{}
    config.AlsoExclude = []string{}

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())

        // Skip empty lines and comments
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }

        // Lines starting with ! are always included
        if strings.HasPrefix(line, "!") {
            pattern := strings.TrimPrefix(line, "!")
            config.AlwaysInclude = append(config.AlwaysInclude, pattern)
        } else {
            // Everything else is excluded
            config.AlsoExclude = append(config.AlsoExclude, line)
        }
    }

    return config
}

func collectFiles(root string) []string {
    var files []string
    filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return nil
        }
        info, err := d.Info()
        if err != nil || info.Size() > maxSizeBytes {
            return nil
        }
        files = append(files, path)
        return nil
    })
    return files
}

func getIgnoredPaths(paths []string, alsoExclude []string) map[string]bool {
    ignored := make(map[string]bool)

    // First pass: manually check ALSO_EXCLUDE patterns using gitignore rules
    for _, p := range paths {
        if shouldExclude(p, alsoExclude) {
            ignored[p] = true
        }
    }

    // Second pass: use git check-ignore for .gitignore rules
    // Only check files that weren't already excluded
    var toCheck []string
    for _, p := range paths {
        if !ignored[p] {
            toCheck = append(toCheck, p)
        }
    }

    if len(toCheck) > 0 {
        cmd := exec.Command("git", "check-ignore", "--stdin")
        stdin, _ := cmd.StdinPipe()

        go func() {
            for _, p := range toCheck {
                fmt.Fprintln(stdin, p)
            }
            stdin.Close()
        }()

        output, _ := cmd.Output()
        for _, line := range strings.Split(string(output), "\n") {
            if line != "" {
                ignored[line] = true
            }
        }
    }

    return ignored
}

// shouldExclude checks if a path matches any ALSO_EXCLUDE pattern
// using gitignore-style matching rules
func shouldExclude(path string, patterns []string) bool {
    for _, pattern := range patterns {
        if matchesGitignorePattern(path, pattern) {
            return true
        }
    }
    return false
}

// matchesGitignorePattern implements basic gitignore pattern matching
func matchesGitignorePattern(path, pattern string) bool {
    // Remove leading "./" if present
    path = strings.TrimPrefix(path, "./")

    // Handle directory patterns (ending with /)
    if strings.HasSuffix(pattern, "/") {
        dirPattern := strings.TrimSuffix(pattern, "/")
        // Match if path is in this directory or is the directory itself
        return path == dirPattern || strings.HasPrefix(path, dirPattern+"/")
    }

    // Handle wildcard patterns like *.pyc
    if strings.Contains(pattern, "*") {
        // Match against basename for simple wildcards
        matched, _ := filepath.Match(pattern, filepath.Base(path))
        if matched {
            return true
        }
        // Also try matching the full path for patterns like **/*.pyc
        matched, _ = filepath.Match(pattern, path)
        return matched
    }

    // Exact match or prefix match for simple patterns
    return path == pattern || strings.HasPrefix(path, pattern+"/")
}

func filterIncluded(all []string, ignored map[string]bool, alwaysInclude []string) []string {
    var included []string
    for _, f := range all {
        // Check if file should always be included
        shouldInclude := false
        for _, pattern := range alwaysInclude {
            if matchesGitignorePattern(f, pattern) {
                shouldInclude = true
                break
            }
        }

        // Include if not ignored OR if it's in alwaysInclude
        if shouldInclude || !ignored[f] {
            included = append(included, f)
        }
    }
    sort.Strings(included)
    return included
}

func printTree(paths []string, symbols bool) {
    fmt.Println(".")
    fmt.Print(treeString(paths, symbols))
}

func treeString(paths []string, symbols bool) string {
    tree := make(map[string]interface{})
    for _, path := range paths {
        parts := strings.Split(path, string(os.PathSeparator))
        current := tree
        for i, part := range parts {
            if i == len(parts)-1 {
                current[part] = nil
            } else {
                if _, ok := current[part]; !ok {
                    current[part] = make(map[string]interface{})
                }
                current = current[part].(map[string]interface{})
            }
        }
    }
    var buf bytes.Buffer
    renderTree(&buf, tree, "", symbols)
    return buf.String()
}

func renderTree(buf *bytes.Buffer, node map[string]interface{}, prefix string, symbols bool) {
    keys := make([]string, 0, len(node))
    for k := range node {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    for i, k := range keys {
        last := i == len(keys)-1
        conn := "├── "
        nextPrefix := prefix + "│   "
        if last {
            conn = "└── "
            nextPrefix = prefix + "    "
        }
        if node[k] == nil {
            if symbols {
                buf.WriteString(fmt.Sprintf("%s%s%s\n", prefix, conn, k))
            } else {
                buf.WriteString(fmt.Sprintf("%s    %s\n", prefix, k))
            }
        } else {
            if symbols {
                buf.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, conn, k))
            } else {
                buf.WriteString(fmt.Sprintf("%s    %s/\n", prefix, k))
            }
            renderTree(buf, node[k].(map[string]interface{}), nextPrefix, symbols)
        }
    }
}