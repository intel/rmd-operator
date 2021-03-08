FROM clearlinux/os-core:latest 

ENV OPERATOR=/usr/local/bin/intel-rmd-node-agent \
    USER_UID=1001 \
    USER_NAME=intel-rmd-node-agent

# install operator binary
COPY build/_output/bin/intel-rmd-node-agent ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup
