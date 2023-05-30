package plc

import (
	"context"
	"errors"
	"sync"
	"time"

	plc4go "github.com/apache/plc4x/plc4go/pkg/api"
	"github.com/apache/plc4x/plc4go/pkg/api/model"
)

var (
	ErrPoolConnFailed    = errors.New("conn_pool: connection failed")
	ErrPoolClosed        = errors.New("conn_pool: pool is closed")
	ErrConnTimeout       = errors.New("plc_conn: connection timeout")
	ErrConnClosed        = errors.New("plc_conn: connection is closed")
	ErrConnAlreadyClosed = errors.New("plc_conn: connection already closed")
	ErrConnWriteOnly     = errors.New("plc_conn: can't read, write only connection")
	ErrConnReadOnly      = errors.New("plc_conn: can't write, read only connection")
)

type ConnPool struct {
	plcURI       string
	plcDriver    plc4go.PlcDriverManager
	connTimeout  time.Duration
	maxOpen      int
	connRequests map[chan connRequest]struct{}
	numOpen      int
	mu           *sync.Mutex
	closed       bool
}

func (c *ConnPool) SetMaxOpenConns(n int) {
	c.mu.Lock()
	c.maxOpen = n
	if n < 0 {
		// Unlimited connections
		c.maxOpen = 0
	}
	c.mu.Unlock()
}

func (c *ConnPool) SetConnTimeout(t time.Duration) {
	c.mu.Lock()
	c.connTimeout = t
	c.mu.Unlock()
}

func (c *ConnPool) newConn() (plc4go.PlcConnection, error) {
	plcConnChan := c.plcDriver.GetConnection(c.plcURI)
	plcConnResult, err := resultWithTimeout(plcConnChan, c.connTimeout)
	if err != nil {
		return nil, err
	}

	return plcConnResult.GetConnection(), nil
}

func (c *ConnPool) conn(ctx context.Context) (*driverConn, error) {
	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrPoolClosed
	}

	if c.maxOpen > 0 && c.numOpen >= c.maxOpen {
		req := make(chan connRequest, 1)
		c.connRequests[req] = struct{}{}
		c.mu.Unlock()

		ret, ok := <-req
		if !ok {
			return nil, ErrPoolClosed
		}
		if ret.conn == nil {
			return nil, ret.err
		}

		return ret.conn, ret.err
	}

	c.mu.Unlock()
	plcConn, err := c.newConn()
	if err != nil {
		return nil, err
	}
	c.mu.Lock()

	c.numOpen++
	drvConn := newDriveConn(c, plcConn)
	c.mu.Unlock()
	return drvConn, nil
}

func (c *ConnPool) releaseConn(drvConn *driverConn) error {
	err := drvConn.closeConn()
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.numOpen--
	if c.maxOpen > 0 && c.numOpen > c.maxOpen {
		c.mu.Unlock()
		return nil
	}
	if l := len(c.connRequests); l > 0 {
		var req chan connRequest
		for req = range c.connRequests {
			break
		}
		delete(c.connRequests, req)
		c.numOpen++

		c.mu.Unlock()
		plcConn, err := c.newConn()
		c.mu.Lock()

		req <- connRequest{
			conn: newDriveConn(c, plcConn),
			err:  err,
		}
	}
	c.mu.Unlock()
	return nil
}

func (c *ConnPool) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}

	c.closed = true
	for req := range c.connRequests {
		close(req)
	}

	c.mu.Unlock()
	return nil
}

func (c *ConnPool) ReadTagAddress(ctx context.Context, tagName string, tagAddress string) (model.PlcReadResponse, error) {
	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	conn, err := c.conn(ctx)
	if err != nil {
		return nil, err
	}

	response, err := conn.readTagAddress(ctx, tagName, tagAddress)
	if err != nil {
		rlErr := c.releaseConn(conn)
		if rlErr != nil {
			return nil, errors.Join(err, rlErr)
		}
		return nil, err
	}

	err = c.releaseConn(conn)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *ConnPool) WriteTagAddress(ctx context.Context, tagName string, tagAddress string, value any) (model.PlcWriteResponse, error) {
	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	conn, err := c.conn(ctx)
	if err != nil {
		return nil, err
	}
	response, err := conn.writeTagAddress(ctx, tagName, tagAddress, value)
	if err != nil {
		rlErr := c.releaseConn(conn)
		if rlErr != nil {
			return nil, errors.Join(err, rlErr)
		}
		return nil, err
	}

	err = c.releaseConn(conn)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *ConnPool) Ping(ctx context.Context) error {
	select {
	default:
	case <-ctx.Done():
		return ctx.Err()
	}

	conn, err := c.conn(ctx)
	if err != nil {
		return err
	}
	err = conn.ping(ctx)
	if err != nil {
		rlErr := c.releaseConn(conn)
		if rlErr != nil {
			return errors.Join(err, rlErr)
		}
		return err
	}

	err = c.releaseConn(conn)
	if err != nil {
		return err
	}
	return nil
}

