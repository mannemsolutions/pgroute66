FROM postgres:12
MAINTAINER S. Mannem <sebas@mannem.nl>
ENV PGVERSION=12
ENV PGDATA=/var/lib/postgresql/data/
ENV PGBIN=/usr/lib/postgresql/${PGVERSION}/bin

RUN apt-get update && apt-get install -y iproute2 && apt-get clean

COPY entrypoint.sh /
USER 999

ENTRYPOINT ["/entrypoint.sh"]
CMD ["sleep"]
