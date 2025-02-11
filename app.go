package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	_ "embed"
)

// RSS 数据结构定义
type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	ChannelLink string `xml:"link"`
	Items       []Item `xml:"item"`
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

func (c *Channel) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
			switch t.Name.Local {
			case "title":
				var title string
				if err := d.DecodeElement(&title, &t); err != nil {
					return err
				}
				c.Title = title
			case "link":
				// 获取 atom:link 中的 href 属性
				// 或者 link 的值
				var link string
				for _, attr := range t.Attr {
					// More flexible href matching
					if (attr.Name.Local == "href") && attr.Value != "" {
						link = attr.Value
						break
					}
				}
				// If no href attribute found, try decoding the element
				if link == "" {
					if err := d.DecodeElement(&link, &t); err != nil {
						return err
					}
				}

				c.ChannelLink = link

			case "item":
				var item Item
				if err := d.DecodeElement(&item, &t); err != nil {
					return err
				}
				c.Items = append(c.Items, item)
			}
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
}

func main() {
	http.HandleFunc("/process-rss", processRSSHandler)
	http.HandleFunc("/process-html", processHTMLHandler)
	http.HandleFunc("/", serveReadme)
	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)
}

//go:embed README.md
var readmeContent []byte

func serveReadme(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Write(readmeContent)
}

type HTMLParams struct {
	URL         string
	Selector    string
	Separator   string
	StripHTML   bool
	RemoveSpace bool
}

func extractHTMLParams(r *http.Request) (HTMLParams, error) {
	// Extract URL
	htmlURL := r.URL.Query().Get("url")
	if htmlURL == "" {
		return HTMLParams{}, fmt.Errorf("missing HTML URL parameter")
	}

	// Parse URL
	parsedURL, err := url.Parse(htmlURL)
	if err != nil {
		return HTMLParams{}, fmt.Errorf("invalid HTML URL")
	}

	// Extract and set default parameters
	params := HTMLParams{
		URL:         parsedURL.String(),
		Selector:    r.URL.Query().Get("selector"),
		Separator:   r.URL.Query().Get("separator"),
		StripHTML:   r.URL.Query().Get("stripHTML") != "false",
		RemoveSpace: r.URL.Query().Get("removeSpace") != "false",
	}

	// Set default selector
	if params.Selector == "" {
		params.Selector = "body"
	}

	// Set default separator and replace escaped newlines
	if params.Separator == "" {
		params.Separator = "\n\n"
	}
	params.Separator = strings.ReplaceAll(params.Separator, "\\n", "\n")

	return params, nil
}

func getHTMLContent(params HTMLParams) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(params.URL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch HTML: %w", err)
	}
	defer resp.Body.Close()

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var content string
	// Get webpage meta title
	title := doc.Find("title").Text()
	if title != "" {
		content = "website title: " + title + params.Separator
	}

	// Process content based on parameters
	var extractedContent string
	if params.StripHTML {
		extractedContent = doc.Find(params.Selector).Text()
	} else {
		extractedContent, err = doc.Find(params.Selector).Html()
		if err != nil {
			return "", fmt.Errorf("failed to get HTML content: %w", err)
		}
	}

	// Apply additional processing
	if params.RemoveSpace {
		extractedContent = ReplaceAllSpace(extractedContent)
	}

	content += extractedContent

	if content == "" {
		return "", fmt.Errorf("empty HTML content")
	}

	return content, nil
}

func processHTMLHandler(w http.ResponseWriter, r *http.Request) {
	// Validate HTTP method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract and validate parameters
	params, err := extractHTMLParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch and process HTML content
	htmlContent, err := getHTMLContent(params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch HTML: %v", err), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(htmlContent))
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
	removeSpace := r.URL.Query().Get("removeSpace")
	if removeSpace == "" {
		removeSpace = "true" // removeSpace 默认为 true
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
	content, err := fetchAndParseRSS(parsedURL.String(), separatorChar, stripHTMLBool, lengthInt, removeSpace == "true")
	if err != nil {
		http.Error(w, "Error processing RSS feed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(content))
}

func fetchAndParseRSS(url string, separatorChar string, stripHTML bool, length int, removeSpace bool) (string, error) {
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

	return formatContent(rss, separatorChar, stripHTML, length, removeSpace), nil
}

func formatContent(rss RSS, separatorChar string, stripHTML bool, length int, removeSpace bool) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Channel Title: %s\n", rss.Channel.Title))
	if rss.Channel.ChannelLink != "" {
		builder.WriteString(fmt.Sprintf("Channel Link: %s", rss.Channel.ChannelLink))
	}
	builder.WriteString(separatorChar)
	// 格式化每个条目
	for i, item := range rss.Channel.Items {
		if length != 0 && i >= length {
			break
		}
		builder.WriteString(fmt.Sprintf("Channel Item %d:\n", i+1))

		// 使用反射获取所有字段
		keys := make([]string, 0, len(item.Fields))
		for key := range item.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := item.Fields[key]
			var valueStr string
			if stripHTML {
				if removeSpace {
					valueStr = ReplaceAllSpace(strings.TrimSpace(stripHTMLTags(value)))
				} else {
					valueStr = strings.TrimSpace(stripHTMLTags(value))
				}
			} else {
				if removeSpace {
					valueStr = ReplaceAllSpace(strings.TrimSpace(value))
				} else {
					valueStr = strings.TrimSpace(value)
				}
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

func ReplaceAllSpace(input string) string {
	// Remove multiple consecutive whitespace characters, including newlines
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(input, " ")
}