type responseWithErr interface {
	GetErr() error
}

type driverConn struct {
	connPool *ConnPool
	conn     plc4go.PlcConnection
	closed   bool
	mu       sync.Mutex
}

func (dc *driverConn) readTagAddress(ctx context.Context, tagName string, tagAddress string) (model.PlcReadResponse, error) {
	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	dc.mu.Lock()
	if dc.closed {
		dc.mu.Unlock()
		return nil, ErrConnClosed
	}

	if !dc.conn.GetMetadata().CanRead() {
		dc.mu.Unlock()
		return nil, ErrConnWriteOnly
	}
	req, err := dc.conn.ReadRequestBuilder().
		AddTagAddress(tagName, tagAddress).
		Build()
	if err != nil {
		dc.mu.Unlock()
		return nil, err
	}
	respChan := req.ExecuteWithContext(ctx)
	reqResult, err := resultWithTimeout(respChan, dc.connPool.connTimeout)
	if err != nil {
		dc.mu.Unlock()
		return nil, err
	}

	dc.mu.Unlock()
	return reqResult.GetResponse(), nil
}

func (dc *driverConn) writeTagAddress(ctx context.Context, tagName string, tagAddress string, value any) (model.PlcWriteResponse, error) {
	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	dc.mu.Lock()
	if dc.closed {
		dc.mu.Unlock()
		return nil, ErrConnClosed
	}

	if !dc.conn.GetMetadata().CanWrite() {
		dc.mu.Unlock()
		return nil, ErrConnReadOnly
	}
	req, err := dc.conn.WriteRequestBuilder().
		AddTagAddress(tagName, tagAddress, value).
		Build()
	if err != nil {
		dc.mu.Unlock()
		return nil, err
	}
	respChan := req.ExecuteWithContext(ctx)
	reqResult, err := resultWithTimeout(respChan, dc.connPool.connTimeout)
	if err != nil {
		dc.mu.Unlock()
		return nil, err
	}

	dc.mu.Unlock()
	return reqResult.GetResponse(), nil
}

func (dc *driverConn) ping(ctx context.Context) error {
	select {
	default:
	case <-ctx.Done():
		return ctx.Err()
	}
	dc.mu.Lock()
	if dc.closed {
		dc.mu.Unlock()
		return ErrConnClosed
	}

	respChan := dc.conn.Ping()
	_, err := resultWithTimeout(respChan, dc.connPool.connTimeout)
	if err != nil {
		dc.mu.Unlock()
		return err
	}

	dc.mu.Unlock()
	return nil
}

func (dc *driverConn) closeConn() error {
	dc.mu.Lock()
	if dc.closed {
		dc.mu.Unlock()
		return ErrConnAlreadyClosed
	}
	dc.closed = true

	closeChan := dc.conn.Close()
	_, err := resultWithTimeout(closeChan, dc.connPool.connTimeout)
	if err != nil {
		dc.mu.Unlock()
		return err
	}

	dc.mu.Unlock()
	return nil
}

func resultWithTimeout[T responseWithErr](respChan <-chan T, timeout time.Duration) (T, error) {
	var empty T
	if timeout != 0 {
		toutCtx, toutCancelCtx := context.WithTimeout(context.Background(), timeout)
		defer toutCancelCtx()
		for {
			select {
			case reqResult := <-respChan:
				if reqResult.GetErr() != nil {
					return empty, reqResult.GetErr()
				}
				return reqResult, nil
			case <-toutCtx.Done():
				return empty, ErrConnTimeout
			}
		}
	}

	reqResult := <-respChan
	if reqResult.GetErr() != nil {
		return empty, reqResult.GetErr()
	}
	return reqResult, nil
}

func newDriveConn(connPool *ConnPool, conn plc4go.PlcConnection) *driverConn {
	return &driverConn{
		connPool: connPool,
		conn:     conn,
	}
}

type connRequest struct {
	conn *driverConn
	err  error
}

func NewConnPool(driver plc4go.PlcDriverManager, plcURI string) (*ConnPool, error) {
	connPool := &ConnPool{
		plcURI:       plcURI,
		plcDriver:    driver,
		connTimeout:  5 * time.Second,
		connRequests: make(map[chan connRequest]struct{}),
		mu:           &sync.Mutex{},
	}

	err := connPool.Ping(context.Background())
	if err != nil {
		return nil, errors.Join(ErrPoolConnFailed, err)
	}

	return connPool, nil
}
