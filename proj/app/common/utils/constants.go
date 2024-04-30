package utils

const GRS_NETWORK string = "grs-net"
const GRS_IMAGE string = "grs"
const GRS_LOAD_BALANCER string = "load_balancer"

const NGINX_CONFIG_PATH string = "../load_balancer/config.conf"
const NGINX_DEFAULT_CONF string = `
pid /run/nginx;

events {
	worker_connections 768;
}

http {
	server {
    		listen localhost;

    		server_name status.localhost;

            allow all;
	}
}
`

const NGINX_LOAD_BALANCER_CONF string = `
events {
    worker_connections 1024;
}

http {
    upstream load_balancer {
        server modest_mcnulty:80;
    }

    server {
        listen 80;

        location / {
            proxy_pass http://load_balancer;
        }
    }
}
`
