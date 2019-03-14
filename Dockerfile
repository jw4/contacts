#
# Build on Debian Stretch
#

FROM golang:stretch as builder

COPY . /src/contacts

WORKDIR /src/contacts

ARG BUILD_VERSION=v0.0.0

ENV BUILD_VERSION ${BUILD_VERSION}

RUN go get -v -u ./...
RUN go build -tags netgo -ldflags="-s -w -X jw4.us/contacts.Version=${BUILD_VERSION}" -o server ./cmd/server/


#
# Create Image on scratch
#

FROM scratch

LABEL maintainer="John Weldon <johnweldon4@gmail.com>" \
      company="John Weldon Consulting" \
      description="Contacts Server"

COPY --from=builder /src/contacts/server /server
COPY public /public/
COPY templates /templates/

ENV PORT 8818
ENV PUBLIC_FOLDER /public
ENV TEMPLATE_FOLDER /templates

ENV LDAP_HOST ldap
ENV LDAP_PORT 389
ENV LDAP_USER anonymous
ENV LDAP_PASS anonymous
ENV LDAP_BASE dc=example,dc=org

EXPOSE 8818

ENTRYPOINT ["/server"]
