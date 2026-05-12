package export

import "encoding/json"

type Conversation struct {
	UUID         string          `json:"uuid"`
	Name         string          `json:"name"`
	Summary      string          `json:"summary"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
	Account      json.RawMessage `json:"account"`
	ChatMessages []Message       `json:"chat_messages"`
}

type Message struct {
	UUID              string         `json:"uuid"`
	ParentMessageUUID string         `json:"parent_message_uuid"`
	Sender            string         `json:"sender"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
	Text              string         `json:"text"`
	Content           []ContentBlock `json:"content"`
	Attachments       []Attachment   `json:"attachments"`
	Files             []FileRef      `json:"files"`
}

type ContentBlock struct {
	Type string `json:"type"`

	Text     string          `json:"text,omitempty"`
	Thinking string          `json:"thinking,omitempty"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`

	// ID は tool_use ブロックの一意 id。tool_result.tool_use_id と対応する。
	ID string `json:"id,omitempty"`

	ToolUseID         string          `json:"tool_use_id,omitempty"`
	ResultContent     json.RawMessage `json:"content,omitempty"`
	StructuredContent json.RawMessage `json:"structured_content,omitempty"`
	DisplayContent    json.RawMessage `json:"display_content,omitempty"`
	IsError           bool            `json:"is_error,omitempty"`
}

type Attachment struct {
	FileName         string `json:"file_name"`
	FileSize         int64  `json:"file_size"`
	FileType         string `json:"file_type"`
	ExtractedContent string `json:"extracted_content"`
}

type FileRef struct {
	FileName string `json:"file_name"`
	FileUUID string `json:"file_uuid"`
}
