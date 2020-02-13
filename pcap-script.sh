#!/bin/bash

service tor start
#curl --socks5 localhost:9050 --socks5-hostname localhost:9050 -s https://check.torproject.org/ | cat | grep -m 1 Congratulations | xargs

tcpdump -i eth0 -w /trace.pcap & 

export SSLKEYLOGFILE=/ssl.log

#/usr/local/bin/mida go --add-browser-flags=headless,disable-gpu $@
/usr/local/bin/mida go --add-browser-flags=headless,disable-gpu,remote-debugging-port=9222,proxy-server="socks5://localhost:9050" --completion=CompleteOnLoadEvent --timeout=60 $@
#chromium-browser --no-sandbox --headless --remote-debugging-port=9222 --disable-gpu --proxy-server=“socks5://localhost:9050” $@ &
#sleep 2 &
kill -1 %%

sleep 1

mv -v /trace.pcap /ssl.log /results/*/*/

cp -av /results/* /data


