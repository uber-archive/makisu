FROM debian:9 AS phase1
RUN mkdir /tmp/dst1 && ln -s /tmp/dst1 /tmp/link1 && touch /tmp/dst1/test.txt
RUN mkdir /tmp/dst2 && ln -s /tmp/dst2 /tmp/link2
RUN mkdir -p /var/run/umakisu/test1 && chmod -R 777 /var/run/umakisu

FROM debian:9
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
RUN groupadd --gid 234567 umakisu
RUN useradd --uid 234567 -g umakisu -d /home/umakisu -m -s /bin/bash umakisu
COPY --from=phase1  --chown=umakisu:umakisu /var/run/umakisu /var/run/umakisu

ENTRYPOINT ["/bin/sh", "-c", "cat /mine/link2/test.txt && cat /mine/link2/new/test.txt && cat /mine/test.txt && cat /mine/new/test.txt && [ $(stat -c %a /run/umakisu/test1) -eq \"777\" ] && echo \"passed\" || exit 1"]
