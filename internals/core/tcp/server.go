package tcp

// import (
// 	"crypto/tls"
// 	"fmt"
// 	"net"
// 	"os"
// 	"sync"
// 	"sync/atomic"
// 	"time"

// 	kokaq_tcp "github.com/kokaq/protocol/core/tcp"
// 	"github.com/kokaq/server/internals/prometheus"
// 	"github.com/kokaq/server/internals/trace"
// 	"github.com/sirupsen/logrus"
// 	sdktrace "go.opentelemetry.io/otel/sdk/trace"
// )

// type KokaqWireServerConfig struct {
// 	Port               int    `json:"port"`
// 	Timeout            int    `json:"timeout"`               // Timeout in seconds for each request
// 	UseTls             bool   `json:"use_tls"`               // Whether to use TLS for secure connections
// 	CertFile           string `json:"cert_file"`             // Path to the TLS certificate file
// 	KeyFile            string `json:"key_file"`              // Path to the TLS key file
// 	AuthMode           string `json:"auth_mode"`             // Authentication mode, e.g., "none", "basic", "token"
// 	MetricsAddress     string `json:"metrics_address"`       // Address for Prometheus metrics
// 	UsePrometheus      bool   `json:"use_prometheus"`        // Whether to enable Prometheus metrics
// 	MaxConnections     int    `json:"max_connections"`       // Maximum number of concurrent connections
// 	UseAuditLogger     bool   `json:"use_audit_logger"`      // Whether to use an audit logger
// 	UseFileAuditLogger bool   `json:"use_file_audit_logger"` // Whether to use a file-based audit logger
// 	UseTrace           bool   `json:"use_trace"`             // Whether to enable tracing
// 	// Add more configuration options as needed
// }

// type KokaqWireServer struct {
// 	port               int
// 	listener           net.Listener
// 	tracer             *sdktrace.TracerProvider
// 	wg                 sync.WaitGroup
// 	idleTimeout        int
// 	authMode           string
// 	useTls             bool
// 	certFile           string
// 	keyFile            string
// 	metricsAddress     string
// 	metrics            *prometheus.PrometheusMetrics
// 	usePrometheus      bool
// 	promClient         *prometheus.PrometheusClient
// 	logger             *logrus.Logger
// 	auditLogger        *logrus.Logger
// 	useAuditLogger     bool // Whether to use an audit logger
// 	useFileAuditLogger bool // Whether to use a file-based audit logger
// 	useTrace           bool // Whether to enable tracing
// 	maxConnections     int  // Maximum number of concurrent connections
// }

// type IKokaqServerhandler interface {
// 	HandleRequest(*kokaq_tcp.KokaqWireRequest) (*kokaq_tcp.KokaqWireResponse, error)
// }

// // NewKokaqWireServer initializes a new KokaqWireServer with the provided port.
// func NewKokaqWireServer(config KokaqWireServerConfig) *KokaqWireServer {
// 	server := &KokaqWireServer{
// 		port:               config.Port,
// 		idleTimeout:        config.Timeout,
// 		authMode:           config.AuthMode,
// 		useTls:             config.UseTls,
// 		certFile:           config.CertFile,
// 		keyFile:            config.KeyFile,
// 		logger:             logrus.New(),
// 		useTrace:           config.UseTrace,
// 		useAuditLogger:     config.UseAuditLogger,
// 		useFileAuditLogger: config.UseAuditLogger,
// 		auditLogger:        logrus.New(),
// 		maxConnections:     config.MaxConnections,
// 		usePrometheus:      config.UsePrometheus,
// 	}

// 	if config.UseTrace {
// 		var err error
// 		server.logger.Info("Tracing enabled")
// 		// Initialize OpenTelemetry tracing
// 		var tracerClose func()
// 		server.tracer, err, tracerClose = trace.SetupTracer()
// 		if err != nil {
// 			server.logger.WithError(err).Warn("error starting open-telemetry tracing")
// 		}
// 		defer tracerClose()
// 	}

// 	if config.UseAuditLogger {
// 		server.logger.Info("Audit logging enabled")
// 		server.auditLogger.SetFormatter(&logrus.JSONFormatter{
// 			TimestampFormat: time.RFC3339,
// 		})
// 		server.auditLogger.SetLevel(logrus.InfoLevel)
// 		if config.UseFileAuditLogger {
// 			server.logger.Info("File audit logging enabled")
// 			file, err := os.OpenFile("./audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
// 			if err != nil {
// 				server.logger.WithError(err).Error("Failed to open audit log file")
// 				server.auditLogger.SetOutput(os.Stdout)
// 			} else {
// 				server.auditLogger.SetOutput(file)
// 			}
// 		} else {
// 			server.logger.Info("Using stdout for audit logging")
// 		}
// 	}

// 	if config.UsePrometheus {
// 		server.logger.Info("Prometheus metrics enabled")
// 		server.metricsAddress = config.MetricsAddress
// 		server.metrics = prometheus.NewPrometheusMetrics()
// 		server.promClient = prometheus.NewPrometheusClient(server.logger, server.metricsAddress, server.metrics)
// 	}

// 	return server
// }

// // Start begins the TCP listener and handles incoming client requests.
// func (server *KokaqWireServer) Start(handler IKokaqServerhandler) error {

// 	var err error

// 	// Start TCP server
// 	server.listener, err = server.StartInternal()
// 	defer server.listener.Close()
// 	if err != nil {
// 		server.logger.Error("Error starting server ", err)
// 		return err
// 	}

