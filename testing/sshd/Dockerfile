FROM alpine:3.20

RUN apk add --no-cache --update openssh openssh-keygen

RUN ssh-keygen -v -A

RUN addgroup -S terraform && adduser -S terraform -G terraform && passwd -u terraform

# Create ~/.ssh up front. Newer BuildKit applies the COPY --chmod to the parent
# directories it creates implicitly, which leaves .ssh without its execute bit
# and makes sshd unable to read authorized_keys at all.
RUN install -d -m 700 -o terraform -g terraform /home/terraform/.ssh

COPY --chown=terraform:terraform --chmod=600 ./authorized_keys /home/terraform/.ssh/authorized_keys

COPY sshd_config /etc/ssh/sshd_config

ENTRYPOINT [ "/usr/sbin/sshd", "-D", "-e" ]
