package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RSS 数据结构定义
type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title string `xml:"title"`
	Items []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
}

func main() {
	http.HandleFunc("/process-rss", processRSSHandler)
	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)
}

func processRSSHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从查询参数中获取 RSS URL
	rssURL := r.URL.Query().Get("url")
	if rssURL == "" {
		http.Error(w, "Missing RSS URL parameter", http.StatusBadRequest)
		return
	}

	// 获取并解析 RSS 内容
	content, err := fetchAndParseRSS(rssURL)
	if err != nil {
		http.Error(w, "Error processing RSS feed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(content))
}

func fetchAndParseRSS(url string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return "", fmt.Errorf("failed to parse RSS: %w", err)
	}

	return formatContent(rss), nil
}

func formatContent(rss RSS) string {
	var builder strings.Builder

	// 添加频道标题
	builder.WriteString(fmt.Sprintf("频道: %s\n\n", rss.Channel.Title))

	// 格式化每个条目
	for i, item := range rss.Channel.Items {
		builder.WriteString(fmt.Sprintf("条目 %d:\n", i+1))
		builder.WriteString(fmt.Sprintf("标题: %s\n", strings.TrimSpace(item.Title)))
		builder.WriteString(fmt.Sprintf("描述: %s\n", strings.TrimSpace(item.Description)))
		builder.WriteString(fmt.Sprintf("链接: %s\n", strings.TrimSpace(item.Link)))

		// 添加分隔线（最后一个条目不加）
		if i < len(rss.Channel.Items)-1 {
			builder.WriteString("\n────────────────────────\n\n")
		}
	}

	return builder.String()
}
