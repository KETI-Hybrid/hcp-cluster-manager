FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/hcp-cluster-manager \
    USER_UID=1001 \
    USER_NAME=hcp-cluster-manager

# install operator binary
COPY _output/bin/hcp-cluster-manager ${OPERATOR}
COPY bin /usr/local/bin

RUN mkdir /usr/local/kubeconfig
RUN  /usr/local/bin/user_setup
RUN chmod +x /usr/local/bin/entrypoint

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}