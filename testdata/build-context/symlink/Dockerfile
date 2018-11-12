FROM debian:8 AS phase1
RUN mkdir /tmp/dst1 && ln -s /tmp/dst1 /tmp/link1 && touch /tmp/dst1/test.txt
RUN mkdir /tmp/dst2 && ln -s /tmp/dst2 /tmp/link2
RUN mkdir -p /var/run/udocker/test1 && chmod -R 777 /var/run/udocker 

FROM debian:8
RUN mkdir /mine
# Setup.
COPY --from=phase1 /tmp/link2 /mine/link2
# Test copying to existing symlink.
# Content should be copied.
COPY --from=phase1 /tmp/dst1/test.txt /mine/link2/
# Test copying to non-existing sub-directory of existing symlink.
# Sub-directory should be created and content should be copied.
COPY --from=phase1 /tmp/dst1/test.txt /mine/link2/new/
# Test copying symlink to existing location.
# Content should be copied.
COPY --from=phase1 /tmp/link1 /mine/
# Test copying symlink to non-existing location.
# Directory should be created with default permissions.
COPY --from=phase1 /tmp/link1 /mine/new
# Test copying sub-diretory of symlink to sub-directory of existing symlink
# (/var/run is always a symlink to /run).
# Permission of sub-directory should be preserved.
RUN groupadd --gid 100001 udocker
RUN useradd --uid 100001 -g udocker -d /home/udocker -m -s /bin/bash udocker
COPY --from=phase1  --chown=udocker:udocker /var/run/udocker /var/run/udocker

ENTRYPOINT ["/bin/sh", "-c", "cat /mine/link2/test.txt && cat /mine/link2/new/test.txt && cat /mine/test.txt && cat /mine/new/test.txt && [ $(stat -c %a /run/udocker/test1) -eq \"777\" ] && echo \"passed\" || exit 1"]
