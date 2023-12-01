// Copyright (c) 2022  The Go-Enjin Authors
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

package main

import (
	"fmt"
	"os"

	"github.com/go-enjin/golang-org-x-text/language"

	apt "github.com/go-enjin/apt-enjin-theme"
	semantic "github.com/go-enjin/semantic-enjin-theme"

	"github.com/go-enjin/be"
	"github.com/go-enjin/be/drivers/fts/bleve"
	"github.com/go-enjin/be/drivers/kvs/gocache"
	"github.com/go-enjin/be/features/fs/themes"
	"github.com/go-enjin/be/features/pages/pql"
	"github.com/go-enjin/be/features/pages/robots"
	"github.com/go-enjin/be/features/pages/search"
	"github.com/go-enjin/be/pkg/cli/env"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/lang"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/presets/defaults"

	"github.com/go-enjin/starter-apt-enjin/pkg/features/fs/locals/dpkgdeb"
)

const (
	gPagesPqlKvsFeature = "pages-pql-kvs-feature"
	gPagesPqlKvsCache   = "pages-pql-kvs-cache"
)

var (
	SiteTag            = "APT"
	SiteName           = "Apt Enjin"
	SiteAptUrl         = ""
	SiteTagLine        = "apt personal package archives"
	SetupDebUrl        = ""
	SetupDebName       = ""
	PkgSection         = ""
	AptFlavour         = ""
	AptCodename        = ""
	AptComponents      = ""
	AptArchitectures   = ""
	AptPublicKeyFile   = ""
	AptSourcesListFile = ""
)

var (
	UseBasePath   = env.Get("AE_BASEPATH", "apt-repository")
	UseAptFlavour = env.Get("APT_FLAVOUR", AptFlavour)

	fThemes  feature.Feature
	fPublic  feature.Feature
	fContent feature.Feature
	fAptRepo feature.Feature
)

func init() {
	if AptFlavour == "" {
		log.FatalF("build error: .AptFlavour is empty\n")
	}
}

func main() {
	enjin := be.New().
		SiteTag(SiteTag).
		SiteName(SiteName).
		SiteTagLine(SiteTagLine).
		SiteDefaultLanguage(language.English).
		SiteSupportedLanguages(language.English).
		SiteLanguageMode(lang.NewPathMode().Make()).
		SiteCopyrightName(SiteName).
		SiteCopyrightNotice("All rights reserved").
		Set("SiteAptUrl", SiteAptUrl).
		Set("SiteLogoUrl", "/media/go-enjin-logo.png").
		Set("SiteLogoAlt", "Go-Enjin logo").
		Set("SetupPackageUrl", SetupDebUrl).
		Set("SetupPackageName", SetupDebName).
		Set("PkgSection", PkgSection).
		Set("AptFlavour", AptFlavour).
		Set("AptCodename", AptCodename).
		Set("AptComponents", AptComponents).
		Set("AptArchitectures", AptArchitectures).
		Set("AptPublicKeyFile", AptPublicKeyFile).
		Set("AptSourcesListFile", AptSourcesListFile).
		AddPreset(defaults.New().Make()).
		AddFeature(themes.New().
			Include(semantic.Theme()).
			Include(apt.Theme()).
			SetTheme(apt.Name).
			Make()).
		AddFeature(gocache.NewTagged(gPagesPqlKvsFeature).
			AddMemoryCache(gPagesPqlKvsCache).
			Make()).
		AddFeature(pql.NewTagged("pages-pql").
			SetKeyValueCache(gPagesPqlKvsFeature, gPagesPqlKvsCache).
			Make()).
		AddFeature(bleve.NewTagged("bleve-fts").Make()).
		AddFeature(search.New().SetSearchPath("/search").Make()).
		AddFeature(robots.New().
			AddRuleGroup(robots.NewRuleGroup().
				AddUserAgent("*").
				AddAllowed("/").
				Make()).
			Make()).
		AddFeature(fPublic).
		AddFeature(fAptRepo).
		AddFeature(fContent).
		AddFeature(dpkgdeb.New().
			MountPath("/dpkg-deb/"+UseAptFlavour, UseBasePath+"/"+UseAptFlavour).
			Make()).
		SetPublicAccess(
			feature.NewAction("enjin", "view", "page"),
			feature.NewAction("fs-content", "view", "page"),
			feature.NewAction("local-deb-info", "view", "page"),
		).
		SetStatusPage(404, "/404").
		SetStatusPage(500, "/500")
	// add content and status pages
	if err := enjin.Build().Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "enjin.Run error: %v\n", err)
		os.Exit(1)
	}
}