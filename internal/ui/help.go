package ui

import (
	"fmt"
	"strings"
)

func ColorHeadings(text string) string {
	headings := []string{
		"Usage:",
		"Examples:",
		"Available Commands:",
		"Flags:",
		"Aliases:",
		"Additional Commands:",
	}

	// Replace each heading with its colorized version.
	for _, heading := range headings {
		text = strings.ReplaceAll(text, heading, fmt.Sprintf("%s%s%s%s", FgBlue, Bold, heading, Reset))
	}

	return text
}
