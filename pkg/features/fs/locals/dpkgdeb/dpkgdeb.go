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
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fvbommel/sortorder"
	"github.com/urfave/cli/v2"

	"github.com/go-enjin/be/features/pages/indexing/bleve-fts"
	"github.com/go-enjin/be/pkg/hash/sha"

	"github.com/go-enjin/be/pkg/cli/run"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/forms"
	"github.com/go-enjin/be/pkg/fs"
	"github.com/go-enjin/be/pkg/fs/local"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/maps"
	"github.com/go-enjin/be/pkg/page"
	"github.com/go-enjin/be/pkg/theme"
	"github.com/go-enjin/golang-org-x-text/language"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

var (
	DefaultCacheControl = "max-age=604800, must-revalidate"
)

const (
	Tag    feature.Tag = "LocalDebInfo"
	Bucket string      = "local-deb-info"
)

type Feature interface {
	feature.Middleware
	feature.PageProvider
}

type CFeature struct {
	feature.CMiddleware

	enjin  feature.Internals
	search fts.Feature

	setup map[string]string
	mount []fs.MountPoint
	infos map[string]*page.Page

	cacheControl string
}

type MakeFeature interface {
	MountPath(mount, path string) MakeFeature
	SetCacheControl(values string) MakeFeature

	Make() Feature
}

func New() MakeFeature {
	f := new(CFeature)
	f.Init(f)
	return f
}

func (f *CFeature) MountPath(mount, path string) MakeFeature {
	f.setup[path] = mount
	return f
}

func (f *CFeature) SetCacheControl(values string) MakeFeature {
	f.cacheControl = values
	return f
}

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CMiddleware.Init(this)
	f.setup = make(map[string]string)
	f.infos = make(map[string]*page.Page)
}

func (f *CFeature) Tag() (tag feature.Tag) {
	tag = Tag
	return
}

func (f *CFeature) Build(_ feature.Buildable) (err error) {
	return
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.enjin = enjin

	for _, feat := range enjin.Features() {
		if s, ok := feat.(fts.Feature); ok {
			f.search = s
			break
		}
	}

	if f.search == nil {
		log.FatalF("fts enjin feature not found")
		return
	}

	var err error
	for _, path := range maps.SortedKeys(f.setup) {

		var lfs fs.FileSystem
		if lfs, err = local.New(path); err != nil {
			log.FatalF(`error mounting filesystem: %v`, err)
			return
		}

		mp := fs.MountPoint{
			Path:  path,
			Mount: f.setup[path],
			FS:    lfs,
		}
		f.mount = append(f.mount, mp)

		log.DebugF("mounted local debinfo filesystem %v", mp)
	}
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {

	for _, mp := range f.mount {
		files, _ := mp.FS.ListAllFiles(".")
		for _, file := range files {
			if strings.HasSuffix(file, ".deb") || strings.HasSuffix(file, ".udeb") {
				if p, ee := f.makeDebPage(file, mp); ee != nil {
					err = fmt.Errorf("error making deb page: %v - %v", file, ee)
					return
				} else {
					f.infos[p.Url] = p
					log.DebugF("cached dpkg-deb info: %v", p.Url)
				}
			}
		}
	}

	// err = fmt.Errorf("testing")
	return
}

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including local debinfo middleware: %v", f.listMountPaths())
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := forms.SanitizeRequestPath(r.URL.Path)
			if err := f.ServePath(path, s, w, r); err == nil {
				return
			} else if err.Error() != "path not found" {
				log.ErrorF("local debinfo error: %v", err)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (f *CFeature) ServePath(path string, s feature.System, w http.ResponseWriter, r *http.Request) (err error) {
	// log.DebugF("checking path: %v", path)
	if p, ok := f.infos[path]; ok {
		pg := p.Copy()
		var cacheControl string
		if f.cacheControl == "" {
			cacheControl = DefaultCacheControl
		} else {
			cacheControl = f.cacheControl
		}
		cacheControl = pg.Context.String("CacheControl", cacheControl)
		pg.Context.SetSpecific("CacheControl", cacheControl)
		if err = s.ServePage(pg, w, r); err == nil {
			log.DebugF("served local %v debinfo: [%v] %v", f.setup[path], pg.Language, path)
			return
		}
		err = fmt.Errorf("serve local %v debinfo: %v - error: %v", f.setup[path], path, err)
		return
	}

	err = fmt.Errorf("path not found")
	return
}

func (f *CFeature) FindRedirection(path string) (p *page.Page) {
	// p, _ = f.cache.LookupRedirect(Bucket, path)
	return
}

func (f *CFeature) FindTranslations(path string) (found []*page.Page) {
	// found = f.cache.LookupTranslations(Bucket, path)
	return
}

func (f *CFeature) FindPage(tag language.Tag, path string) (p *page.Page) {
	if pg, ok := f.infos[path]; ok {
		p = pg
	}
	return
}

func (f *CFeature) LookupPrefixed(path string) (pages []*page.Page) {
	// pages = f.cache.LookupPrefix(Bucket, path)
	return
}

func (f *CFeature) listMountPaths() (paths []string) {
	for path, _ := range f.setup {
		paths = append(paths, path)
	}
	sort.Sort(sortorder.Natural(paths))
	return
}

func (f *CFeature) makeDebPage(file string, mp fs.MountPoint) (p *page.Page, err error) {
	if !fs.FileExists(file) {
		err = fmt.Errorf("file not found: %v", file)
		return
	}

	fullpath := filepath.Join(mp.Path, file)

	var infoStdout string
	if infoStdout, _, _, err = run.Cmd("dpkg-deb", "--info", fullpath); err != nil {
		err = fmt.Errorf("dpkg-deb --info error: %v - %v", file, err)
		return
	}
	var debContentsOutput string
	if debContentsOutput, _, _, err = run.Cmd("dpkg-deb", "--contents", fullpath); err != nil {
		err = fmt.Errorf("dpkg-deb --contents error: %v - %v", file, err)
		return
	}

	makeIntoLines := func(lines []string) (output string) {
		last := len(lines) - 1
		for idx, line := range lines {
			comma := ","
			if idx == last {
				comma = ""
			}
			output += fmt.Sprintf("\"%v\"%s", EscapeQuotes(line), comma)
		}
		return
	}

	debname := filepath.Base(file)
	parsed, lines, order := ParseDpkgDebInfoOutput(infoStdout)
	// name, summary, description, section, version, homepage := DecomposeDpkgDebInfo(parsed)
	_, summary, description, _, _, _ := DecomposeDpkgDebInfo(parsed)
	url := mp.Mount + "/" + debname

	infoCodeBlock := makeIntoLines(lines)
	contentsBlock := makeIntoLines(strings.Split(debContentsOutput, "\n"))

	fields := MakePackageFields(parsed, order)

	var source = fmt.Sprintf(
		gPageTemplate,
		debname, "Debian package details for "+debname, url,
		debname,
		fields,
		summary, MakeLongDescriptionParagraphs(description),
		infoCodeBlock,
		contentsBlock,
	)

	created := time.Now().Unix()
	var t *theme.Theme
	if t, err = f.enjin.GetTheme(); err != nil {
		return
	}
	var shasum string
	if shasum, err = sha.DataHash10([]byte(source)); err != nil {
		return
	}
	if p, err = page.New(fullpath, source, shasum, created, created, t, f.enjin.Context()); err != nil {
		err = fmt.Errorf("error making new page: %v - %v", fullpath, err)
		return
	}
	p.SetSlugUrl(url)
	err = f.search.AddToSearchIndex(nil, p)
	return
}

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