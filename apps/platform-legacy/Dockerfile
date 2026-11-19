FROM nginxinc/nginx-unprivileged:1.31-alpine@sha256:b5047b9b76845eac124358373c6852377c52e4b05834a47647e955dd91cb8f1e

USER root
RUN apk upgrade --no-cache
USER 101:101

ENV SERVER_NAME=_ \
    BASE_URL=/ \
    API_HOST=http://gateway:5000/ \
    AUTH_ROOT_URL=http://localhost:35000

COPY --chown=101:101 dist/ /opt/epos-gui/
COPY nginx.conf /etc/nginx/conf.d/default.conf

WORKDIR /opt/epos-gui/
USER 101:101

CMD escaped_auth_root_url=$(printf '%s' "$AUTH_ROOT_URL" | sed 's/[&|]/\\&/g') && \
    find /opt/epos-gui -name '*.js' -exec sed -i "s|__AUTH_ROOT_URL__|$escaped_auth_root_url|g" {} \; && \
    sed -Ei 's|<base href="[^"]*"[[:space:]]*/?>|<base href="'$BASE_URL'">|g' /opt/epos-gui/index.html && \
    sed -i 's|SERVER_NAME|'$SERVER_NAME'|g' /etc/nginx/conf.d/default.conf && \
    sed -i 's|BASE_URL|'$BASE_URL'|g' /etc/nginx/conf.d/default.conf && \
    sed -i 's|API_HOST|'$API_HOST'|g' /etc/nginx/conf.d/default.conf && \
    nginx -g "daemon off;"

EXPOSE 8080
