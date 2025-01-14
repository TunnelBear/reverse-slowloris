# reverse-slowloris

This is a take on the
[slowloris attack](https://en.wikipedia.org/wiki/Slowloris_(computer_security))
except done in reverse. Anyone who connects to this server will be sent an
infinite slow stream of data until they terminate the connection.

This came up when I was looking through my nginx access logs and saw many
requests for endpoints in search of security holes, like `/phpmyadmin`,
`/.git/HEAD` and many others. I took a sample of the most common endpoints
requested and added a block like this to my nginx configuration

    location /wp-login.php {
        proxy_pass http://localhost:8080;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_request_buffering off;
        proxy_buffering off;
    }
    location /mysql {
        proxy_pass http://localhost:8080;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_request_buffering off;
        proxy_buffering off;
    }
    location /databases {
        proxy_pass http://localhost:8080;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_request_buffering off;
        proxy_buffering off;
    }

I would then include this in any configurations in my nginx `conf.d` directory
for easy reuse. I also adjusted the base nginx conf to proxy into this server,
for some handling when no hostname is specified (connecting via IP address)

I don't know if this annoys the bots at all, but I have seen some stay connected
for over 14 hours.


## notes for CloudFlare

- CloudFlare might kill the connection after 100 seconds ([docs](https://support.cloudflare.com/hc/en-us/articles/115003011431-Troubleshooting-Cloudflare-5XX-errors#524error))
- Or up to 100 minutes if you have enterprise and ([api](https://api.cloudflare.com/#zone-settings-change-proxy-read-timeout-setting)) bump the proxy read timeout. the higher timeout will apply across the entire zone though.
- (or maybe CF won't time it out since it _is_ sending response bytes the whole time?)
