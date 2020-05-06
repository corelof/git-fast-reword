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
# change c2, c5

exec 7>&1
exec >/dev/null

mkdir -p $1

cd $1
git init
git config --local user.name "Foo bar"
git config --local user.email "foobar@mail.com"
i=1

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s

git branch b1 && git checkout b1
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
RES_HASH="$(git rev-parse HEAD)" # hash of commit c2
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
git checkout master

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
git merge -m "merge" --no-ff b1 && sleep 0.01s

git branch -d b1

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
RES_HASH="$RES_HASH$(git rev-parse HEAD)" # hash of commit c5

git branch b2 && git checkout b2
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
git checkout master

git branch b3 && git checkout b3
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
git checkout master

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s

printf $RES_HASH >&7