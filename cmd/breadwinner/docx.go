/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package breadwinner

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
)

func xmlsafe(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(text)
}

func piece(text string, bold, ital bool, size int) string {
	props := ""
	if bold {
		props += "<w:b/>"
	}
	if ital {
		props += "<w:i/>"
	}

	return fmt.Sprintf(`<w:r><w:rPr>%s<w:sz w:val="%d"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r>`, props, size, xmlsafe(text))
}

func pieces(text string, force bool, size int) string {
	var out strings.Builder

	for _, part := range spans(text) {
		out.WriteString(piece(part.Text, part.Bold || force, part.Ital, size))
	}

	return out.String()
}

func graph(props, body string) string {
	if props != "" {
		props = "<w:pPr>" + props + "</w:pPr>"
	}
	return "<w:p>" + props + body + "</w:p>"
}

func docxbody(d doc, text int) string {
	var out strings.Builder

	out.WriteString(graph(`<w:pStyle w:val="Title"/><w:jc w:val="center"/>`, piece(d.Name, true, false, 40)))

	if d.Title != "" {
		out.WriteString(graph(`<w:jc w:val="center"/>`, piece(d.Title, false, false, 22)))
	}

	if items := d.contact(); len(items) > 0 {
		out.WriteString(graph(`<w:spacing w:after="80"/><w:jc w:val="center"/>`, piece(strings.Join(items, "  |  "), false, false, 19)))
	}

	for _, sec := range d.shown() {
		out.WriteString(graph(`<w:pStyle w:val="Heading1"/>`, piece(strings.ToUpper(sec.Title), true, false, 22)))

		for _, item := range sec.Blocks {
			switch item.Kind {
			case para:
				out.WriteString(graph(`<w:spacing w:after="40"/>`, pieces(item.Text, false, 20)))
			case plain:
				out.WriteString(graph(`<w:spacing w:after="40"/>`, pieces(item.Text, false, 20)))
			case note:
				out.WriteString(graph(`<w:spacing w:after="20"/>`, piece(flat(item.Text), false, true, 20)))
			case bullet:
				left := 360 + item.Level*280
				props := fmt.Sprintf(`<w:spacing w:after="20"/><w:ind w:left="%d" w:hanging="220"/>`, left)
				out.WriteString(graph(props, piece("•", false, false, 20)+"<w:r><w:tab/></w:r>"+pieces(item.Text, false, 20)))
			case entry:
				props := fmt.Sprintf(`<w:keepNext/><w:tabs><w:tab w:val="right" w:pos="%d"/></w:tabs><w:spacing w:before="80" w:after="20"/>`, text)

				var body strings.Builder

				for idx, part := range item.Parts {
					if idx == 0 {
						body.WriteString(pieces(part, true, 21))
						continue
					}
					body.WriteString(piece(" | ", false, false, 21))
					body.WriteString(pieces(part, false, 21))
				}

				if item.Date != "" {
					body.WriteString("<w:r><w:tab/></w:r>" + piece(item.Date, false, false, 20))
				}

				out.WriteString(graph(props, body.String()))
			}
		}
	}

	return out.String()
}

func docx(d doc, paper string) ([]byte, error) {
	wide, high := 11906, 16838
	if paper == "letter" {
		wide, high = 12240, 15840
	}

	text := wide - 2160

	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		docxbody(d, text) +
		fmt.Sprintf(`<w:sectPr><w:pgSz w:w="%d" w:h="%d"/><w:pgMar w:top="1080" w:right="1080" w:bottom="1080" w:left="1080" w:header="500" w:footer="500" w:gutter="0"/></w:sectPr>`, wide, high) +
		`</w:body></w:document>`

	styles := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial" w:cs="Arial" w:eastAsia="Arial"/><w:sz w:val="20"/><w:szCs w:val="20"/></w:rPr></w:rPrDefault>` +
		`<w:pPrDefault><w:pPr><w:spacing w:after="0" w:line="260" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>` +
		`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:after="40"/></w:pPr><w:rPr><w:b/><w:sz w:val="40"/></w:rPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>` +
		`<w:pPr><w:keepNext/><w:pBdr><w:bottom w:val="single" w:sz="4" w:space="1" w:color="000000"/></w:pBdr><w:spacing w:before="200" w:after="80"/><w:outlineLvl w:val="0"/></w:pPr>` +
		`<w:rPr><w:b/><w:sz w:val="22"/></w:rPr></w:style></w:styles>`

	types := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
		`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/></Types>`

	root := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/></Relationships>`

	links := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`

	props := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>` + xmlsafe(d.Name+" - Resume") + `</dc:title><dc:creator>` + xmlsafe(d.Name) + `</dc:creator></cp:coreProperties>`

	var buf bytes.Buffer

	pack := zip.NewWriter(&buf)

	for _, file := range [][2]string{
		{"[Content_Types].xml", types},
		{"_rels/.rels", root},
		{"word/document.xml", document},
		{"word/styles.xml", styles},
		{"word/_rels/document.xml.rels", links},
		{"docProps/core.xml", props},
	} {
		handle, err := pack.Create(file[0])
		if err != nil {
			return nil, err
		}

		if _, err := handle.Write([]byte(file[1])); err != nil {
			return nil, err
		}
	}

	if err := pack.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
