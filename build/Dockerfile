FROM clearlinux/os-core:latest 

ENV OPERATOR=/usr/local/bin/intel-rmd-operator \
    USER_UID=1001 \
    USER_NAME=intel-rmd-operator

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
