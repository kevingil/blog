package tools

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"backend/pkg/types"
)

type SourceContextResource struct {
	SourceID          string `json:"source_id"`
	Title             string `json:"title"`
	URL               string `json:"url,omitempty"`
	SourceType        string `json:"source_type"`
	OriginTool        string `json:"origin_tool,omitempty"`
	OriginQuery       string `json:"origin_query,omitempty"`
	OriginQuestion    string `json:"origin_question,omitempty"`
	Author            string `json:"author,omitempty"`
	PublishedDate     string `json:"published_date,omitempty"`
	UsageStatus       string `json:"usage_status,omitempty"`
	Preview           string `json:"preview,omitempty"`
	SelectedExcerpt   string `json:"selected_excerpt,omitempty"`
	SelectedExcerptID string `json:"selected_excerpt_id,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
}

func BuildSourceContextResources(sources []types.Source) []SourceContextResource {
	resources := make([]SourceContextResource, 0, len(sources))
	for _, src := range sources {
		resourceMeta := getResourceMeta(src.MetaData)
		preview := strings.TrimSpace(firstNonEmpty(
			asString(resourceMeta["selected_excerpt"]),
			src.Content,
		))
		if preview != "" && len(preview) > 220 {
			preview = preview[:220] + "..."
		}

		resources = append(resources, SourceContextResource{
			SourceID:          src.ID.String(),
			Title:             src.Title,
			URL:               src.URL,
			SourceType:        src.SourceType,
			OriginTool:        asString(resourceMeta["origin_tool"]),
			OriginQuery:       asString(resourceMeta["origin_query"]),
			OriginQuestion:    asString(resourceMeta["origin_question"]),
			Author:            asString(resourceMeta["author"]),
			PublishedDate:     asString(resourceMeta["published_date"]),
			UsageStatus:       asString(resourceMeta["usage_status"]),
			Preview:           preview,
			SelectedExcerpt:   asString(resourceMeta["selected_excerpt"]),
			SelectedExcerptID: asString(resourceMeta["selected_excerpt_id"]),
			CreatedAt:         src.CreatedAt.Format(time.RFC3339),
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].CreatedAt > resources[j].CreatedAt
	})

	return resources
}

func FilterSelectedSourceResources(resources []SourceContextResource) []SourceContextResource {
	selected := make([]SourceContextResource, 0, len(resources))
	for _, resource := range resources {
		if resource.SelectedExcerpt == "" {
			continue
		}
		status := strings.ToLower(resource.UsageStatus)
		if status == "" || status == "selected" || status == "used" {
			selected = append(selected, resource)
		}
	}
	return selected
}

func FormatSourceInventoryContext(resources []SourceContextResource) string {
	if len(resources) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Available Sources:\n")
	for _, resource := range resources {
		line := fmt.Sprintf("- [%s] %s", resource.SourceID, firstNonEmpty(resource.Title, "(untitled source)"))
		if resource.URL != "" {
			line += fmt.Sprintf(" | %s", resource.URL)
		}
		if resource.SourceType != "" {
			line += fmt.Sprintf(" | type=%s", resource.SourceType)
		}
		if resource.OriginTool != "" {
			line += fmt.Sprintf(" | via=%s", resource.OriginTool)
		}
		if resource.UsageStatus != "" {
			line += fmt.Sprintf(" | status=%s", resource.UsageStatus)
		}
		b.WriteString(line)
		b.WriteString("\n")
		if resource.Preview != "" {
			b.WriteString("  preview: ")
			b.WriteString(resource.Preview)
			b.WriteString("\n")
		}
	}

	return strings.TrimSpace(b.String())
}

func FormatSelectedSourcesContext(resources []SourceContextResource) string {
	if len(resources) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Selected Sources For This Edit:\n")
	for _, resource := range resources {
		b.WriteString(fmt.Sprintf("- [%s] %s\n", resource.SourceID, firstNonEmpty(resource.Title, "(untitled source)")))
		if resource.URL != "" {
			b.WriteString("  url: ")
			b.WriteString(resource.URL)
			b.WriteString("\n")
		}
		if resource.SelectedExcerptID != "" {
			b.WriteString("  excerpt_id: ")
			b.WriteString(resource.SelectedExcerptID)
			b.WriteString("\n")
		}
		b.WriteString("  excerpt:\n")
		b.WriteString(resource.SelectedExcerpt)
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}

func getResourceMeta(meta map[string]interface{}) map[string]interface{} {
	if meta == nil {
		return map[string]interface{}{}
	}
	if resourceMeta, ok := meta["resource"].(map[string]interface{}); ok {
		return resourceMeta
	}
	return map[string]interface{}{}
}

func asString(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
