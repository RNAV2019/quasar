package ui

import (
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

func createStyle() ansi.StyleConfig {
	customStyle := styles.DarkStyleConfig

	// Document & Paragraphs
	customStyle.Document.Margin = nil
	customStyle.Document.BlockPrefix = ""
	customStyle.Document.BlockSuffix = ""

	customStyle.Paragraph.Margin = nil
	customStyle.Paragraph.BlockPrefix = ""
	customStyle.Paragraph.BlockSuffix = ""

	// Blockquotes (Keeps the vertical bar '│ ')
	customStyle.BlockQuote.Margin = nil
	customStyle.BlockQuote.BlockPrefix = ""
	customStyle.BlockQuote.BlockSuffix = ""

	// Code & CodeBlocks
	customStyle.CodeBlock.Margin = nil
	customStyle.CodeBlock.BlockPrefix = ""
	customStyle.CodeBlock.BlockSuffix = ""

	customStyle.Code.Margin = nil
	customStyle.Code.BlockPrefix = ""
	customStyle.Code.BlockSuffix = ""

	// Lists (Keeps bullets '• ' and numbers '1. ')
	customStyle.List.Margin = nil
	customStyle.List.BlockPrefix = ""
	customStyle.List.BlockSuffix = ""

	// customStyle.Item.BlockPrefix = ""
	customStyle.Item.BlockSuffix = ""

	// customStyle.Enumeration.BlockPrefix = ""
	customStyle.Enumeration.BlockSuffix = ""

	customStyle.Task.BlockPrefix = ""
	customStyle.Task.BlockSuffix = ""

	// Headings
	customStyle.Heading.Margin = nil
	customStyle.Heading.BlockPrefix = ""
	customStyle.Heading.BlockSuffix = ""

	customStyle.H1.Margin = nil
	customStyle.H1.BlockPrefix = ""
	customStyle.H1.BlockSuffix = ""

	customStyle.H2.Margin = nil
	customStyle.H2.BlockPrefix = ""
	customStyle.H2.BlockSuffix = ""

	customStyle.H3.Margin = nil
	customStyle.H3.BlockPrefix = ""
	customStyle.H3.BlockSuffix = ""

	customStyle.H4.Margin = nil
	customStyle.H4.BlockPrefix = ""
	customStyle.H4.BlockSuffix = ""

	customStyle.H5.Margin = nil
	customStyle.H5.BlockPrefix = ""
	customStyle.H5.BlockSuffix = ""

	customStyle.H6.Margin = nil
	customStyle.H6.BlockPrefix = ""
	customStyle.H6.BlockSuffix = ""

	// Tables
	customStyle.Table.Margin = nil
	customStyle.Table.BlockPrefix = ""
	customStyle.Table.BlockSuffix = ""

	// Inline Elements
	customStyle.Text.BlockPrefix = ""
	customStyle.Text.BlockSuffix = ""

	customStyle.Strong.BlockPrefix = ""
	customStyle.Strong.BlockSuffix = ""

	customStyle.Emph.BlockPrefix = ""
	customStyle.Emph.BlockSuffix = ""

	customStyle.Strikethrough.BlockPrefix = ""
	customStyle.Strikethrough.BlockSuffix = ""

	customStyle.Link.BlockPrefix = ""
	customStyle.Link.BlockSuffix = ""

	customStyle.LinkText.BlockPrefix = ""
	customStyle.LinkText.BlockSuffix = ""

	customStyle.Image.BlockPrefix = ""
	customStyle.Image.BlockSuffix = ""

	customStyle.ImageText.BlockPrefix = ""
	customStyle.ImageText.BlockSuffix = ""
	return customStyle
}
