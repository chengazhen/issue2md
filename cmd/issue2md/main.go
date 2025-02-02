package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bigwhite/issue2md/internal/converter"
	"github.com/bigwhite/issue2md/internal/github"
)

func usage() {
	fmt.Println("Usage: issue2md [issue-url | -f urls-file | -r repo-url] [output-dir]")
	fmt.Println("Arguments:")
	fmt.Println("  issue-url    The URL of the github issue to convert")
	fmt.Println("  -f urls-file A file containing GitHub issue URLs (one per line)")
	fmt.Println("  -r repo-url  The GitHub repository URL to fetch all issues")
	fmt.Println("  output-dir   (optional) The output directory for markdown files (default: downloads)")
}

func parseRepoURL(repoURL string) (owner string, repo string, err error) {
	// Handle URLs like:
	// https://github.com/owner/repo
	// https://github.com/owner/repo/
	// github.com/owner/repo
	repoURL = strings.TrimSuffix(repoURL, "/")
	parts := strings.Split(repoURL, "/")

	// Find the position of "github.com" in the URL
	githubIndex := -1
	for i, part := range parts {
		if part == "github.com" {
			githubIndex = i
			break
		}
	}

	if githubIndex == -1 || githubIndex+2 >= len(parts) {
		return "", "", fmt.Errorf("invalid repository URL format")
	}

	owner = parts[githubIndex+1]
	repo = parts[githubIndex+2]
	return owner, repo, nil
}

func init() {
	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())
}

func randomDelay() {
	// 生成 0-3 秒的随机延迟
	delay := rand.Float64() * 2
	time.Sleep(time.Duration(delay * float64(time.Second)))
}

func convertIssue(issueURL, outputDir string) error {
	owner, repo, issueNumber, err := github.ParseIssueURL(issueURL)
	if err != nil {
		return fmt.Errorf("error parsing issue URL: %v", err)
	}

	token := os.Getenv("GITHUB_TOKEN")

	// Check if file already exists
	markdownFile := fmt.Sprintf("%s_%s_issue_%d.md", owner, repo, issueNumber)
	if outputDir != "" {
		markdownFile = filepath.Join(outputDir, markdownFile)
	}
	if _, err := os.Stat(markdownFile); err == nil {
		return nil // 直接返回，不打印消息（因为已经在上层打印了）
	}

	issue, err := github.FetchIssue(owner, repo, issueNumber, token)
	if err != nil {
		return fmt.Errorf("error fetching issue: %v", err)
	}

	// Add a random delay between API calls
	randomDelay()

	comments, err := github.FetchComments(owner, repo, issueNumber, token)
	if err != nil {
		return fmt.Errorf("error fetching comments: %v", err)
	}

	markdown := converter.IssueToMarkdown(issue, comments)
	if outputDir != "" {
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("error creating output directory: %v", err)
		}
	}

	file, err := os.Create(markdownFile)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	_, err = io.WriteString(file, markdown)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	fmt.Printf("Issue saved as Markdown in file %s\n", markdownFile)
	return nil
}

