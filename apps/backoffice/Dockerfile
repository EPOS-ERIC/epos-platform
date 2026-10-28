FROM nginxinc/nginx-unprivileged:stable-alpine

USER root
ENV API_HOST=http://gateway:5000 \
    BASE_URL=/ \
    AUTH_ROOT_URL=

COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY dist/epos-backoffice-gui/browser/ /usr/share/nginx/html/
RUN chown -R 101:101 /usr/share/nginx/html
USER 101:101

CMD sed -i "s|<base href=\"/\">|<base href=\"$BASE_URL\">|" /usr/share/nginx/html/index.html && \
    find /usr/share/nginx/html -name '*.js' -exec sed -i "s|__AUTH_ROOT_URL__|$AUTH_ROOT_URL|g" {} + && \
    exec nginx -g 'daemon off;'

EXPOSE 8080