// 	var connCount int32 = 0

// 	server.logger.Info("Listening on port ", server.port)
// 	for {
// 		server.logger.Info("Waiting for a connection...")
// 		client, err := server.listener.Accept()
// 		if err != nil {
// 			server.logger.WithError(err).Warn("Failed to accept connection")
// 			continue
// 		}
// 		server.logger.Info("Receiving a request...")

// 		// Check if the maximum number of connections is reached

// 		if atomic.AddInt32(&connCount, 1) > int32(server.maxConnections) {
// 			server.logger.Warn("Too many connections — rejecting client")
// 			client.Close()
// 			atomic.AddInt32(&connCount, -1)
// 			continue
// 		}

// 		// Handle each client in a separate goroutine
// 		server.wg.Add(1)
// 		go func(client net.Conn) {
// 			server.logger.Info("Received a request")
// 			defer server.wg.Done()
// 			defer client.Close()
// 			// Increment the active connections gauge
// 			server.metrics.ActiveConnections.Inc()
// 			defer server.metrics.ActiveConnections.Dec()

// 			tlsConn, ok := client.(*tls.Conn)
// 			if !ok {
// 				server.logger.Warn("Not a TLS connection")
// 				return
// 			}

// 			if err := tlsConn.Handshake(); err != nil {
// 				server.logger.WithError(err).Warn("TLS handshake failed")
// 				server.auditLogger.WithFields(logrus.Fields{
// 					"time":      time.Now().Format(time.RFC3339),
// 					"event":     "tls_handshake_failed",
// 					"error":     err.Error(),
// 					"remote_ip": client.RemoteAddr().String(),
// 				}).Warn("TLS handshake failed")
// 				return
// 			}

// 			// state := tlsConn.ConnectionState()

// 			// if len(state.PeerCertificates) == 0 {
// 			// 	server.logger.Warn("Client cert not provided")
// 			// 	return
// 			// }
// 			// Increment the request counter
// 			server.metrics.RequestCounter.Inc()
// 			server.logger.WithField("remote_addr", client.RemoteAddr().String()).Info("Handling request from client")

// 			// Set a deadline for the client connection to avoid hanging indefinitely
// 			client.SetDeadline(time.Now().Add(time.Second * time.Duration(server.idleTimeout)))

// 			var req = &kokaq_tcp.KokaqWireRequest{}

// 			err := req.ReadFromStream(client)
// 			if err != nil {
// 				server.logger.Error("Error reading request", err)
// 				return
// 			}

// 			switch req.MessageType {
// 			case kokaq_tcp.MessageTypeOperational:
// 				server.logger.Info("Received an operational message.")
// 			case kokaq_tcp.MessageTypeControl:
// 				server.logger.Info("Received a control message.")
// 			case kokaq_tcp.MessageTypeAdmin:
// 				server.logger.Info("Received a adminops message.")
// 			default:
// 				server.logger.Info("Received an unknown message type.")
// 				return
// 			}

// 			switch req.OpCode {
// 			case kokaq_tcp.OpCodeCreate:
// 				server.logger.Info("Received a create request.")
// 			case kokaq_tcp.OpCodeDelete:
// 				server.logger.Info("Received a delete request.")
// 			case kokaq_tcp.OpCodeGet:
// 				server.logger.Info("Received a get request.")
// 			case kokaq_tcp.OpCodePeek:
// 				server.logger.Info("Received a peek request.")
// 			case kokaq_tcp.OpCodePop:
// 				server.logger.Info("Received a pop request.")
// 			case kokaq_tcp.OpCodePush:
// 				server.logger.Info("Received a push request.")
// 			case kokaq_tcp.OpCodeAcquirePeekLock:
// 				server.logger.Info("Received an acquire peek lock request.")
// 			default:
// 				server.logger.Info("Received an unknown operation.")
// 				return
// 			}

// 			defer atomic.AddInt32(&connCount, -1)
// 			res, err := handler.HandleRequest(req)
// 			err = res.WriteToStream(client)
// 			if err != nil {
// 				server.logger.Error("Error writing response", err)
// 				return
// 			}
// 			server.logger.Info("Done handling the request.")
// 		}(client)
// 	}
// }

// func (server *KokaqWireServer) StartInternal() (net.Listener, error) {

// 	var err error
// 	address := fmt.Sprintf(":%d", server.port)
// 	var listner net.Listener
// 	if server.useTls {
// 		server.logger.Info("Starting server with TLS enabled.")
// 		var cer tls.Certificate
// 		cer, err = tls.LoadX509KeyPair(server.certFile, server.keyFile)
// 		if err != nil {
// 			server.logger.Error("failed to load TLS certs", err)
// 			return nil, err
// 		}
// 		config := &tls.Config{Certificates: []tls.Certificate{cer}}
// 		listner, err = tls.Listen("tcp", address, config)
// 	} else {
// 		server.logger.Info("Starting server without TLS.")
// 		if server.useTls {
// 			server.logger.Warn("Warning: TLS is enabled but no certs provided, falling back to plain TCP.")
// 		}
// 		listner, err = net.Listen("tcp", address)
// 	}
// 	if err != nil {
// 		server.logger.Error("Error starting server ", err)
// 		return nil, err
// 	}
// 	server.logger.WithField("addr", address).Info("TCP server started")

// 	return listner, nil
// }

// // Stop stops the server and waits for all connections to be handled.
// func (server *KokaqWireServer) Stop() {
// 	server.listener.Close()
// 	server.wg.Wait()
// }
