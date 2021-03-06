package config

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/juju/errors"
)

// Buffer sizes define internal socket buffer sizes.
const (
	BufferWriteSize = 32 * 1024
	BufferReadSize  = 32 * 1024
)

// Config represents common configuration of mtg.
type Config struct {
	Debug   bool
	Verbose bool

	BindPort       uint16
	PublicIPv4Port uint16
	PublicIPv6Port uint16
	StatsPort      uint16

	BindIP     net.IP
	PublicIPv4 net.IP
	PublicIPv6 net.IP
	StatsIP    net.IP

	Secret []byte
	AdTag  []byte
}

// URLs contains links to the proxy (tg://, t.me) and their QR codes.
type URLs struct {
	TG        string `json:"tg_url"`
	TMe       string `json:"tme_url"`
	TGQRCode  string `json:"tg_qrcode"`
	TMeQRCode string `json:"tme_qrcode"`
}

// IPURLs contains links to both ipv4 and ipv6 of the proxy.
type IPURLs struct {
	IPv4 URLs `json:"ipv4"`
	IPv6 URLs `json:"ipv6"`
}

// BindAddr returns connection for this server to bind to.
func (c *Config) BindAddr() string {
	return getAddr(c.BindIP, c.BindPort)
}

// StatAddr returns connection string to the stats API.
func (c *Config) StatAddr() string {
	return getAddr(c.StatsIP, c.StatsPort)
}

// UseMiddleProxy defines if this proxy has to connect middle proxies
// which supports promoted channels or directly access Telegram.
func (c *Config) UseMiddleProxy() bool {
	return len(c.AdTag) > 0
}

// GetURLs returns configured IPURLs instance with links to this server.
func (c *Config) GetURLs() IPURLs {
	urls := IPURLs{}
	if c.PublicIPv4 != nil {
		urls.IPv4 = getURLs(c.PublicIPv4, c.PublicIPv4Port, c.Secret)
	}
	if c.PublicIPv6 != nil {
		urls.IPv6 = getURLs(c.PublicIPv6, c.PublicIPv6Port, c.Secret)
	}

	return urls
}

func getAddr(host fmt.Stringer, port uint16) string {
	return net.JoinHostPort(host.String(), strconv.Itoa(int(port)))
}

// NewConfig returns new configuration. If required, it manages and
// fetches data from external sources. Parameters passed to this
// function, should come from command line arguments.
func NewConfig(debug, verbose bool, // nolint: gocyclo
	bindIP net.IP, bindPort uint16,
	publicIPv4 net.IP, PublicIPv4Port uint16,
	publicIPv6 net.IP, publicIPv6Port uint16,
	statsIP net.IP, statsPort uint16,
	secret, adtag string) (*Config, error) {
	secret = strings.TrimPrefix(secret, "dd")
	if len(secret) != 32 {
		return nil, errors.New("Telegram demands secret of length 32")
	}
	secretBytes, err := hex.DecodeString(secret)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create config")
	}

	var adTagBytes []byte
	if len(adtag) != 0 {
		adTagBytes, err = hex.DecodeString(adtag)
		if err != nil {
			return nil, errors.Annotate(err, "Cannot create config")
		}
	}

	if publicIPv4 == nil {
		publicIPv4, err = getGlobalIPv4()
		if err != nil {
			publicIPv4 = nil
		} else if publicIPv4.To4() == nil {
			return nil, errors.Errorf("IP %s is not IPv4", publicIPv4.String())
		}
	}
	if PublicIPv4Port == 0 {
		PublicIPv4Port = bindPort
	}

	if publicIPv6 == nil {
		publicIPv6, err = getGlobalIPv6()
		if err != nil {
			publicIPv6 = nil
		} else if publicIPv6.To4() != nil {
			return nil, errors.Errorf("IP %s is not IPv6", publicIPv6.String())
		}
	}
	if publicIPv6Port == 0 {
		publicIPv6Port = bindPort
	}

	if statsIP == nil {
		statsIP = publicIPv4
	}

	conf := &Config{
		Debug:          debug,
		Verbose:        verbose,
		BindIP:         bindIP,
		BindPort:       bindPort,
		PublicIPv4:     publicIPv4,
		PublicIPv4Port: PublicIPv4Port,
		PublicIPv6:     publicIPv6,
		PublicIPv6Port: publicIPv6Port,
		StatsIP:        statsIP,
		StatsPort:      statsPort,
		Secret:         secretBytes,
		AdTag:          adTagBytes,
	}

	return conf, nil
}
