#!/bin/bash
# Script creates git repository in $1 directory
# It creates some commits, branches and prints testable commit hash
# Repo is used for project testing

exec 7>&1
exec >/dev/null

mkdir -p $1

cd $1
git init
git config --local user.name "Foo bar"
git config --local user.email "foobar@mail.com"
i=1

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
RES_HASH="$(git rev-parse HEAD)"

git branch b1 && git checkout b1
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
git checkout master

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
git merge -m "merge" --no-ff b1

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
DH_START=$(git rev-parse HEAD) # hash of commit c5
RES_HASH="$RES_HASH$(git rev-parse HEAD)"

git branch b2 && git checkout b2
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
git checkout master

git branch b3 && git checkout b3
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
git checkout master

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))

git checkout $DH_START && echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
RES_HASH="$RES_HASH$(git rev-parse HEAD)"

printf $RES_HASH >&7