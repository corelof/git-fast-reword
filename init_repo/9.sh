#!/bin/bash
# Script creates git repository in $1 directory
# It creates some commits, branches and prints testable commit hash
# Repo is used for project testing

#                         - c7 <- b3
#                        |
# c1 ---- c4 -- merge -- c5 -- c8 <- master
# |             |        |
#  - c2 - c3 ---          - c6 <- b2
#
# change all c1
# lightweight tags: c2, c5
# annotated tags: merge, c1

exec 7>&1
exec >/dev/null

mkdir -p $1

cd $1
git init
git config --local user.name "Foo bar"
git config --local user.email "foobar@mail.com"
i=1

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && RES_HASH="$(git rev-parse HEAD)"
A1="$(git rev-parse HEAD)"

git branch b1 && git checkout b1
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && RES_HASH="$RES_HASH$(git rev-parse HEAD)"
L1="$(git rev-parse HEAD)"
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && RES_HASH="$RES_HASH$(git rev-parse HEAD)"
git checkout master

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && RES_HASH="$RES_HASH$(git rev-parse HEAD)"
git merge -m "merge" --no-ff b1

git branch -d b1

A2="$(git rev-parse HEAD)"

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && RES_HASH="$RES_HASH$(git rev-parse HEAD)"
L2="$(git rev-parse HEAD)"

git branch b2 && git checkout b2
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && RES_HASH="$RES_HASH$(git rev-parse HEAD)"
git checkout master

git branch b3 && git checkout b3
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && RES_HASH="$RES_HASH$(git rev-parse HEAD)"
git checkout master

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && RES_HASH="$RES_HASH$(git rev-parse HEAD)"

printf $RES_HASH >&7

git tag lw1 $L1
git tag lw2 $L2
git tag -a an1 -m "message an1" $A1
git tag -a an2 -m "message an2" $A2