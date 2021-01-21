FROM golang:1.13 AS build

# Set proxies
#ENV http_proxy=
#ENV https_proxy=
#ENV ftp_proxy=
#ENV socks_proxy=
#ENV no_proxy=

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
COPY build/manifests/rmd-node-agent-ds.yaml /rmd-manifests/rmd-node-agent-ds.yaml
COPY build/manifests/rmd-ds.yaml /rmd-manifests/rmd-ds.yaml
COPY build/certs /etc/certs
RUN  /usr/local/bin/user_setup
RUN chmod -R 650 /etc/certs/private

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
