FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update
RUN apt-get install -y ttyd bash vim nano tree git 
RUN apt-get install -y gcc g++ make python3 python3-venv python3-pip
RUN apt-get install -y iputils-ping net-tools curl wget
RUN apt-get install -y procps locales ca-certificates tzdata passwd util-linux
RUN rm -rf /var/lib/apt/lists/*

RUN locale-gen en_US.UTF-8

ARG CONTAINER_RUNTIME_USER=linuxus

ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Create a fixed runtime user at build time
RUN groupadd -g 1000 $CONTAINER_RUNTIME_USER && useradd -m -u 1000 -g 1000 -s /bin/bash $CONTAINER_RUNTIME_USER

COPY start.sh /start.sh
RUN chmod +x /start.sh

CMD ["/start.sh"]