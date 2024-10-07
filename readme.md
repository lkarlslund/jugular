# jugular

Use jugs to look for CUPS: an ultrafast UDP sender, that tries to tickle listening cups-browsed processes to do a HTTP callback.

This is a high performance scanner that writes UDP packets directly to the network.

## Install (Ubuntu or similar)

```
sudo apt install golang libpcap0.8-dev
go install github.com/lkarlslund/jugular@main
```

... or download the Linux x64 binary from the builds

## Usage

1. Grant jugular needed network permissions

Allow jugular to bind to low ports and access raw sockets without being run as root (you only need to do this once):

```sudo setcap CAP_NET_BIND_SERVICE,CAP_NET_RAW=eip jugular```

2. Start jugular listening server, it's a simple HTTP server that replies with 404 to all requests, but saves the request headers in json files in current working directory - you can use something different if you want

```jugular listen --bind 0.0.0.0:80```

3. Start prodding CUPS instances, by sending UDP packets that announce a printer. Note: scanning the internet is in the grey area of legality, so be careful. If you're company IT security, you can use it internally ofcourse.

Run jugular and do the scan:

```jugular prod --network 10.0.0.0/8 --delay 1 --url http://yourdomain.net/printer/scanning-you --addip true --ip your-own-machine-ip```

Since this software creates and writes its own packets directly to an outgoing network for performance reasons, you *need* to assign your own correct machine IP address (run `ip addr` to find it). The remote instances need to be able to route back to the URL you put in that parameter as well.

Test locally before going on a hunt, and please remember that this software can overwhelm your NAT firewall and other equipment (use the delay option).

*PLEASE NOTE* There is an up to 300 second delay from CUPS recieving the message to it reaches out to the URL you gave it. This is due to a hardcoded scheduling wait that is inside the CUPS software. You may also trigger endless notifications, as some CUPS servers do not stop trying to connect back to the URL even after they have received an error response.

## SUGGESTED FIXES 
Choose one or more ... (for Ubuntu, YMMV with others - I updated + enabled ufw)

### On a recent distro (Ubuntu and others), just update CUPS, it'll reconfigure it for you!
- sudo apt update
- sudo apt upgrade (package "cups" should be listed among those that will be updated)

### Keep CUPS but enable firewalling. This disables unsolicited remote access to CUPS (be careful if you're remote):
- sudo ufw allow ssh (if you need that, could also be http, https etc)
- sudo ufw enable (enable firewall)

### You can remove CUPS if you're not printing:
- sudo apt remove cups

### Keep CUPS but stop and disable it:
- sudo systemctl disable --now cups

*If you get interesting results, I'd love to hear about them.*

### Other implementations:
- https://github.com/MalwareTech/CVE-2024-47176-Scanner by MalwareTech
