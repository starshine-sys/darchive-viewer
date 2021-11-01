package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"codeberg.org/evieDelta/darchive/darchivev3"
	"github.com/bwmarrin/discordgo"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/go-chi/chi/v5"
	"github.com/starshine-sys/dischtml"
)

//go:embed index.html
var page string

const urlFormat = "https://cdn.discordapp.com/attachments/%v/%v/%v"

var tmpl = template.Must(template.New("").Parse(page))

func serve(w http.ResponseWriter, r *http.Request) {
	var (
		channelID    = chi.URLParam(r, "channelID")
		attachmentID = chi.URLParam(r, "attachmentID")
		attachment   = chi.URLParam(r, "attachment")
	)

	// quick sanity check--yes this is Probably™ security through obscurity, i don't care
	if !strings.HasSuffix(attachment, ".json") {
		log.Printf("Attachment name %v didn't end in .json!", attachment)
		fmt.Fprintf(w, "Attachment name should end in .json!")
		return
	}

	discordURL := fmt.Sprintf(urlFormat, channelID, attachmentID, attachment)

	req, err := http.NewRequestWithContext(r.Context(), "GET", discordURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		fmt.Fprintf(w, "Unknown error while creating the request.")
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error doing request: %v", err)
		fmt.Fprintf(w, "Unknown error while doing request.")
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 404:
		fmt.Fprintf(w, "Attachment not found.")
		return
	case 429:
		fmt.Fprintf(w, "Hit Discord rate limit.")
		return
	default:
		switch {
		case resp.StatusCode >= 300:
			fmt.Fprintf(w, "%v error code", resp.StatusCode)
			return
		}
	}

	var doc darchivev3.ArchiveData
	err = json.NewDecoder(resp.Body).Decode(&doc)
	if err != nil {
		log.Printf("Error decoding file: %v", err)
		fmt.Fprintf(w, "Error decoding file: %v", err)
		return
	}

	c := &dischtml.Converter{}

	chID, _ := discord.ParseSnowflake(doc.Channel.ID)
	ch := discord.Channel{
		ID:   discord.ChannelID(chID),
		Name: doc.Channel.Name,
	}

	msgs := msgsToArikawa(c, doc.Messages)

	s, err := c.ConvertHTML(msgs)
	if err != nil {
		log.Printf("Error converting to HTML: %v", err)
		fmt.Fprintf(w, "Unknown error converting to HTML.")
		return
	}

	data := struct {
		HighlightCSS, CSS template.CSS
		HighlightJS       template.JS
		Channel           discord.Channel
		Content           template.HTML
		MsgCount          int
	}{CSS: template.CSS(dischtml.CSS), HighlightCSS: template.CSS(dischtml.HighlightCSS), HighlightJS: template.JS(dischtml.HighlightJS), Channel: ch, Content: s, MsgCount: len(msgs)}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		fmt.Fprintf(w, "Unknown error executing template.")
		return
	}
}

func msgsToArikawa(c *dischtml.Converter, msgs []*darchivev3.Message) []discord.Message {
	dmsgs := make([]discord.Message, len(msgs))
	for i := range msgs {
		dmsgs[i] = msgToArikawa(c, msgs[i])
	}
	return dmsgs
}

func msgToArikawa(c *dischtml.Converter, m *darchivev3.Message) discord.Message {
	webhookID, _ := discord.ParseSnowflake(m.WebhookID)
	messageID, _ := discord.ParseSnowflake(m.ID)

	embeds := make([]discord.Embed, len(m.Embeds))
	for i := range m.Embeds {
		embeds[i] = embedToArikawa(m.Embeds[i])
	}

	mentions := make([]discord.GuildUser, len(m.MentionUsers))
	for i := range m.MentionUsers {
		u := userToArikawa(m.MentionUsers[i])
		mentions[i] = discord.GuildUser{User: u}
		c.Users = append(c.Users, u)
	}

	return discord.Message{
		Content:   m.Content,
		ID:        discord.MessageID(messageID),
		WebhookID: discord.WebhookID(webhookID),
		Author:    userToArikawa(m.Author),
		Mentions:  mentions,
		Embeds:    embeds,
	}
}

func userToArikawa(u *darchivev3.User) discord.User {
	id, _ := discord.ParseSnowflake(u.ID)

	return discord.User{
		ID:            discord.UserID(id),
		Username:      u.Username,
		Discriminator: u.Discriminator,
		Bot:           u.Bot,
		Avatar:        u.Avatar,
	}
}

func embedToArikawa(e *discordgo.MessageEmbed) discord.Embed {
	t, _ := discordgo.Timestamp(e.Timestamp).Parse()

	de := discord.Embed{
		Title:       e.Title,
		Type:        discord.EmbedType(e.Type),
		Description: e.Description,
		URL:         e.URL,
		Timestamp:   discord.Timestamp(t),
		Color:       discord.Color(e.Color),
	}

	if e.Footer != nil {
		de.Footer = &discord.EmbedFooter{
			Text:      e.Footer.Text,
			Icon:      e.Footer.IconURL,
			ProxyIcon: e.Footer.ProxyIconURL,
		}
	}

	if e.Image != nil {
		de.Image = &discord.EmbedImage{
			URL:    e.Image.URL,
			Proxy:  e.Image.ProxyURL,
			Height: uint(e.Image.Height),
			Width:  uint(e.Image.Width),
		}
	}

	if e.Thumbnail != nil {
		de.Thumbnail = &discord.EmbedThumbnail{
			URL:    e.Thumbnail.URL,
			Proxy:  e.Thumbnail.ProxyURL,
			Height: uint(e.Thumbnail.Height),
			Width:  uint(e.Thumbnail.Width),
		}
	}

	if e.Video != nil {
		de.Video = &discord.EmbedVideo{
			URL:    e.Video.URL,
			Height: uint(e.Video.Height),
			Width:  uint(e.Video.Width),
		}
	}

	if e.Provider != nil {
		de.Provider = &discord.EmbedProvider{
			Name: e.Provider.Name,
			URL:  e.Provider.URL,
		}
	}

	if e.Author != nil {
		de.Author = &discord.EmbedAuthor{
			Name:      e.Author.Name,
			Icon:      e.Author.IconURL,
			URL:       e.Author.URL,
			ProxyIcon: e.Author.ProxyIconURL,
		}
	}

	for _, f := range e.Fields {
		de.Fields = append(de.Fields, discord.EmbedField{
			Name:   f.Name,
			Value:  f.Value,
			Inline: f.Inline,
		})
	}

	return de
}
