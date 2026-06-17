package metadata

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func Extract(raw RawMetadata) ExtractedMetadata {
	out := ExtractedMetadata{
		Source:   SourceUnknown,
		Raw:      copyRaw(raw),
		Settings: map[string]string{},
	}
	if len(raw) == 0 {
		return out
	}

	jsonMaps := collectJSONMaps(raw)
	if isComfyUI(raw, jsonMaps) {
		out.Source = SourceComfyUI
		out.Workflow = summarizeComfyWorkflow(raw, jsonMaps)
		positive, negative := comfyPrompts(jsonMaps)
		out.PositivePrompt = firstNonEmpty(positive, comfyPositivePrompt(jsonMaps), raw["Description"])
		out.NegativePrompt = firstNonEmpty(negative, raw["negative_prompt"], raw["uc"])
		out.Settings = collectSettings(raw, jsonMaps)
		out.Tags = promptTags(out.PositivePrompt, "positive")
		return out
	}

	if isNovelAI(raw, jsonMaps) {
		out.Source = SourceNAI
		out.PositivePrompt = firstNonEmpty(raw["Description"], raw["prompt"], lookupJSON(jsonMaps, "prompt"))
		out.NegativePrompt = firstNonEmpty(raw["uc"], lookupJSON(jsonMaps, "uc"), raw["negative_prompt"])
		out.Settings = collectSettings(raw, jsonMaps)
		out.Tags = promptTags(out.PositivePrompt, "positive")
		return out
	}

	out.PositivePrompt = firstNonEmpty(raw["Description"], raw["prompt"], lookupJSON(jsonMaps, "prompt"))
	out.NegativePrompt = firstNonEmpty(raw["uc"], raw["negative_prompt"], lookupJSON(jsonMaps, "uc"))
	out.Settings = collectSettings(raw, jsonMaps)
	out.Tags = promptTags(out.PositivePrompt, "positive")
	if out.PositivePrompt != "" || out.NegativePrompt != "" || len(out.Settings) > 0 {
		out.Source = SourceGeneric
	}
	return out
}

func copyRaw(raw RawMetadata) RawMetadata {
	out := RawMetadata{}
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func isNovelAI(raw RawMetadata, jsonMaps []map[string]any) bool {
	if strings.Contains(strings.ToLower(raw["Software"]), "novelai") {
		return true
	}
	for _, key := range []string{"uc", "sampler", "scale"} {
		if raw[key] != "" {
			return true
		}
		if lookupJSON(jsonMaps, key) != "" {
			return true
		}
	}
	return false
}

func isComfyUI(raw RawMetadata, jsonMaps []map[string]any) bool {
	for _, key := range []string{"prompt", "workflow"} {
		if looksLikeComfyJSON(raw[key]) {
			return true
		}
	}
	for _, value := range jsonMaps {
		if comfyNodeCount(value) > 0 {
			return true
		}
	}
	return false
}

func looksLikeComfyJSON(text string) bool {
	var value map[string]any
	if json.Unmarshal([]byte(text), &value) != nil {
		return false
	}
	return comfyNodeCount(value) > 0
}

func collectJSONMaps(raw RawMetadata) []map[string]any {
	var maps []map[string]any
	for _, value := range raw {
		var parsed map[string]any
		if json.Unmarshal([]byte(value), &parsed) == nil {
			maps = append(maps, parsed)
		}
	}
	return maps
}

func lookupJSON(maps []map[string]any, key string) string {
	for _, value := range maps {
		if found, ok := lookupJSONValue(value, key); ok {
			return stringify(found)
		}
	}
	return ""
}

func lookupJSONValue(value any, key string) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		if found, ok := typed[key]; ok {
			return found, true
		}
		for _, child := range typed {
			if found, ok := lookupJSONValue(child, key); ok {
				return found, true
			}
		}
	case []any:
		for _, child := range typed {
			if found, ok := lookupJSONValue(child, key); ok {
				return found, true
			}
		}
	}
	return nil, false
}

func collectSettings(raw RawMetadata, maps []map[string]any) map[string]string {
	settings := map[string]string{}
	for _, key := range []string{"seed", "sampler", "sampler_name", "scale", "cfg_scale", "steps", "scheduler"} {
		if value := raw[key]; value != "" {
			settings[normalizeSettingKey(key)] = value
			continue
		}
		if value := lookupJSON(maps, key); value != "" {
			settings[normalizeSettingKey(key)] = value
		}
	}
	return settings
}

