// Purpose: Attach immutable local image bytes using each provider's documented request dialect.
// Pattern: Multimodal request adapter.
// Usage: Messages flushes tool images after their complete native result group.
// Related: internal/provider/protocol/tools.go, internal/provider/protocol/lmstudio.go.
package protocol

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func attachImages(kind connection.ProviderKind, message *Message, images []media.Image) error {
	if len(images) == 0 {
		return nil
	}
	if message.Role != "user" {
		return fmt.Errorf("provider image inputs require a user message")
	}
	text, ok := message.Content.(string)
	if !ok && message.Content != nil {
		return fmt.Errorf("image inputs require an unencoded text message")
	}
	var parts []contentPart
	if kind != connection.Ollama {
		parts = make([]contentPart, 0, 1+2*len(images))
		if text != "" {
			parts = append(parts, contentPart{Type: "text", Text: text})
		}
	}
	for _, image := range images {
		if err := image.Check(); err != nil {
			return err
		}
		label := fmt.Sprintf("Image from read_image: %s (%s, %d x %d)", image.Path, image.Digest, image.Width, image.Height)
		prefix := ""
		if kind != connection.Ollama {
			prefix = "data:" + image.MIME + ";base64,"
		}
		encoded, err := encodeImage(image.Data, prefix)
		if err != nil {
			return err
		}
		if kind == connection.Ollama {
			message.Images = append(message.Images, encoded)
			text = strings.TrimSpace(text + "\n" + label)
		} else {
			parts = append(parts, contentPart{Type: "text", Text: label}, contentPart{Type: "image_url", ImageURL: &imageURL{URL: encoded}})
		}
	}
	message.Content = text
	if kind != connection.Ollama {
		message.Content = parts
	}
	return nil
}

// The builder owns one final-size allocation; the encoder writes directly into
// it instead of allocating separate encoded bytes, a string and a data URL.
func encodeImage(data []byte, prefix string) (string, error) {
	var out strings.Builder
	out.Grow(len(prefix) + base64.StdEncoding.EncodedLen(len(data)))
	out.WriteString(prefix)
	encoder := base64.NewEncoder(base64.StdEncoding, &out)
	if _, err := encoder.Write(data); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func lmContent(message Message) any {
	parts, _ := message.Content.([]contentPart)
	if len(parts) == 0 {
		return message.Content
	}
	content := make([]map[string]any, 0, len(parts))
	for _, part := range parts {
		if part.ImageURL != nil {
			content = append(content, map[string]any{"type": "input_image", "image_url": part.ImageURL.URL})
		} else {
			content = append(content, map[string]any{"type": "input_text", "text": part.Text})
		}
	}
	return content
}
