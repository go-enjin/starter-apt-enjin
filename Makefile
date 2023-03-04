#!/usr/bin/make --no-print-directory --jobs=1 --environment-overrides -f

# Copyright (c) 2023  The Go-Enjin Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

-include Sitefile
export
-include .env
export

BE_LOCAL_PATH ?= ../be

#
#: apt-enjin custom variables
#

export SITETAG   ?= PPA
export SITEKEY   ?= sitename-ppa
export SITEURL   ?= http://localhost:3334
export SITENAME  ?= Site Name
export SITEMAIL  ?= site@email.address
export SITEMAINT ?= ${SITENAME}

export APT_FLAVOUR       ?= debian
export APT_CODENAME      ?= bullseye
export APT_COMPONENTS    ?= main
export APT_ARCHITECTURES ?= source arm64 amd64

export PKG_SECTION ?= go-enjin
export PKG_VERSION ?= 0.1.0
export PKG_CHANGES ?= Initial release.

export AE_GPG_FILE ?= ${SITEKEY}.gpg
export AE_SIGN_KEY ?= ${SITEMAIL}
export AE_ARCHIVES ?= apt-archives
export AE_BASEPATH ?= apt-repository

export AE_GPG_HOME ?= $(shell realpath .gpg)
export GNUPGHOME=${AE_GPG_HOME}

export GEN_FILE ?= ${SITEKEY}.gen-key
export KEY_FILE ?= ${SITEKEY}.asc
export LST_FILE ?= ${SITEKEY}.list
export APT_PUBKEY_NAME=$(shell basename "${KEY_FILE}")
export APT_PUBKEY_FILE=/${APT_PUBKEY_NAME}
export APT_SRCLST_NAME=$(shell basename "${LST_FILE}")
export APT_SRCLST_FILE=/${APT_SRCLST_NAME}

export APT_BASEURL=${SITEURL}/${APT_FLAVOUR}
export APT_CONFPATH=${AE_BASEPATH}/${APT_FLAVOUR}/conf
export APT_PKG_PATH=apt-package
export APT_PKG_SITE=${APT_PKG_PATH}/${SITEKEY}
export APT_DEB_FILE=${SITEKEY}_${PKG_VERSION}_all.deb
export LATEST_DEB=${SITEKEY}_latest.deb
export APT_DEB_DEST=${AE_ARCHIVES}/${APT_FLAVOUR}/${APT_DEB_FILE}
export SKIP_APT_PACKAGE=$(shell if [ -f "${APT_DEB_DEST}" ]; then echo "${APT_DEB_DEST}"; fi)

#
#: standard enjin variables
#

APP_NAME    ?= be-apt-enjin
APP_SUMMARY ?= ${SITEKEY} apt repository

COMMON_TAGS = locals,stock_pgc,page_search,bleve_fts,page_robots,header_proxy,papertrail,htmlify
BUILD_TAGS = prd,embeds,$(COMMON_TAGS)
DEV_BUILD_TAGS = dev,$(COMMON_TAGS)
EXTRA_PKGS =
EXTRA_CLEAN = be-*
EXTRA_LDFLAGS = \
	-X 'main.SiteTag=${SITETAG}' \
	-X 'main.SiteAptUrl=${SITEURL}' \
	-X 'main.SiteName=${SITENAME}' \
	-X 'main.SiteTagLine=${APP_SUMMARY}' \
	-X 'main.PkgSection=${PKG_SECTION}' \
	-X 'main.AptFlavour=${APT_FLAVOUR}' \
	-X 'main.AptCodename=${APT_CODENAME}' \
	-X 'main.AptComponents=${APT_COMPONENTS}' \
	-X 'main.AptArchitectures=${APT_ARCHITECTURES}' \
	-X 'main.AptPublicKeyFile=${APT_PUBKEY_FILE}' \
	-X 'main.AptSourcesListFile=${APT_SRCLST_FILE}' \
	-X 'main.SetupDebUrl=/${LATEST_DEB}' \
	-X 'main.SetupDebName=${LATEST_DEB}'

# Custom go.mod locals
GOPKG_KEYS = SET GOXT DJHT