func normalizeSettingKey(key string) string {
	if key == "sampler_name" {
		return "sampler"
	}
	if key == "cfg_scale" {
		return "scale"
	}
	return key
}

func summarizeComfyWorkflow(raw RawMetadata, maps []map[string]any) WorkflowSummary {
	summary := WorkflowSummary{}
	for _, key := range []string{"prompt", "workflow"} {
		var parsed map[string]any
		if json.Unmarshal([]byte(raw[key]), &parsed) == nil {
			mergeWorkflowSummary(&summary, parsed)
		}
	}
	for _, parsed := range maps {
		mergeWorkflowSummary(&summary, parsed)
	}
	summary.Models = sortedUnique(summary.Models)
	summary.Checkpoints = sortedUnique(summary.Checkpoints)
	summary.Samplers = sortedUnique(summary.Samplers)
	summary.Schedulers = sortedUnique(summary.Schedulers)
	return summary
}

func mergeWorkflowSummary(summary *WorkflowSummary, value map[string]any) {
	nodes := comfyNodes(value)
	if len(nodes) > summary.NodeCount {
		summary.NodeCount = len(nodes)
	}
	for _, node := range nodes {
		inputs, _ := node["inputs"].(map[string]any)
		for key, rawValue := range inputs {
			text := stringify(rawValue)
			if text == "" {
				continue
			}
			lower := strings.ToLower(key)
			switch {
			case strings.Contains(lower, "ckpt") || strings.Contains(lower, "checkpoint"):
				summary.Checkpoints = append(summary.Checkpoints, text)
			case strings.Contains(lower, "model"):
				summary.Models = append(summary.Models, text)
			case strings.Contains(lower, "sampler"):
				summary.Samplers = append(summary.Samplers, text)
			case strings.Contains(lower, "scheduler"):
				summary.Schedulers = append(summary.Schedulers, text)
			}
		}
	}
}

func comfyNodeCount(value map[string]any) int {
	return len(comfyNodes(value))
}

func comfyNodes(value map[string]any) []map[string]any {
	if rawNodes, ok := value["nodes"].([]any); ok {
		var nodes []map[string]any
		for _, rawNode := range rawNodes {
			if node, ok := rawNode.(map[string]any); ok {
				nodes = append(nodes, node)
			}
		}
		return nodes
	}

	var nodes []map[string]any
	for _, rawNode := range value {
		node, ok := rawNode.(map[string]any)
		if !ok {
			continue
		}
		if _, hasClass := node["class_type"]; hasClass {
			nodes = append(nodes, node)
			continue
		}
		if _, hasInputs := node["inputs"]; hasInputs {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func comfyPositivePrompt(maps []map[string]any) string {
	for _, parsed := range maps {
		for _, node := range comfyNodes(parsed) {
			classType := strings.ToLower(stringify(node["class_type"]))
			inputs, _ := node["inputs"].(map[string]any)
			text := stringify(inputs["text"])
			if text == "" {
				continue
			}
			if strings.Contains(classType, "cliptextencode") || classType == "" {
				return text
			}
		}
	}
	return ""
}

func comfyPrompts(maps []map[string]any) (string, string) {
	for _, parsed := range maps {
		nodesByID := map[string]map[string]any{}
		for id, rawNode := range parsed {
			if node, ok := rawNode.(map[string]any); ok {
				nodesByID[id] = node
			}
		}
		for _, node := range nodesByID {
			classType := strings.ToLower(stringify(node["class_type"]))
			if !strings.Contains(classType, "ksampler") {
				continue
			}
			inputs, _ := node["inputs"].(map[string]any)
			positive := linkedText(nodesByID, inputs["positive"])
			negative := linkedText(nodesByID, inputs["negative"])
			if positive != "" || negative != "" {
				return positive, negative
			}
		}
	}
	return "", ""
}

func linkedText(nodesByID map[string]map[string]any, link any) string {
	var id string
	switch typed := link.(type) {
	case []any:
		if len(typed) > 0 {
			id = stringify(typed[0])
		}
	case string:
		id = typed
	}
	if id == "" {
		return ""
	}
	node := nodesByID[id]
	inputs, _ := node["inputs"].(map[string]any)
	return stringify(inputs["text"])
}

func promptTags(prompt, source string) []Tag {
	parts := strings.Split(prompt, ",")
	tags := make([]Tag, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name != "" {
			tags = append(tags, Tag{Name: name, Source: source})
		}
	}
	return tags
}

func sortedUnique(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%g", typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}
