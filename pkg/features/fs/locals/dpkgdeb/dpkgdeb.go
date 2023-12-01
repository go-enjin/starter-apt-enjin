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
	"sort"
	"strings"

	"github.com/fvbommel/sortorder"
	"github.com/urfave/cli/v2"

	"github.com/go-enjin/golang-org-x-text/language"

	"github.com/go-enjin/be/drivers/fs/local"
	"github.com/go-enjin/be/drivers/fts/bleve"
	"github.com/go-enjin/be/pkg/feature"
	uses_actions "github.com/go-enjin/be/pkg/feature/uses-actions"
	"github.com/go-enjin/be/pkg/forms"
	"github.com/go-enjin/be/pkg/fs"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/maps"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

var (
	DefaultCacheControl = "max-age=604800, must-revalidate"
)

const (
	Tag    feature.Tag = "local-deb-info"
	Bucket string      = "local-deb-info"
)

type Feature interface {
	feature.Feature
	feature.PageProvider
	feature.UseMiddleware
	feature.UserActionsProvider
}

type MakeFeature interface {
	MountPath(mount, path string) MakeFeature
	SetCacheControl(values string) MakeFeature

	Make() Feature
}

type CFeature struct {
	feature.CFeature
	uses_actions.CUsesActions

	search bleve.Feature

	setup map[string]string
	mount []*feature.CMountPoint
	infos map[string]*dpkgDeb

	cacheControl string
}

func New() MakeFeature {
	return NewTagged(Tag)
}

func NewTagged(tag feature.Tag) MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.PackageTag = Tag
	f.FeatureTag = tag
	f.CFeature.Construct(f)
	f.CUsesActions.ConstructUsesActions(f)
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
	f.setup = make(map[string]string)
	f.infos = make(map[string]*dpkgDeb)
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

func (f *CFeature) Build(_ feature.Buildable) (err error) {
	return
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)

	for _, feat := range feature.FilterTyped[bleve.Feature](enjin.Features().List()) {
		f.search = feat
		break
	}

	if f.search == nil {
		log.FatalF("fts enjin feature not found")
		return
	}

	var err error
	for _, path := range maps.SortedKeys(f.setup) {

		var lfs fs.FileSystem
		if lfs, err = local.New(f.Tag().String(), path); err != nil {
			log.FatalF(`error mounting filesystem: %v`, err)
			return
		}

		mp := &feature.CMountPoint{
			Path:  path,
			Mount: f.setup[path],
			ROFS:  lfs,
			RWFS:  nil,
		}
		f.mount = append(f.mount, mp)

		log.DebugF("mounted local debinfo filesystem: %v", mp)
	}
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	if err = f.CFeature.Startup(ctx); err != nil {
		return
	}

	for _, mp := range f.mount {
		files, _ := mp.ROFS.ListAllFiles(".")
		for _, file := range files {
			if strings.HasSuffix(file, ".deb") || strings.HasSuffix(file, ".udeb") {
				_, url := f.makeDebNameUrl(mp.Mount, file)
				if f.infos[url], err = f.makeDpkgDeb(file, mp); err != nil {
					err = fmt.Errorf("error caching dpkg-deb outputs: %v - %w", file, err)
					return
				}
				if p, ee := f.makeDebPage(nil, f.infos[url]); ee == nil {
					if err = f.search.AddToSearchIndex(nil, p); err != nil {
						err = fmt.Errorf("error indexing dpkg-deb page: %v - %w", url, err)
						return
					}
				}
				log.DebugF("cached and indexed dpkg-deb: %v", url)
			}
		}
	}

	// err = fmt.Errorf("testing")
	return
}

func (f *CFeature) UserActions() (actions feature.Actions) {
	actions = append(actions, feature.NewAction(f.Tag().Kebab(), "view", "page"))
	return
}

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including local debinfo middleware: %v", f.listMountPaths())
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := forms.CleanRequestPath(r.URL.Path)
			if err := f.ServePath(path, s, w, r); err == nil {
				return
			} else if err.Error() != "path not found" {
				log.ErrorF("local debinfo error: %v", err)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (f *CFeature) ServePath(path string, _ feature.System, w http.ResponseWriter, r *http.Request) (err error) {
	// log.DebugF("checking path: %v", path)
	if dd, ok := f.infos[path]; ok {

		var p feature.Page
		if p, err = f.makeDebPage(r, dd); err != nil {
			err = fmt.Errorf("error making deb page: %v - %w", path, err)
			return
		}

		pg := p.Copy()
		var cacheControl string
		if f.cacheControl == "" {
			cacheControl = DefaultCacheControl
		} else {
			cacheControl = f.cacheControl
		}
		cacheControl = pg.Context().String("CacheControl", cacheControl)
		pg.Context().SetSpecific("CacheControl", cacheControl)
		if err = f.Enjin.ServePage(pg, w, r); err != nil {
			err = fmt.Errorf("serve local %v debinfo: %v - error: %w", f.setup[path], path, err)
			return
		}

		log.DebugF("served local %v debinfo: [%v] %v", f.setup[path], pg.Language(), path)
		return
	}

	err = fmt.Errorf("path not found")
	return
}

func (f *CFeature) FindRedirection(path string) (p feature.Page) {
	// p, _ = f.cache.LookupRedirect(Bucket, path)
	return
}

func (f *CFeature) FindTranslations(path string) (found []feature.Page) {
	// found = f.cache.LookupTranslations(Bucket, path)
	return
}

func (f *CFeature) FindPage(r *http.Request, tag language.Tag, url string) (p feature.Page) {
	var err error
	if dd, ok := f.infos[url]; ok {
		if p, err = f.makeDebPage(r, dd); err != nil {
			err = fmt.Errorf("error making deb page: %v - %w", url, err)
			return
		}
	}
	return
}

func (f *CFeature) LookupPrefixed(path string) (pages []feature.Page) {
	// pages = f.cache.LookupPrefix(Bucket, path)
	return
}

func (f *CFeature) FindTranslationUrls(url string) (pages map[language.Tag]string) {
	//f.RLock()
	//defer f.RUnlock()

	pages = make(map[language.Tag]string)

	for _, p := range f.FindTranslations(url) {
		pages[p.LanguageTag()] = p.Url()
	}

	return
}

func (f *CFeature) listMountPaths() (paths []string) {
	for path, _ := range f.setup {
		paths = append(paths, path)
	}
	sort.Sort(sortorder.Natural(paths))
	return
}