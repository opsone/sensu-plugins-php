FROM debian:13.7-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends nginx php8.4-fpm \
 && rm -rf /var/lib/apt/lists/* \
 && sed -i 's|^;ping.path = .*|ping.path = /fpm-ping|' /etc/php/8.4/fpm/pool.d/www.conf \
 && mkdir -p /run/php
COPY nginx.conf /etc/nginx/sites-available/default
COPY check-php-fpm /usr/local/bin/check-php-fpm
CMD ["sleep", "infinity"]
