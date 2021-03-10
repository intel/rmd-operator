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

ENV SHELL=/bin/bash
ENV LD_LIBRARY_PATH=/usr/local/lib
ENV DEVICEPLUGIN=/usr/local/bin/intel-rmd-deviceplugin

COPY build/_output/bin/intel-rmd-deviceplugin ${DEVICEPLUGIN}
