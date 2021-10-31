FROM postgres:12
MAINTAINER S. Mannem <sebas@mannem.nl>
ENV PGVERSION=12
ENV PGDATA=/var/lib/postgresql/data/
ENV PGBIN=/usr/lib/postgresql/${PGVERSION}/bin

COPY entrypoint.sh /
USER 999

ENTRYPOINT ["/entrypoint.sh"]
CMD ["sleep"]
