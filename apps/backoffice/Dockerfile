FROM nginxinc/nginx-unprivileged:stable-alpine

USER 101:101
ENV API_HOST=http://gateway:5000

COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY dist/epos-backoffice-gui/browser/ /usr/share/nginx/html/

EXPOSE 8080
