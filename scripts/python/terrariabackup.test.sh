#!/bin/bash

# Create folder structure with files
testdir="__terrariabackup__"
mkdir $testdir
mkdir $testdir/content
mkdir $testdir/content/sub1
mkdir $testdir/content/sub1/subsub1
mkdir $testdir/content/sub2

touch $testdir/content/content1.txt
touch $testdir/content/content2.txt
touch $testdir/content/sub1/sub1_1.txt
touch $testdir/content/sub1/sub1_2.txt
touch $testdir/content/sub1/subsub1/subsub1_1.txt

echo "content1" > $testdir/content/content1.txt
echo "content2" > $testdir/content/content2.txt
echo "sub1_1" > $testdir/content/sub1/sub1_1.txt
echo "sub1_2" > $testdir/content/sub1/sub1_2.txt
echo "subsub1_1" > $testdir/content/sub1/subsub1/subsub1_1.txt

# Backup folder
python ./terrariabackup.py -i "$testdir/content" -o "$testdir/content.zip"

read
