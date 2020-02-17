FROM centos:7
RUN yum -y update && \
    yum -y install openssh-server passwd && \
    yum clean all 
RUN SSH_USERPASS=cisco && \
    echo -e "$SSH_USERPASS\n$SSH_USERPASS" | passwd && \
    mkdir /var/run/sshd && \
    ssh-keygen -t rsa -f /etc/ssh/ssh_host_rsa_key -N ''
EXPOSE 22
ENTRYPOINT ["/usr/sbin/sshd", "-D"]
