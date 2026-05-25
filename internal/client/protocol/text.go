package protocol

import "strings"

// StripCardURLLines removes lines that contain only a googleusercontent card URL placeholder.
// It cleans assistant response text while preserving normal text lines.
func StripCardURLLines(text string) string {
	lines := strings.Split(text, "\n")
	var kept []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "http://googleusercontent.com/card_content/") ||
			strings.HasPrefix(trimmed, "http://googleusercontent.com/video_gen_chip/") ||
			strings.HasPrefix(trimmed, "http://googleusercontent.com/generated_video_content/") ||
			strings.HasPrefix(trimmed, "http://googleusercontent.com/generated_media_content/") ||
			strings.HasPrefix(trimmed, "http://googleusercontent.com/generated_music_content/") {
			continue
		}
		kept = append(kept, line)
	}
	result := strings.Join(kept, "\n")
	return strings.TrimRight(result, "\n")
}
