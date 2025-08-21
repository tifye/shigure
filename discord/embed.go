package discord

import "fmt"

// PreventURLEmbed reformats URLs to prevent discord from embedding their
// OG images went sent in a channel.
func PreventURLEmbed(url string) string {
	return fmt.Sprintf("<%s>", url)
}
