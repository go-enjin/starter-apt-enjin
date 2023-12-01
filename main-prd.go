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
)

//go:embed public/**
var publicFs embed.FS

//go:embed content/**
var contentFs embed.FS

func init() {
	fPublic = public.New().
		MountEmbedPath("/", "public", publicFs).
		Make()

	fAptRepo = public.NewTagged("fs-public-apt-repo").
		MountLocalPath("/"+UseAptFlavour, UseBasePath+"/"+UseAptFlavour).
		SetRegexCacheControl("/dists/", "no-store").
		Make()

	fContent = content.New().
		MountEmbedPath("/", "content", contentFs).
		AddToIndexProviders("pages-pql").
		AddToSearchProviders("bleve-fts").
		Make()
}