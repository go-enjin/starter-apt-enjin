//go:build dev || !prd

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
	"github.com/go-enjin/be/features/fs/locals/content"
	"github.com/go-enjin/be/features/fs/locals/public"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/theme"
)

func ppaEnjinTheme() (t *theme.Theme) {
	var err error
	if t, err = theme.NewLocal("themes/apt-enjin"); err != nil {
		log.FatalF("error loading local apt-enjin theme: %v", err)
	}
	return
}

func ppaPublicFeature() (f feature.Feature) {
	f = public.New().
		MountPath("/", "public").
		MountPath("/"+UseAptFlavour, UseBasePath+"/"+UseAptFlavour).
		SetRegexCacheControl("/dists/", "no-store").
		Make()
	return
}

func ppaAptRepoFeature() (f feature.Feature) {
	return
}

func ppaContentFeature() (f feature.Feature) {
	f = content.New().
		MountPath("/", "content").
		Make()
	return
}
