FROM centos:7
RUN rm -vf /etc/yum.repos.d/*
ENTRYPOINT ["/bin/bash", "-c", "echo $(ls -la /etc/yum.repos.d/)"]
