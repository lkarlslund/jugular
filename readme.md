# jugular

Use jugs to look for CUPS: an ultrafast UDP sender, that tries to tickle listening cups-browsed processes to do a HTTP callback.

## Install

```go install github.com/lkarlslund/jugular@main```

... or download the Linux x64 binary from the builds

## Usage

1. Start jugular listening server, it's a simple HTTP server that replies with 404 to all requests, but saves the request headers in json files in current working directory - you can use something different if you want

```jugular listen --bind 0.0.0.0:80```

2. Start prodding CUPS instances, by sending UDP packets that announce a printer. Note: scanning the internet is in the grey area of legality, so be careful. If you're company IT security, you can use it internally ofcourse.

```jugular prod --network 10.0.0.0/8 --delay 1 --url http://yourdomain.net/printer/scanning-you --addip true --ip your-own-machine-ip```

Since this software creates and writes its own packets directly to an outgoing network for performance reasons, you *need* to assign your own correct machine IP address (run `ip addr` to find it). The remote instances need to be able to route back to the URL you put in that parameter as well.

Test locally before going on a hunt, and please remember that this software can overwhelm your NAT firewall and other equipment (use the delay option).

*PLEASE NOTE* There is a 300 second delay from CUPS recieving the message to it reaches out to the URL you gave it. This is due to a hardcoded scheduling wait that is inside the CUPS software.

If you get interesting results, I'd love to hear about them.
