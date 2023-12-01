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
	"strings"
	"time"

	"github.com/go-enjin/be/pkg/cli/run"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/types/page"
)

type dpkgDeb struct {
	Info     string
	Contents string
	File     string
	MP       *feature.CMountPoint
}

func (f *CFeature) makeDebNameUrl(mount, file string) (name, url string) {
	name = filepath.Base(file)
	url = mount + "/" + name
	return
}

func (f *CFeature) makeDpkgDeb(file string, mp *feature.CMountPoint) (dd *dpkgDeb, err error) {
	fullpath := filepath.Join(mp.Path, file)
	dd = &dpkgDeb{
		File: file,
		MP:   mp,
	}

	if dd.Info, _, _, err = run.Cmd("dpkg-deb", "--info", fullpath); err != nil {
		err = fmt.Errorf("dpkg-deb --info error: %v - %v", file, err)
		return
	}
	if dd.Contents, _, _, err = run.Cmd("dpkg-deb", "--contents", fullpath); err != nil {
		err = fmt.Errorf("dpkg-deb --contents error: %v - %v", file, err)
		return
	}

	return
}

func (f *CFeature) makeDebPage(r *http.Request, dd *dpkgDeb) (p feature.Page, err error) {

	fullpath := filepath.Join(dd.MP.Path, dd.File)

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

	debName, url := f.makeDebNameUrl(dd.MP.Mount, dd.File)
	parsed, lines, order := ParseDpkgDebInfoOutput(dd.Info)
	// name, summary, description, section, version, homepage := DecomposeDpkgDebInfo(parsed)
	_, summary, description, _, _, _ := DecomposeDpkgDebInfo(parsed)

	infoCodeBlock := makeIntoLines(lines)
	contentsBlock := makeIntoLines(strings.Split(dd.Contents, "\n"))

	fields := MakePackageFields(parsed, order)

	var source = fmt.Sprintf(
		gPageTemplate,
		debName, "Debian package details for "+debName, url,
		debName,
		fields,
		summary, MakeLongDescriptionParagraphs(description),
		infoCodeBlock,
		contentsBlock,
	)

	created := time.Now().Unix()
	t := f.Enjin.MustGetTheme()
	if p, err = page.New(f.Tag().Kebab(), fullpath, source, created, created, t, f.Enjin.Context(r)); err != nil {
		err = fmt.Errorf("error making new page: %v - %v", fullpath, err)
		return
	}
	p.SetSlugUrl(url)
	//p.PageMatter = matter.NewPageMatter(f.Tag().String(), p.Path, source, matter.JsonMatter, p.Context)
	//err = f.search.AddToSearchIndex(nil, p)
	return
}