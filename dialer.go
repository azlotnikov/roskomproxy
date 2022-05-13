package main

import (
	"bytes"
	"context"
	"github.com/9seconds/httransform/v2/cache"
	"github.com/9seconds/httransform/v2/dialers"
	"github.com/9seconds/httransform/v2/errors"
	"github.com/libp2p/go-reuseport"
	utls "github.com/refraction-networking/utls"
	"github.com/valyala/fasthttp"
	"net"
	"sync"
	"time"
)

const (
	TLSConfigCacheSize = 512
	TLSConfigTTL       = 10 * time.Minute
)

type utlsDialer struct {
	netDialer      net.Dialer
	resolver       *net.Resolver
	tlsConfigsLock sync.Mutex
	tlsConfigs     cache.Interface
	tlsSkipVerify  bool
}

func (d *utlsDialer) Dial(ctx context.Context, host, port string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, d.netDialer.Timeout)
	defer cancel()

	ips, err := d.resolver.LookupHost(ctx, host)
	if err != nil {
		return nil, errors.Annotate(err, "cannot resolve IPs", "dns_no_ips", 0)
	}

	if len(ips) == 0 {
		return nil, dialers.ErrNoIPs
	}

	var conn net.Conn

	for _, ip := range ips {
		conn, err = d.netDialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
		if err == nil {
			return conn, nil
		}
	}

	return nil, errors.Annotate(err, "cannot dial to "+host, "cannot_dial", 0)
}

func (d *utlsDialer) UpgradeToTLS(ctx context.Context, conn net.Conn, host, _ string) (net.Conn, error) {
	subCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		timer := fasthttp.AcquireTimer(d.netDialer.Timeout)
		defer fasthttp.ReleaseTimer(timer)

		select {
		case <-subCtx.Done():
		case <-ctx.Done():
			select {
			case <-subCtx.Done():
			default:
				_ = conn.Close()
			}
		case <-timer.C:
			select {
			case <-subCtx.Done():
			default:
				_ = conn.Close()
			}
		}
	}()
	tlsConn := utls.UClient(conn, d.getTLSConfig(host), utls.HelloCustom)

	if err := tlsConn.ApplyPreset(getSpec()); err != nil {
		return nil, errors.Annotate(err, "cannot set TLS Hello spec", "tls_hello_spec", 0)
	}
	if err := tlsConn.Handshake(); err != nil {
		return nil, errors.Annotate(err, "cannot perform TLS handshake", "tls_handshake", 0)
	}

	return tlsConn, nil
}

func (d *utlsDialer) PatchHTTPRequest(req *fasthttp.Request) {
	if bytes.EqualFold(req.URI().Scheme(), []byte("http")) {
		req.SetRequestURIBytes(req.URI().PathOriginal())
	}
}

func (d *utlsDialer) getTLSConfig(host string) *utls.Config {
	if conf := d.tlsConfigs.Get(host); conf != nil {
		return conf.(*utls.Config)
	}

	d.tlsConfigsLock.Lock()
	defer d.tlsConfigsLock.Unlock()

	if conf := d.tlsConfigs.Get(host); conf != nil {
		return conf.(*utls.Config)
	}

	conf := &utls.Config{
		ClientSessionCache: utls.NewLRUClientSessionCache(0),
		InsecureSkipVerify: d.tlsSkipVerify, // nolint: gosec
	}

	d.tlsConfigs.Add(host, conf)

	return conf
}

func NewUTLSDialer(opt dialers.Opts, dns string) dialers.Dialer {
	rv := &utlsDialer{
		netDialer: net.Dialer{
			Timeout: opt.GetTimeout(),
			Control: reuseport.Control,
		},
		tlsConfigs: cache.New(TLSConfigCacheSize,
			TLSConfigTTL,
			cache.NoopEvictCallback),
		tlsSkipVerify: opt.GetTLSSkipVerify(),
		resolver:      net.DefaultResolver,
	}
	if dns != "" {
		rv.resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Millisecond * time.Duration(10000),
				}
				return d.DialContext(ctx, network, dns+":53")
			},
		}
	}

	return rv
}