# Semantic Enjin Theme
SET_GO_PACKAGE = github.com/go-enjin/semantic-enjin-theme
SET_LOCAL_PATH = ../semantic-enjin-theme

# Go-Enjin gotext package
GOXT_GO_PACKAGE = github.com/go-enjin/golang-org-x-text
GOXT_LOCAL_PATH = ../golang-org-x-text

# Go-Enjin times package
DJHT_GO_PACKAGE = github.com/go-enjin/github-com-djherbis-times
DJHT_LOCAL_PATH = ../github-com-djherbis-times

EXTRA_BUILD_TARGET_DEPS = _update_gpg _prepare_sources_list

define pre_run =
if [ ! -d "${AE_ARCHIVES}/${APT_FLAVOUR}" ]; then \
	echo "# apt-enjin directory not found: ${AE_ARCHIVES}/${APT_FLAVOUR}" 1>&2; \
	false; \
fi
endef

define _get_gpg_key_id =
$(shell \
	if [ -n "${AE_SIGN_KEY}" ]; then \
		if [ -z "$${GNUPGHOME}" ]; then \
			export GNUPGHOME=`realpath .gpg`; \
		fi; \
		if [ -d "$${GNUPGHOME}" ]; then \
			gpg --list-keys "${AE_SIGN_KEY}" \
				| head -2 \
				| tail -1 \
				| awk '{print $$1}'; \
		fi; \
	fi)
endef

include ./Enjin.mk

#
#: apt-enjin gpg shell scripts
#

