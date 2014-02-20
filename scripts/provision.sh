#!/bin/sh
set -e

# Ensure that agent forwarding is set up properly.
if [ "$SSH_AUTH_SOCK" ]; then
    echo "SSH_AUTH_SOCK is set; you are successfully agent-forwarding. These keys are loaded:"
    if [ -z "$1" ]; then
        ssh-add -l
    else
        echo "Not attempting to list keys because windows ssh-agent communication is broken..."
    fi
else
    echo "No SSH_AUTH_SOCK was found in the environment!"
    exit 3
fi

# apt-get update.
if [ -e /root/package-list-updated ]; then
    echo "Skipping package cache update. To force, remove /root/package-list-updated and re-provision."
else
    echo "Updating package cache."
    sudo apt-get update -qq
	touch /root/package-list-updated
fi

echo "Installing Packages..."
export DEBIAN_FRONTEND=noninteractive
( sed -e 's/#.*$//' | xargs sudo apt-get install -qqy --force-yes ) <<-EOF
	git

	# Stuff required by medley
	python-software-properties	# TODO why do we need this?
	build-essential				# needed to compile parts of packages
	curl						# many scripts expect this to fetch urls.
	python-dev					# for compiling python modules
	python-imaging				# Useful if you do not want to compile PIL
	python-pip					# for installing things
	python-psycopg2				# python postgresql library
	python-setuptools			# for installing/making packages
	python-unittest2			# standard unit testing library
	python-virtualenv			# for partioning python projects
	pv							# "pipe viewer", for nice progressbars in med
    exuberant-ctags             # required by 'med tags'

	# geospatial libraries
	libgdal1-dev
	libgdal1-1.7.0
	libgeos-3.2.2
	libgeos-c1
	libgeos-dev

	python-lxml					# TODO why do we need this?
	libxml2						# TODO why do we need this?
	libxml2-dev					# TODO why do we need this?
	libxslt1-dev				# TODO why do we need this?

	postgresql-9.0				# our database.
	postgresql-contrib-9.0		# django wants this
	postgresql-server-dev-9.0	# TODO why do we need this?

	# postgresql-9.0-postgis is not available in standard repos, so we install
	# my custom package later.
	#postgresql-9.0-postgis 	# utilize geo stuff in postgres


	# Development helpers that I'm asserting don't need to be installed by
	# default for everyone using this thing, they can just install them when
	# they need them.
	# ack-grep					# adreyer thinks you should have it
	# ipython					# adreyer thinks you should have it
	# bpython					# nksmith thinks you should have it
	# memcached					# django can work around not having this
	# proj						# TODO apt says it is transitional. wtf is it for?
	# pyflakes					# adreyer thinks you should have it
	# pylint					# adreyer thinks you should have it
	# python-pycryptopp			# adreyer thinks you should have it
	# virtualenvwrapper			# convenience tool
	# rabbitmq-server			# adreyer thinks you should have it
	# vim						# adreyer thinks you should have it

	openjdk-7-jre-headless		# needed to run solr
EOF
