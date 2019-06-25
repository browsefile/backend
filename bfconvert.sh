#!/bin/sh
if [ "$3" = 'image' ]; then
  #echo " vipsthumbnail $1"
  vipsthumbnail -s 100 "$1" -o "$2"
  #ffmpeg -n -v error -i "$1" -vf scale=100:-1 "$2"
elif [ "$3" = 'video' ]; then
  #echo " ffmpeg $1"
  ffmpeg -n -v error -ss 0 -t 10 -i "$1" -vf fps=10,scale=100:-1 -gifflags -transdiff -n "$2"
elif [ "$3" != 'audio' ] && [ "$3" != 'text' ]; then
  #echo " convert $1"
  convert -background White -alpha Background -resize 100x100 -flatten "$1" "$2"
fi