##: BEGIN MAKE GPG KEY SH
define __make_gpg_key_sh =
usage () { while [ $# -gt 0 ]; do echo "error: $1" 1>&2; shift; done; };
[ -z "${SITEKEY}" ] && usage "SITEKEY not found";
[ -z "${SITENAME}" ] && usage "SITENAME not found";
[ -z "${SITEMAIL}" ] && usage "SITEMAIL not found";
DST_GPG_FILE=${SITEKEY}.gpg;
DST_ASC_FILE=${SITEKEY}.asc;
GEN_KEY_FILE=${SITEKEY}.gen-key;
[ -f "${DST_GPG_FILE}" ] && usage "${DST_GPG_FILE} exists already";
echo "# making gpg key: ${DST_GPG_FILE}";
echo "# using gpg home: ${GNUPGHOME}"
[ -z "${GNUPGPHOME}" ] && export GNUPGHOME=$(realpath ".gpg");
[ ! -d "${GNUPGHOME}" ] && mkdir -vp "${GNUPGHOME}";
chmod -v 0700 "${GNUPGHOME}";
echo "# generating gpg key...";
if [ ! -f ${GEN_KEY_FILE} ]
then
    cat - > ${GEN_KEY_FILE} <<EOT
Key-Type: 1
Key-Length: 4096
Subkey-Type: 1
Subkey-Length: 4096
Name-Real: ${SITEMAINT}
Name-Email: ${SITEMAIL}
Expire-Date: 0
EOT
fi;
gpg --pinentry-mode="loopback" --passphrase "" \
    --no-tty --command-fd 0 \
    --batch --gen-key ${GEN_KEY_FILE};
echo "# exporting source and public key files...";
gpg --export-secret-keys="${SITEMAIL}" > ${DST_GPG_FILE};
gpg --armor --export="${SITEMAIL}" > ${DST_ASC_FILE};
endef
export _make_gpg_key_sh = $(value __make_gpg_key_sh)
##: END MAKE GPG KEY SH

##: BEGIN IMPORT GPG KEY SH
define __import_gpg_key_sh =
usage () { while [ $# -gt 0 ]; do echo "error: $1" 1>&2; shift; done; };
[ -z "${SITEKEY}" ] && usage "SITEKEY not found";
[ -z "${SITEMAIL}" ] && usage "SITEMAIL not found";
[ -z "${AE_GPG_FILE}" ] && AE_GPG_FILE=${SITEKEY}.gpg;
[ -z "${AE_SIGN_KEY}" ] && AE_SIGN_KEY=${SITEMAIL};
[ ! -f "${AE_GPG_FILE}" ] && usage "${AE_GPG_FILE} not found";
echo "# importing gpg key: ${AE_GPG_FILE}";
echo "# using gpg home: ${GNUPGHOME}"
DST_ASC_FILE=${SITEKEY}.asc;
[ -z "${GNUPGPHOME}" ] && export GNUPGHOME=$(realpath ".gpg");
[ ! -d "${GNUPGHOME}" ] && mkdir -vp "${GNUPGHOME}";
chmod -v 0700 "${GNUPGHOME}";
echo "# importing gpg key...";
gpg --import "${AE_GPG_FILE}";
if [ ! -f "${DST_ASC_FILE}" ]; then
    echo "# exporting pubkey...";
    gpg --armor --export="${AE_SIGN_KEY}" > ${DST_ASC_FILE};
fi
endef
export _import_gpg_key_sh = $(value __import_gpg_key_sh)
##: END IMPORT GPG KEY SH

#
#: apt-enjin custom targets
#

_write_sitefile:
	@if [ ! -f Sitefile ]; then \
		echo "# writing Sitefile..."; \
		( \
			echo "SITEKEY=${SITEKEY}"; \
			echo "SITEURL=${SITEURL}"; \
			echo "SITENAME=${SITENAME}"; \
			echo "SITEMAIL=${SITEMAIL}"; \
			echo "SITEMAINT=${SITEMAINT}"; \
			echo "PKG_SECTION=${PKG_SECTION}"; \
			echo "PKG_VERSION=${PKG_VERSION}"; \
			echo "APT_FLAVOUR=${APT_FLAVOUR}"; \
			echo "APT_CODENAME=${APT_CODENAME}"; \
			echo "APT_COMPONENTS=${APT_COMPONENTS}"; \
			echo "APT_ARCHITECTURES=${APT_ARCHITECTURES}"; \
		) | tee Sitefile; \
	fi

write-config: _write_sitefile

_setup_gpg: _enjenv
	@if [ ! -f "${AE_GPG_FILE}" ]; then \
		${CMD} eval "$${_make_gpg_key_sh}"; \
	else \
		${CMD} eval "$${_import_gpg_key_sh}"; \
	fi

_update_gpg:
	@if [ -f "${KEY_FILE}" ]; then \
		if [ ! -f "public/${KEY_FILE}" ]; then \
			echo "# copying ${KEY_FILE} to ./public/"; \
			cp -v ${KEY_FILE} public/; \
		fi; \
	fi

_prepare_gpg: _setup_gpg _update_gpg
	@if [ -n "$(call _get_gpg_key_id)" ]; then \
		echo "# gpg key <${AE_SIGN_KEY}> verified"; \
	else \
		echo "# gpg key <${AE_SIGN_KEY}> not found"; \
		false; \
	fi

_prepare_sources_list:
	@if [ ! -f "${LST_FILE}" ]; then \
		echo "# ${APP_SUMMARY}" > ${LST_FILE}; \
		for component in ${APT_COMPONENTS}; do \
			echo "deb ${APT_BASEURL} ${APT_CODENAME} $${component}" >> ${LST_FILE}; \
			echo "deb-src ${APT_BASEURL} ${APT_CODENAME} $${component}" >> ${LST_FILE}; \
		done; \
	fi
	@if [ ! -f "public/${LST_FILE}" ]; then \
		echo "# copying ${LST_FILE} to ./public/"; \
		cp -v ${LST_FILE} public/; \
	fi

_prepare_apt_repository: export APT_CONF_DISTS=${APT_CONFPATH}/distributions
_prepare_apt_repository: _prepare_gpg
	@if [ ! -d "${APT_CONFPATH}" -o ! -f "${APT_CONF_DISTS}" ]; then \
		echo "# preparing: ${APT_CONF_DISTS}"; \
		mkdir -vp ${APT_CONFPATH}; \
		KEY_ID=$(call _get_gpg_key_id); \
		echo "Codename: ${APT_CODENAME}"            > ${APT_CONF_DISTS}; \
		echo "Components: ${APT_COMPONENTS}"       >> ${APT_CONF_DISTS}; \
		echo "Architectures: ${APT_ARCHITECTURES}" >> ${APT_CONF_DISTS}; \
		echo "SignWith: $${KEY_ID}"                >> ${APT_CONF_DISTS}; \
	else \
		echo "# found prepared: ${APT_CONF_DISTS}"; \
	fi

_prepare_apt_package: _prepare_gpg _prepare_sources_list
	@echo "# preparing: ${APT_PKG_SITE}"
	@if [ ! -d ${APT_PKG_SITE} ]; then \
		mkdir -vp ${APT_PKG_SITE}; \
		cp -v ${KEY_FILE} ${APT_PKG_SITE}/; \
		cp -v ${LST_FILE} ${APT_PKG_SITE}/; \
		cp -v _tmpl/Debian.mk ${APT_PKG_SITE}/Makefile; \
		pushd ${APT_PKG_SITE} > /dev/null; \
		$(MAKE) debian; \
		popd > /dev/null; \
	fi

_build_apt_package: _prepare_apt_package
	@echo "# building apt-package"
	@mkdir -vp ${AE_ARCHIVES}/${APT_FLAVOUR}
	@if [ ! -f "${APT_DEB_DEST}" ]; then \
		KEY_ID=$(call _get_gpg_key_id); \
		pushd ${APT_PKG_SITE} > /dev/null; \
			dpkg-buildpackage --build=full --post-clean --sign-key="${KEY_ID}"; \
		popd > /dev/null; \
		if [ -f "${APT_PKG_PATH}/${APT_DEB_FILE}" ]; then \
			mv -v ${APT_PKG_SITE}_* ${AE_ARCHIVES}/${APT_FLAVOUR}/; \
			cp -v ${AE_ARCHIVES}/${APT_FLAVOUR}/${APT_DEB_FILE} public/${LATEST_DEB}; \
		fi; \
	fi

ifeq (${SKIP_APT_PACKAGE},)
build-apt-package: _write_sitefile _build_apt_package
else
build-apt-package: _write_sitefile _prepare_gpg _prepare_sources_list
endif
build-apt-package:
	@if [ -f "${APT_DEB_DEST}" ]; then \
		echo "# build-apt-package found: ${APT_DEB_DEST}"; \
		cp -v "${APT_DEB_DEST}" public/${LATEST_DEB}; \
	else \
		echo "# build-apt-package missing: ${APT_DEB_DEST}"; \
		false; \
	fi

process-apt-archives:
	@for src in ${AE_ARCHIVES}/${APT_FLAVOUR}/*.dsc; do \
		echo "# calling reprepro include dsc: $${src}"; \
		reprepro -s -s -b ${AE_BASEPATH}/${APT_FLAVOUR} includedsc bullseye $${src}; \
	done
	@for src in ${AE_ARCHIVES}/${APT_FLAVOUR}/*.deb; do \
		echo "# calling reprepro include deb: $${src}"; \
		reprepro -s -s -b ${AE_BASEPATH}/${APT_FLAVOUR} includedeb bullseye $${src}; \
	done

build-apt-repository: _prepare_apt_repository build-apt-package process-apt-archives

from-scratch: clean
	@echo "#"
	@echo "# make from-scratch is a fully destructive process"
	@echo "# the following files and directories will be removed:"
	@echo "#"
	@echo "#  rm -rfv ./.gpg ./apt-package ./apt-archives ./apt-repository"
	@echo "#  rm -fv ./Sitefile ${KEY_FILE} ${LST_FILE} ${GEN_FILE}"
	@echo "#  rm -fv ./public/${KEY_FILE} ./public/${LST_FILE} ./public/${LATEST_DEB}"
	@echo "#"
	@echo "#"
	@echo "# please type: \"understood\" to continue"
	@echo "#"
	@if read -e -p "> " ANSWER; then \
		case "$${ANSWER}" in \
			'"understood"') \
				;; \
			*) \
				echo "# stopping now, answer was not exactly \"understood\"."; \
				false; \
				;; \
		esac; \
		echo "# proceeding to make from-scratch..."; \
	fi
	@rm -rfv ./.gpg ./apt-package ./apt-archives ./apt-repository
	@rm -fv ./Sitefile ${KEY_FILE} ${LST_FILE} ${GEN_FILE}
	@rm -fv ./public/${KEY_FILE} ./public/${LST_FILE} ./public/${LATEST_DEB}
