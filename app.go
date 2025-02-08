package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RSS 数据结构定义
type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title         string `xml:"title"`
	LastBuildDate string `xml:"lastBuildDate"`
	Items         []Item `xml:"item"`
}

type Item struct {
	// 使用 map 存储动态字段
	Fields map[string]string `xml:"-"`
}

// 自定义 UnmarshalXML 方法以支持动态字段
func (i *Item) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// 初始化 Fields map
	i.Fields = make(map[string]string)

	// 循环处理所有 XML 元素
	for {
		token, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			var value string
			err := d.DecodeElement(&value, &t)
			if err != nil {
				return err
			}
			// 将每个元素存储到 Fields map 中，使用元素名作为键
			i.Fields[t.Name.Local] = value

		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
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

	// 处理 URL 中可能存在的额外查询参数
	parsedURL, err := url.Parse(rssURL)
	if err != nil {
		http.Error(w, "Invalid RSS URL", http.StatusBadRequest)
		return
	}

	// 获取分隔线配置
	separatorChar := r.URL.Query().Get("separator")
	if separatorChar == "" {
		separatorChar = "\n\n" // 默认分隔线
	}
	// 获取长度配置
	length := r.URL.Query().Get("length")
	if length == "" {
		length = "0" // 默认不限制长度
	}
	// 转换成数值
	lengthInt, err := strconv.Atoi(length)
	if err != nil {
		http.Error(w, "Invalid length parameter", http.StatusBadRequest)
		return
	}
	// 替换转义的换行符
	separatorChar = strings.ReplaceAll(separatorChar, "\\n", "\n")

	// 获取是否去除HTML标签的配置
	stripHTML := r.URL.Query().Get("stripHTML")
	stripHTMLBool := stripHTML != "false" // 默认为true

	// 获取并解析 RSS 内容
	content, err := fetchAndParseRSS(parsedURL.String(), separatorChar, stripHTMLBool, lengthInt)
	if err != nil {
		http.Error(w, "Error processing RSS feed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(content))
}

func fetchAndParseRSS(url string, separatorChar string, stripHTML bool, length int) (string, error) {
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

	return formatContent(rss, separatorChar, stripHTML, length), nil
}

func formatContent(rss RSS, separatorChar string, stripHTML bool, length int) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Channel Title: %s\n", rss.Channel.Title))
	builder.WriteString(fmt.Sprintf("Channel Last Build Date: %s%s", rss.Channel.LastBuildDate, separatorChar))
	// 格式化每个条目
	for i, item := range rss.Channel.Items {
		if length != 0 && i >= length {
			break
		}
		builder.WriteString(fmt.Sprintf("Channel Item %d:\n", i+1))

		// 使用反射获取所有字段
		for key, value := range item.Fields {
			var valueStr string
			if stripHTML {
				valueStr = strings.TrimSpace(stripHTMLTags(value))
			} else {
				valueStr = strings.TrimSpace(value)
			}
			builder.WriteString(fmt.Sprintf("%s: %s\n", key, valueStr))
		}

		// 添加分隔线（最后一个条目不加）
		if i < len(rss.Channel.Items)-1 {
			builder.WriteString(fmt.Sprintf("%s", strings.Repeat(separatorChar, 1)))
		}
	}

	return builder.String()
}

// stripHTMLTags removes HTML tags from a string
func stripHTMLTags(input string) string {
	// Remove HTML tags
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	return htmlTagRegex.ReplaceAllString(input, "")
}
