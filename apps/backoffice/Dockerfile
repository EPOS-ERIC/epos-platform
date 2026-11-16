FROM nginxinc/nginx-unprivileged:stable-alpine

USER root
ENV API_HOST=http://gateway:5000 \
    BASE_URL=/

COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY dist/epos-backoffice-gui/browser/ /usr/share/nginx/html/
RUN chown -R 101:101 /usr/share/nginx/html
USER 101:101

CMD sed -i "s|<base href=\"/\">|<base href=\"$BASE_URL\">|" /usr/share/nginx/html/index.html && \
    exec /docker-entrypoint.sh nginx -g 'daemon off;'

EXPOSE 8080
