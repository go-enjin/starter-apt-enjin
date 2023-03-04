#!/usr/bin/make -f

SHELL = /bin/bash

export SITEKEY   ?= sitename-ppa
export SITEURL   ?= url://sitename
export SITENAME  ?= Site Name
export SITEMAIL  ?= site@email.address
export SITEMAINT ?= ${SITENAME}

export PKG_SECTION ?= go-enjin
export PKG_VERSION ?= 0.1.0
export PKG_CHANGES ?= Initial release.

export KEY_FILE ?= ${SITEKEY}.asc
export LST_FILE ?= ${SITEKEY}.list

export ETC_PATH ?= ${DESTDIR}/etc

export KEY_PATH ?= ${ETC_PATH}/apt/trusted.gpg.d
export LST_PATH ?= ${ETC_PATH}/apt/sources.list.d

export KEY_DEST = ${KEY_PATH}/${SITEKEY}.asc
export LST_DEST = ${LST_PATH}/${SITEKEY}.list

.PHONY: all help build install debian

help:
	@echo "usage: make <help|build|install|debian>"

build:
	@if [ ! -f "${KEY_FILE}" ]; then \
		echo "# KEY_FILE not found, expected: ${KEY_FILE}"; \
		false; \
	fi
	@if [ ! -f "${LST_FILE}" ]; then \
		echo "# LST_FILE not found, expected: ${LST_FILE}"; \
		false; \
	fi

install: build
	@mkdir -vp ${KEY_PATH} ${LST_PATH}
	@/usr/bin/install -v -b -m 0644 -T "${LST_FILE}" "${LST_DEST}"
	@/usr/bin/install -v -b -m 0644 -T "${KEY_FILE}" "${KEY_DEST}"

debian:
	@if [ -d debian ]; then \
		echo "# debian directory found, nothing to do"; \
	else \
		eval "$$__make_debian_value"; \
	fi

define __make_debian_script =
mkdir -v debian debian/source
pushd debian > /dev/null

#: source dir
echo "3.0 (native)" > source/format

#: package install file
echo "etc/apt/sources.list.d/${SITEKEY}.list /etc/apt/sources.list.d/" > ${SITEKEY}.install
echo "etc/apt/trusted.gpg.d/${SITEKEY}.asc /etc/apt/trusted.gpg.d/" >> ${SITEKEY}.install

#: changelog file
cat - > changelog <<EOT
${SITEKEY} (${PKG_VERSION}) unstable; urgency=medium

  * ${PKG_CHANGES}

 -- ${SITEMAINT} <${SITEMAIL}>  $(date +'%a, %d %b %Y %H:%M:%S %z')
EOT

#: control file
cat - > control <<EOT
Source: ${SITEKEY}
Section: ${PKG_SECTION}
Priority: optional
Maintainer: ${SITEMAINT} <${SITEMAIL}>
Build-Depends: debhelper-compat (= 13)
Standards-Version: 4.5.1
Homepage: ${SITEURL}
Rules-Requires-Root: no

Package: ${SITEKEY}
Architecture: all
Depends: \${misc:Depends}
Description: ${SITENAME} PPA configuration
 This package adds the ${SITEKEY} debian package repository
 (and gpg key) to your local apt installation.
 .
 Remember to update apt after installing or removing this package.
EOT


#: rules file
cat - > rules <<EOT
#!/usr/bin/make -f

export DH_VERBOSE=1

override_dh_auto_install: export DESTDIR=\${CURDIR}/debian/tmp
override_dh_auto_install:
	\$(MAKE) install

%:
	dh \$@
EOT
chmod +x rules

#: postrm file
cat - > postrm <<EOT
#!/bin/bash
set -e
case "\$1" in
    remove)
    ;;
		purge|upgrade|failed-upgrade|abort-install|abort-upgrade|disappear)
		;;
    *)
        echo "postrm called with unknown argument '\$1'" >&2
        exit 1
    ;;
esac
exit 0
EOT
chmod +x postrm

#: copyright file
YEAR=$(date "+%Y")
cat - > copyright <<EOT
ormat: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: ${SITEKEY}
Upstream-Contact: ${SITEMAINT} <${SITEMAIL}>
Source: <${SITEURL}>

Files: *
Copyright: ${YEAR} ${SITEMAINT} <${SITEMAIL}>
License: Apache-2.0

Files: debian/*
Copyright: ${YEAR} ${SITEMAINT} <${SITEMAIL}>
License: Apache-2.0

License: Apache-2.0
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 .
 https://www.apache.org/licenses/LICENSE-2.0
 .
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
 .
 On Debian systems, the complete text of the Apache version 2.0 license
 can be found in "/usr/share/common-licenses/Apache-2.0".
EOT

popd > /dev/null
endef
export __make_debian_value = $(value __make_debian_script)
