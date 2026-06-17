package metadata

type SourceType string

const (
	SourceNAI     SourceType = "nai"
	SourceComfyUI SourceType = "comfyui"
	SourceGeneric SourceType = "generic"
	SourceUnknown SourceType = "unknown"
)

type RawMetadata map[string]string

type Tag struct {
	Name       string
	Source     string
	Normalized string
	Kind       string
}

type WorkflowSummary struct {
	NodeCount   int
	Models      []string
	Checkpoints []string
	Samplers    []string
	Schedulers  []string
}

type ExtractedMetadata struct {
	Source         SourceType
	Raw            RawMetadata
	PositivePrompt string
	NegativePrompt string
	Settings       map[string]string
	Tags           []Tag
	Workflow       WorkflowSummary
}

type ImageMetadata struct {
	Format   string
	Width    int
	Height   int
	Metadata []Record
	Tags     []ImageTag
}

type Record struct {
	Source          SourceType
	Raw             map[string]any
	PositivePrompt  string
	NegativePrompt  string
	Settings        map[string]any
	WorkflowSummary map[string]any
}

type ImageTag struct {
	Value      string
	Normalized string
	Source     SourceType
	Kind       string
}
