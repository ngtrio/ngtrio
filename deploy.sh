#!/usr/bin/env bash

# install dependencies
npm install hexo-cli -g
npm install

# fetch mtime from git log
git ls-files -z source/_posts |
    while read -d '' path; do
    if [[ $path == *.md ]]; then
        mtime=$(git log -1 --format="@%ct" $path)
        touch -d $mtime $path
        echo "change $path mtime to $mtime"
    fi
    done

# hexo deploy
hexo clean
hexo deploy
