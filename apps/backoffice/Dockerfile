FROM nginxinc/nginx-unprivileged:1.31.7-alpine@sha256:b5047b9b76845eac124358373c6852377c52e4b05834a47647e955dd91cb8f1e

USER root
RUN apk upgrade --no-cache
ENV API_HOST=http://gateway:5000 \
    BASE_URL=/

COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY dist/epos-backoffice-gui/browser/ /usr/share/nginx/html/
RUN chown -R 101:101 /usr/share/nginx/html
USER 101:101

CMD sed -i "s|<base href=\"/\">|<base href=\"$BASE_URL\">|" /usr/share/nginx/html/index.html && \
    exec /docker-entrypoint.sh nginx -g 'daemon off;'

EXPOSE 8080
