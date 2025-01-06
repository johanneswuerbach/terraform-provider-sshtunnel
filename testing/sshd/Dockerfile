FROM alpine:3.20

RUN apk add --no-cache --update openssh openssh-keygen

RUN ssh-keygen -v -A

RUN addgroup -S terraform && adduser -S terraform -G terraform && passwd -u terraform

COPY --chown=terraform:terraform --chmod=600 ./authorized_keys /home/terraform/.ssh/authorized_keys

COPY sshd_config /etc/ssh/sshd_config

ENTRYPOINT [ "/usr/sbin/sshd", "-D", "-e" ]
