#!/bin/bash
# Script creates git repository in $1 directory
# It creates some commits, branches and prints testable commit hash
# Repo is used for project testing

# c1 - c2 - c3 - c4 <- master
#                |
#                 - c5 <- b1
# change c3 and c4 messages

exec 7>&1
exec >/dev/null

mkdir -p $1

cd $1
git init
git config --local user.name "Foo bar"
git config --local user.email "foobar@mail.com"
i=1
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
RES_HASH="$(git rev-parse HEAD)" # hash of commit c3
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))
RES_HASH="$RES_HASH$(git rev-parse HEAD)" # hash of commit c4

git branch b1 && git checkout b1
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1))

printf $RES_HASH >&7