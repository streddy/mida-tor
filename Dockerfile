FROM ubuntu

RUN apt-get update && apt-get -y upgrade && apt-get -y install \
  python3 ca-certificates chromium-browser tcpdump apt-utils software-properties-common

RUN apt-get -y install wget
RUN apt-get -y install curl
RUN apt -y install apt-transport-https && add-apt-repository universe
RUN wget -q -O - https://deb.torproject.org/torproject.org/A3C4F0F979CAA22CDBA8F512EE8CBC9E886DDD89.asc | apt-key add -
RUN echo "deb https://deb.torproject.org/torproject.org $(lsb_release -cs) main" | tee -a /etc/apt/sources.list
RUN apt update && apt -y install tor deb.torproject.org-keyring

COPY setup.py /root

RUN python3 /root/setup.py 

COPY pcap-script.sh /root

RUN chmod +x /root/pcap-script.sh

ENTRYPOINT ["/root/pcap-script.sh"]
