#!/bin/bash
# Script creates git repository in $1 directory
# It creates some commits, branches and prints testable commit hash
# Repo is used for project testing

# c1 - c2 -- c4 - c5 ----- c7 - c10 <- master
#      |          |        |
#       - c3       - c6     - c8 - c9 <- b3
#         ^          ^
#         b1         b2
# change c3, c4

exec 7>&1
exec >/dev/null

mkdir -p $1

cd $1
git init
git config --local user.name "Foo bar"
git config --local user.email "foobar@mail.com"
i=1
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s

git branch b1 && git checkout b1
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
RES_HASH=$(git rev-parse HEAD) # hash of commit c3
git checkout master

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
RES_HASH="$RES_HASH$(git rev-parse HEAD)" # hash of commit c4
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s

git branch b2 && git checkout b2
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
git checkout master

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s

git branch b3 && git checkout b3
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s
git checkout master

echo $i > "$i.txt" && git add "$i.txt" && git commit -m "c$i" && i=$((i+1)) && sleep 0.01s

printf $RES_HASH >&7