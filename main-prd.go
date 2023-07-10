//go:build prd

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

package main

import (
	"embed"

	"github.com/go-enjin/be/features/fs/content"
	"github.com/go-enjin/be/features/fs/public"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/theme"
)

//go:embed themes/**
var themeFs embed.FS

func ppaEnjinTheme() (t *theme.Theme) {
	var err error
	if t, err = theme.NewEmbed("apt-enjin", "themes/apt-enjin", themeFs); err != nil {
		log.FatalF("error loading embedded apt-enjin theme: %v", err)
	}
	return
}

//go:embed public/**
var publicFs embed.FS

func ppaPublicFeature() (f feature.Feature) {
	f = public.New().
		MountEmbedPath("/", "public", publicFs).
		Make()
	return
}

func ppaAptRepoFeature() (f feature.Feature) {
	f = public.NewTagged("fs-public-apt-repo").
		MountLocalPath("/"+UseAptFlavour, UseBasePath+"/"+UseAptFlavour).
		SetRegexCacheControl("/dists/", "no-store").
		Make()
	return
}

//go:embed content/**
var contentFs embed.FS

func ppaContentFeature() (f feature.Feature) {
	f = content.New().
		MountEmbedPath("/", "content", contentFs).
		AddToIndexProviders("pages-pql").
		AddToSearchProviders("bleve-fts").
		Make()
	return
}