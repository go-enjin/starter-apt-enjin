// Copyright (c) 2023  The Go-Enjin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dpkgdeb

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-enjin/be/pkg/log"
)

func EscapeQuotes(input string) (output string) {
	output = strings.ReplaceAll(input, `"`, `\"`)
	output = strings.ReplaceAll(output, "\n", `\n`)
	return
}

var (
	RxDpkgDebInfoLine = regexp.MustCompile(`^\s*([-_a-zA-Z0-9]+?):\s*(.+?)\s*$`)
	RxDpkgDebInfoDesc = regexp.MustCompile(`(?ms)^\s*Description:\s*(.+?)$(.+?)\z`)
	rxNameAndEmail    = regexp.MustCompile(`^\s*(.+?)\s*<([^>]+?)>\s*$`)
)

func ParseDpkgDebInfoOutput(output string) (parsed map[string]string, lines []string, order []string) {
	parsed = make(map[string]string)
	lines = strings.Split(output, "\n")
	for _, line := range lines {
		if RxDpkgDebInfoLine.MatchString(line) {
			m := RxDpkgDebInfoLine.FindAllStringSubmatch(line, 1)
			parsed[m[0][1]] = m[0][2]
			order = append(order, m[0][1])
		}
	}
	if v, ok := parsed["Maintainer"]; ok {
		if rxNameAndEmail.MatchString(v) {
			m := rxNameAndEmail.FindAllStringSubmatch(v, 1)
			parsed["MaintainerName"] = m[0][1]
			parsed["MaintainerMail"] = m[0][2]
		} else {
			log.ErrorF("error parsing name and email: %v", v)
		}
	}
	if RxDpkgDebInfoDesc.MatchString(output) {
		m := RxDpkgDebInfoDesc.FindAllStringSubmatch(output, 1)
		parsed["Description"] = m[0][1]
		parsed["LongDescription"] = m[0][2]
	} else {
		log.ErrorF("error parsing long description from dpkg-deb --info output:\n[begin output]\n%v[end output]", output)
	}
	return
}

func DecomposeDpkgDebInfo(parsed map[string]string) (name, summary, description, section, version, homepage string) {
	for key, value := range parsed {
		switch key {
		case "Package":
			name = value
		case "Description":
			summary = value
		case "LongDescription":
			description = value
		case "Section":
			section = value
		case "Version":
			version = value
		case "Homepage":
			homepage = value
		}
	}
	return
}

func MakeLongDescriptionParagraphs(input string) (output string) {
	var paragraphs []string
	var current string
	for _, line := range strings.Split(input, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed == "." {
			paragraphs = append(paragraphs, current)
			current = ""
		} else {
			if current != "" {
				current += " "
			}
			current += trimmed
		}
	}
	if current != "" {
		paragraphs = append(paragraphs, current)
	}
	for idx, paragraph := range paragraphs {
		if idx > 0 {
			output += ","
		}
		output += fmt.Sprintf(`{"type":"p","text":["%v"]}`, EscapeQuotes(paragraph))
	}
	return
}

func MakePackageFields(parsed map[string]string, order []string) (output string) {
	var rows []string

	for _, key := range order {
		if v, ok := parsed[key]; ok {
			var value string
			ev := EscapeQuotes(v)
			switch key {
			case "Homepage":
				value = fmt.Sprintf(
					`{"type":"a","href":"%v","text":["%v"],"target":"_blank"}`,
					ev, ev,
				)
			case "Maintainer":
				if mail, ok := parsed["MaintainerMail"]; ok {
					escMail := EscapeQuotes(mail)
					if name, ok := parsed["MaintainerName"]; ok {
						value = fmt.Sprintf(
							`{"type":"a","href":"mailto:%v","text":["%v"]}`,
							escMail,
							EscapeQuotes(name),
						)
					} else {
						value = fmt.Sprintf(
							`{"type":"a","href":"mailto:%v","text":["%v"]}`,
							escMail,
							escMail,
						)
					}
				} else {
					value = `"(missing)"`
				}
			case "MaintainerName", "MaintainerMail", "Installed-Size":
				continue
			default:
				value = `"` + ev + `"`
			}

			row := `{"type": "tr","data": [{ "type": "td", "text": [{"type":"b","text":["%v"]}] },{ "type": "td", "text": [%v] }]}`
			rows = append(rows, fmt.Sprintf(row, key, value))
		}
	}

	output = fmt.Sprintf(`{"type":"table","body":[%v]}`, strings.Join(rows, ","))
	return
}

// gPageTemplate requires the following Sprintf arguments:
//
//   - pageTitle, pageDesc, pageUrl
//   - pageHeader
//   - fields
//   - summary, description
//   - infoBlock, contentsBlock
const gPageTemplate = `+++
"title" = "%v"
"description" = "%v"
"url" = "%v"
"format" = "njn"
"language" = "en"
+++
[
	{
        "type": "header",
        "tag": "main-header",
        "profile": "outer--inner",
        "padding": "top",
        "margins": "bottom",
        "content": {
            "header": [
                "%v"
            ]
        }
    },

    {
        "tag": "main-sidebar",
        "type": "sidebar",
        "profile": "full--outer",
        "padding": "none",
        "margins": "bottom",
        "side": "right",
        "sticky": "true",
        "stack": "top",
        "jump-top": "true",
        "jump-link": "true",
        "content": {

            "aside": [

                 {
                    "tag": "deb-fields",
                    "type": "content",
                    "profile": "full--full",
                    "content": {
                        "section": [%v]
                    }
                }

            ],

            "blocks": [

                {
                    "type": "content",
                    "tag": "package-summary",
                    "profile": "outer--inner",
                    "padding": "both",
                    "margins": "both",
                    "jump-top": "true",
                    "jump-link": "true",
                    "content": {
                        "header": [
                            "%v"
                        ],
                        "section": [%v]
                    }
                },

                {
                    "type": "content",
                    "tag": "dpkg-deb--info--contents",
                    "profile": "outer--inner",
                    "padding": "both",
                    "margins": "both",
                    "jump-top": "true",
                    "jump-link": "true",
                    "content": {
                        "header": [
                            "dpkg-deb --info --contents"
                        ],
                        "section": [
                            {
                                "type": "code",
                                "code": [%v,%v]
                            }
                        ]
                    }
                }

            ]
        }
    }

]`