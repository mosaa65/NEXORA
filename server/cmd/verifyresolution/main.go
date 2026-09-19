// Command verifyresolution runs /api/index against a real directory and reports
// whether the Entity Resolution pipeline actually prevents degenerate works.
//
// This is the end-to-end check for the fix that wired ResolveAndIngest into the
// scan path. Before that fix the probe passed but the API did not, because the
// API still stored each filename as a work.
//
// Usage: go run ./cmd/verifyresolution
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	apiBase = "http://127.0.0.1:8080"
	root    = "C:/Users/mousa/Desktop/M/NEXORA_STRESS_ESTIRAHAT_V2/استراحة رئيسية"
	// serverEnvFile is the name the server itself loads; the helper never writes it.
	serverEnvFile = ".env"
)

// readEnvValue reads one key from the server's .env without interpreting it.
// The file is read only to authenticate; nothing is written to it.
//
// Both candidate locations are tried because the helper may be run from the
// repository root or from server/, and the server itself prefers ./env relative
// to its own working directory.
func readEnvValue(key string) string {
	for _, path := range []string{serverEnvFile, "server/" + serverEnvFile} {
		if value := readEnvFile(path, key); value != "" {
			return value
		}
	}
	return ""
}

func readEnvFile(file, key string) string {
	raw, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.Trim(strings.TrimPrefix(line, key+"="), "\"' \r\n")
		}
	}
	return ""
}

func main() {
	client := &http.Client{Timeout: 15 * time.Minute}

	loginBody, _ := json.Marshal(map[string]string{
		"username": readEnvValue("NEXORA_ADMIN_USER"),
		"password": readEnvValue("NEXORA_ADMIN_PASS"),
	})
	resp, err := client.Post(apiBase+"/api/admin/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		fmt.Println("login transport failed:", err)
		os.Exit(1)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var login struct {
		Token string `json:"token"`
		Ok    bool   `json:"ok"`
	}
	_ = json.Unmarshal(raw, &login)
	if login.Token == "" {
		fmt.Printf("login failed (status %d): %s\n", resp.StatusCode, string(raw))
		os.Exit(1)
	}
	fmt.Println("authenticated OK")

	// Run a full scan so every file is re-evaluated through resolution.
	payload, _ := json.Marshal(map[string]any{
		"roots":      []string{root},
		"mode":       "full",
		"inspect":    false,
		"syncSearch": false,
	})
	req, _ := http.NewRequest(http.MethodPost, apiBase+"/api/index", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+login.Token)

	started := time.Now()
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println("scan request failed:", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var result struct {
		Status          string   `json:"status"`
		Mode            string   `json:"mode"`
		Scanned         int      `json:"scanned"`
		Imported        int      `json:"imported"`
		WorksCreated    int      `json:"worksCreated"`
		QueuedForReview int      `json:"queuedForReview"`
		Unresolved      int      `json:"unresolved"`
		AliasesLearned  int      `json:"aliasesLearned"`
		Warnings        []string `json:"warnings"`
		Report          *struct {
			NewFiles       int64 `json:"newFiles"`
			ModifiedFiles  int64 `json:"modifiedFiles"`
			UnchangedFiles int64 `json:"unchangedFiles"`
			Accepted       int64 `json:"acceptedMedia"`
			Rejected       int64 `json:"rejectedFiles"`
			ErrorKinds     []any `json:"errors"`
		} `json:"report"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("raw response:", string(body))
		os.Exit(1)
	}

	fmt.Printf("\n=== /api/index RESULT (status %d) ===\n", resp.StatusCode)
	fmt.Printf("mode              : %s\n", result.Mode)
	fmt.Printf("status            : %s\n", result.Status)
	fmt.Printf("scanned           : %d\n", result.Scanned)
	fmt.Printf("imported/attached : %d\n", result.Imported)
	fmt.Printf("works created     : %d\n", result.WorksCreated)
	fmt.Printf("queued for review : %d\n", result.QueuedForReview)
	fmt.Printf("unresolved        : %d\n", result.Unresolved)
	fmt.Printf("aliases learned   : %d\n", result.AliasesLearned)
	if result.Report != nil {
		fmt.Printf("new/modified/unch : %d / %d / %d\n",
			result.Report.NewFiles, result.Report.ModifiedFiles, result.Report.UnchangedFiles)
		fmt.Printf("accepted/rejected : %d / %d\n", result.Report.Accepted, result.Report.Rejected)
	}
	if len(result.Warnings) > 0 {
		fmt.Printf("warnings          : %v\n", result.Warnings)
	}
	fmt.Printf("wall clock        : %s\n", time.Since(started).Round(time.Millisecond))
}
