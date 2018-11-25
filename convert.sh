#!/bin/sh
if [ $3 = 'image' ]; then
  echo " vipsthumbnail $1"
  vipsthumbnail -s 100 "$1" -o "$2"
elif [ $3 = 'video' ]; then
  echo " ffmpeg $1"
  ffmpeg -v warning -ss 0 -t 10 -i "$1" -vf fps=10,scale=100:-1 -gifflags -transdiff -n "$2"
else if [ $3 != 'audio' ]; then
  echo " convert $1"
  convert -background White -alpha Background -resize 100x100 -flatten "$1" "$2"
fi