package tcp

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kokaq/protocol/tcp"
)

type ServerConfig struct {
	Port           int    `json:"port"`
	Timeout        int    `json:"timeout"`         // Timeout in seconds for each request
	UseMTls        bool   `json:"use_mtls"`        // Whether to use Mutual TLS for secure connections
	CertFile       string `json:"cert_file"`       // Path to the TLS certificate file
	KeyFile        string `json:"key_file"`        // Path to the TLS key file
	MaxConnections int    `json:"max_connections"` // Maximum number of concurrent connections
	// Add more configuration options as needed
}

type Server struct {
	port               int
	mux                *tcp.Multiplexer
	middlewareRegistry *tcp.MiddlewareRegistry
	listener           net.Listener
	wg                 sync.WaitGroup
	idleTimeout        int
	useMTls            bool
	certFile           string
	keyFile            string
	logger             Logger
	maxConnections     int
}

func NewServer(config ServerConfig) *Server {
	server := &Server{
		port:               config.Port,
		idleTimeout:        config.Timeout,
		useMTls:            config.UseMTls,
		certFile:           config.CertFile,
		keyFile:            config.KeyFile,
		maxConnections:     config.MaxConnections,
		mux:                tcp.NewMultiplexer(),
		middlewareRegistry: tcp.NewMiddlewareRegistry(),
	}
	return server
}

func (server *Server) RegisterHandler(opcode uint8, handler tcp.Handler) {
	server.mux.RegisterHandler(opcode, handler)
}

func (server *Server) RegisterMiddleware(middleware tcp.Middleware) {
	server.middlewareRegistry.RegisterMiddleware(middleware)
}

func (server *Server) WithLogger(logger *Logger) {
	server.logger = *logger
}

func (server *Server) Start() error {
	var err error
	address := fmt.Sprintf(":%d", server.port)
	if server.useMTls {
		server.logger.Infof("Starting server with TLS enabled.")
		var cer tls.Certificate
		cer, err = tls.LoadX509KeyPair(server.certFile, server.keyFile)
		if err != nil {
			server.logger.Errorf("failed to load TLS certs", err)
			return err
		}
		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		server.listener, err = tls.Listen("tcp", address, config)
	} else {
		server.logger.Infof("Starting server without TLS.")
		if server.useMTls {
			server.logger.Warnf("warning: TLS is enabled but no certs provided, falling back to plain TCP.")
		}
		server.listener, err = net.Listen("tcp", address)
	}
	if err != nil {
		server.logger.Errorf("Error starting server ", err)
		return err
	}
	defer server.listener.Close()

	server.logger.Infof("Listening on port ", server.port)
	var connCount int32 = 0
	for {
		server.logger.Infof("Waiting for a connection...")
		client, err := server.listener.Accept()
		if err != nil {
			server.logger.Warnf("Failed to accept connection")
			continue
		}
		server.logger.Infof("Received a request...")
		if atomic.AddInt32(&connCount, 1) > int32(server.maxConnections) {
			server.logger.Warnf("Too many connections — rejecting client")
			client.Close()
			atomic.AddInt32(&connCount, -1)
			continue
		}

		server.wg.Add(1)
		go func(conn net.Conn) {

			defer server.wg.Done()
			defer conn.Close()

			tlsConn, ok := conn.(*tls.Conn)
			if !ok {
				server.logger.Warnf("Not a TLS connection")
				return
			}
			if err := tlsConn.Handshake(); err != nil {
				server.logger.Warnf("TLS handshake failed")
				return
			}
			server.logger.Infof("Accepted request")
			conn.SetDeadline(time.Now().Add(time.Second * time.Duration(server.idleTimeout)))

			// Deliberately not reading entire request from stream
			// This is to reduce connection time of invalid TCP request
			// Avoid reading stream
			var commonHeader *tcp.CommonHeader
			if commonHeader, err = tcp.CommonHeaderFromStream(conn); err != nil {
				server.logger.Error("Cannot read common header")
				return
			}
			if commonHeader.Magic != tcp.Magic {
				server.logger.Error("Bad request: invalid Magic")
				return
			}

			var requestHeader *tcp.RequestHeader
			if requestHeader, err = tcp.RequestHeaderFromStream(conn); err != nil {
				server.logger.Error("Cannot read operational header")
				return
			}

			defer atomic.AddInt32(&connCount, -1)

			// Invoke Handler
			var request *tcp.Request
			request, err = tcp.NewRequest(*commonHeader, *requestHeader)

			server.mux.Mu.RLock()

			var handler tcp.Handler
			var handlerExists bool
			if handler, handlerExists = server.mux.Routes[requestHeader.Opcode]; !handlerExists {
				server.logger.Warnf("Unknown opcode: %d", requestHeader.Opcode)
			}
			server.mux.Mu.RUnlock()

			if err != nil {
				server.logger.Error("Bad request: cannot read payload")
				return
			}

			server.middlewareRegistry.Mu.RLock()
			for i, mdlwr := range server.middlewareRegistry.Middlewares {
				server.logger.Infof(fmt.Sprintf("Running middleware #%d", i))
				mdlwr.Handle(conn, request)
			}
			server.middlewareRegistry.Mu.RUnlock()

			var response *tcp.Response
			response, err = handler.Handle(request)
			response.ToStream(conn)
			server.logger.Infof("Done handling the request.")

		}(client)
	}
}
