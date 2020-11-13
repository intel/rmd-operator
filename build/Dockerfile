FROM golang:1.13 AS build

# Set proxies
ENV http_proxy=http://proxy-chain.intel.com:911
ENV https_proxy=http://proxy-chain.intel.com:911
ENV ftp_proxy=http://proxy-chain.intel.com:911
ENV socks_proxy=http://proxy-us.intel.com:1080
ENV no_proxy=intel.com,.intel.com,localhost,127.0.0.1,10.0.0.0/8,192.168.0.0/16,172.16.0.0/12

# Pull intel-cmt-cat
RUN mkdir -p /home/intel-cmt-cat \
           cd /home/intel-cmt-cat
RUN git clone https://github.com/intel/intel-cmt-cat.git
WORKDIR /go/intel-cmt-cat
RUN make install

FROM clearlinux/os-core:latest
COPY --from=build /go/intel-cmt-cat/lib/* /usr/local/lib/

ENV OPERATOR=/usr/local/bin/intel-rmd-operator \
    USER_UID=1001 \
    USER_NAME=intel-rmd-operator \
    LD_LIBRARY_PATH=/usr/local/lib

# install operator binary
COPY build/_output/bin/intel-rmd-operator ${OPERATOR}
COPY build/bin /usr/local/bin
COPY build/manifests/rmd-node-agent.yaml /rmd-manifests/rmd-node-agent.yaml
COPY build/manifests/rmd-pod.yaml /rmd-manifests/rmd-pod.yaml
COPY build/certs /etc/certs
RUN  /usr/local/bin/user_setup
RUN chmod -R 650 /etc/certs/private

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
