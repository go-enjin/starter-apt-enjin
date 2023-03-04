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

	"github.com/go-enjin/be/features/outputs/htmlify"
	pgc "github.com/go-enjin/be/features/pages/caching/stock-pgc"
	fts "github.com/go-enjin/be/features/pages/indexing/bleve-fts"
	"github.com/go-enjin/be/features/pages/search"
	"github.com/go-enjin/be/pkg/cli/env"
	"github.com/go-enjin/golang-org-x-text/language"
	semantic "github.com/go-enjin/semantic-enjin-theme"

	"github.com/go-enjin/starter-apt-enjin/pkg/features/fs/locals/dpkgdeb"

	"github.com/go-enjin/be"
	"github.com/go-enjin/be/features/log/papertrail"
	"github.com/go-enjin/be/features/pages/formats"
	"github.com/go-enjin/be/features/pages/robots"
	"github.com/go-enjin/be/features/requests/headers/proxy"
	"github.com/go-enjin/be/pkg/lang"
	"github.com/go-enjin/be/pkg/log"
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
	UseBasePath   = ""
	UseAptFlavour = ""
)

func init() {

	if AptFlavour == "" {
		log.FatalF("build error: .AptFlavour is empty\n")
	}

	UseBasePath = env.Get("AE_BASEPATH", "apt-repository")
	UseAptFlavour = env.Get("APT_FLAVOUR", AptFlavour)
}

func main() {
	enjin := be.New().
		SiteTag(SiteTag).
		SiteName(SiteName).
		SiteTagLine(SiteTagLine).
		AddFeature(proxy.New().Enable().Make()).
		AddFeature(formats.New().Defaults().Make()).
		AddFeature(pgc.New().Make()).
		AddFeature(htmlify.New().Make()).
		SiteDefaultLanguage(language.English).
		SiteSupportedLanguages(language.English).
		SiteLanguageMode(lang.NewPathMode().Make()).
		AddTheme(semantic.SemanticEnjinTheme()).
		AddTheme(ppaEnjinTheme()).
		SetTheme("apt-enjin").
		AddFeature(fts.New().Make()).
		AddFeature(search.New().SetPath("/search").Make()).
		AddFeature(papertrail.Make()).
		AddFeature(robots.New().
			AddRuleGroup(robots.NewRuleGroup().
				AddUserAgent("*").AddAllowed("/").Make(),
			).Make(),
		).
		AddFeature(ppaPublicFeature()).
		AddFeature(ppaAptRepoFeature()).
		AddFeature(ppaContentFeature()).
		AddFeature(dpkgdeb.New().
			MountPath("/dpkg-deb/"+UseAptFlavour, UseBasePath+"/"+UseAptFlavour).
			Make(),
		).
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
		SetStatusPage(404, "/404").
		SetStatusPage(500, "/500")
	// add content and status pages
	if err := enjin.Build().Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "enjin.Run error: %v\n", err)
		os.Exit(1)
	}
}