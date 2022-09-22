FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/hcp-cluster-manager \
    USER_UID=root \
    USER_NAME=hcp-cluster-manager

RUN microdnf install unzip openssl

RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"

RUN unzip awscliv2.zip

RUN ./aws/install

WORKDIR /usr/local/bin

RUN curl -o ncp-iam-authenticator https://kr.object.ncloudstorage.com/nks-download/ncp-iam-authenticator/v1.0.5/linux/amd64/ncp-iam-authenticator

RUN chmod 777 ncp-iam-authenticator

WORKDIR /

# install kubectl 
RUN curl -LO "https://dl.k8s.io/release/v1.21.0/bin/linux/amd64/kubectl"

RUN install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# install operator binary
COPY _output/bin/hcp-cluster-manager ${OPERATOR}
COPY bin /usr/local/bin

RUN mkdir /usr/local/kubeconfig
RUN  /usr/local/bin/user_setup
RUN chmod +x /usr/local/bin/entrypoint

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}