func convertAllIssues(repoURL, outputDir string) error {
	owner, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return fmt.Errorf("error parsing repository URL: %v", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Warning: GITHUB_TOKEN not set. API rate limits will be restricted.")
		fmt.Println("To set the token, use: set GITHUB_TOKEN=your_token")
		fmt.Println("You can create a token at: https://github.com/settings/tokens")
		fmt.Println("Press Enter to continue or Ctrl+C to exit...")
		fmt.Scanln() // 等待用户确认
	}

	issues, err := github.FetchAllIssues(owner, repo, token)
	if err != nil {
		return fmt.Errorf("error fetching issues: %v", err)
	}

	// Create output directory if it doesn't exist
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("error creating output directory: %v", err)
		}
	}

	totalIssues := len(issues)
	fmt.Printf("Found %d issues in repository %s/%s\n", totalIssues, owner, repo)

	// 预先检查需要下载的文件数量
	needDownload := 0
	for _, issue := range issues {
		markdownFile := fmt.Sprintf("%s_%s_issue_%d.md", owner, repo, issue.Number)
		if outputDir != "" {
			markdownFile = filepath.Join(outputDir, markdownFile)
		}
		if _, err := os.Stat(markdownFile); err != nil {
			needDownload++
		}
	}

	// 计算预估的 API 请求数
	estimatedRequests := needDownload*2 + 1 // 每个需要下载的 issue 2个请求，加上获取列表的1个请求
	if token == "" && estimatedRequests > 60 {
		fmt.Printf("Warning: This operation will require approximately %d API requests.\n", estimatedRequests)
		fmt.Println("Without a token, you are limited to 60 requests per hour.")
		fmt.Println("Please set GITHUB_TOKEN to avoid rate limiting.")
		fmt.Printf("Found %d issues that need to be downloaded.\n", needDownload)
		fmt.Println("Press Enter to continue with the first 30 new issues, or Ctrl+C to exit...")
		fmt.Scanln() // 等待用户确认

		// 限制处理的 issue 数量到前 30 个需要下载的
		if needDownload > 30 {
			fmt.Println("Limiting to first 30 new issues due to API rate limits.")
			downloadCount := 0
			filteredIssues := make([]github.Issue, 0)
			for _, issue := range issues {
				markdownFile := fmt.Sprintf("%s_%s_issue_%d.md", owner, repo, issue.Number)
				if outputDir != "" {
					markdownFile = filepath.Join(outputDir, markdownFile)
				}
				if _, err := os.Stat(markdownFile); err != nil {
					filteredIssues = append(filteredIssues, issue)
					downloadCount++
					if downloadCount >= 30 {
						break
					}
				}
			}
			issues = filteredIssues
		}
	}

	successCount := 0
	skipCount := 0
	errorCount := 0

	for i, issue := range issues {
		issueURL := fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, issue.Number)

		// 预先检查文件是否存在
		markdownFile := fmt.Sprintf("%s_%s_issue_%d.md", owner, repo, issue.Number)
		if outputDir != "" {
			markdownFile = filepath.Join(outputDir, markdownFile)
		}
		if _, err := os.Stat(markdownFile); err == nil {
			fmt.Printf("[%d/%d] Skipping existing file: %s\n", i+1, len(issues), markdownFile)
			skipCount++
			continue
		}

		fmt.Printf("[%d/%d] Converting issue #%d: %s\n", i+1, len(issues), issue.Number, issue.Title)

		if err := convertIssue(issueURL, outputDir); err != nil {
			fmt.Printf("Error converting issue %s: %v\n", issueURL, err)
			if strings.Contains(err.Error(), "403") {
				fmt.Println("Rate limit exceeded. Please wait an hour or set GITHUB_TOKEN.")
				return fmt.Errorf("rate limit exceeded")
			}
			// Wait a bit longer if we encounter an error (might be rate limiting)
			time.Sleep(10 * time.Second)
			errorCount++
			continue
		}

		successCount++

		// Add random delay between issues
		if i < len(issues)-1 { // 不在最后一个 issue 后添加延迟
			randomDelay()
		}
	}

	// 打印统计信息
	fmt.Printf("\nDownload Summary:\n")
	fmt.Printf("Total issues found: %d\n", totalIssues)
	fmt.Printf("Successfully downloaded: %d\n", successCount)
	fmt.Printf("Skipped (already exist): %d\n", skipCount)
	fmt.Printf("Failed: %d\n", errorCount)

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: issue-url, urls-file, or repo-url is required")
		usage()
		return
	}

	// Set default output directory to "downloads"
	outputDir := "downloads"

	switch os.Args[1] {
	case "-f":
		if len(os.Args) < 3 {
			fmt.Println("Error: urls-file is required with -f flag")
			usage()
			return
		}

		urlsFile := os.Args[2]
		if len(os.Args) >= 4 {
			outputDir = os.Args[3]
		}

		file, err := os.Open(urlsFile)
		if err != nil {
			fmt.Printf("Error opening urls file: %v\n", err)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			issueURL := strings.TrimSpace(scanner.Text())
			if issueURL == "" || strings.HasPrefix(issueURL, "#") {
				continue
			}

			fmt.Printf("Converting issue: %s\n", issueURL)
			if err := convertIssue(issueURL, outputDir); err != nil {
				fmt.Printf("Error converting issue %s: %v\n", issueURL, err)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading urls file: %v\n", err)
		}

	case "-r":
		if len(os.Args) < 3 {
			fmt.Println("Error: repo-url is required with -r flag")
			usage()
			return
		}

		repoURL := os.Args[2]
		if len(os.Args) >= 4 {
			outputDir = os.Args[3]
		}

		if err := convertAllIssues(repoURL, outputDir); err != nil {
			fmt.Printf("Error: %v\n", err)
		}

	default:
		if len(os.Args) >= 3 {
			outputDir = os.Args[2]
		}

		if err := convertIssue(os.Args[1], outputDir); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}
