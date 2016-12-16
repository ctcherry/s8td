#!/bin/bash

echo "running"

KEYFILE="/Users/USER/.s8t.secret"
MPATH="/Users/USER/Desktop"
UPURL="http://yourserver.com/upload"

KEY="$(echo -n $(cat $KEYFILE))"
SFILE="$(find $MPATH -mtime -5s -type f -name 'Screen Shot*' | head -1)"

if [[ -n "$SFILE" ]]; then
  TS="$(date +%s)"
  SIG=$(echo -n $(echo -n "$TS" | openssl dgst -sha1 -hmac "$KEY"))
  SHORTF=${SFILE##*/}
  $(curl -F file=@"$SFILE" -F ts=$TS -F sig=$SIG $UPURL | pbcopy; osascript -e "display notification \"$SHORTF uploaded\" with title \"Screenshot\"" &)
fi

COUNTER=0
while [ $COUNTER -lt 30 ]; do
  sleep 2
  SFILE="$(find $MPATH -mtime -2s -type f -name 'Screen Shot*' | head -1)"
  if [[ -n "$SFILE" ]]; then
    TS="$(date +%s)"
    SIG=$(echo -n $(echo -n "$TS" | openssl dgst -sha1 -hmac "$KEY"))
    SHORTF=${SFILE##*/}
    $(curl -F file=@"$SFILE" -F ts=$TS -F sig=$SIG $UPURL | pbcopy; osascript -e "display notification \"$SHORTF uploaded\" with title \"Screenshot\"" &)
  fi

  let COUNTER=COUNTER+1 
done
