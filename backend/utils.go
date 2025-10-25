package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func mustFindProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("获取工作目录失败: %v", err)
	}
	seen := map[string]bool{}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if seen[dir] {
			break
		}
		seen[dir] = true
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	log.Printf("未找到 go.mod，默认使用当前目录: %s", dir)
	return dir
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json: %v", err)
	}
}

func downloadToFile(ctx context.Context, fileURL, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return err
	}

	transport := &http.Transport{DisableKeepAlives: true, ForceAttemptHTTP2: false}
	defer transport.CloseIdleConnections()

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("下载失败: %s", strings.TrimSpace(string(body)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return os.WriteFile(targetPath, data, 0o644)
}

func saveBase64ToFile(encoded, targetPath string) error {
	encoded = strings.TrimSpace(encoded)
	if strings.HasPrefix(encoded, "data:") {
		if idx := strings.Index(encoded, ","); idx != -1 {
			encoded = encoded[idx+1:]
		}
	}
	encoded = strings.NewReplacer("\n", "", "\r", "").Replace(encoded)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("解码 Base64 数据失败: %w", err)
	}
	return os.WriteFile(targetPath, data, 0o644)
}

func removeGeneratedFile(relPath, prefix, dir string) {
	if !strings.HasPrefix(relPath, prefix) {
		return
	}
	filename := strings.TrimPrefix(relPath, prefix)
	if filename == "" {
		return
	}
	absPath := filepath.Join(dir, filename)
	if err := os.Remove(absPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("删除生成文件失败: %v", err)
	}
}
