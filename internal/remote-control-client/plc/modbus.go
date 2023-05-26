package plc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	plc4go "github.com/apache/plc4x/plc4go/pkg/api"
	"github.com/apache/plc4x/plc4go/pkg/api/model"
)

var (
	ErrPoolClosed    = errors.New("conn_pool: pool is closed")
	ErrClosing       = errors.New("plc_conn: failed to close connection")
	ErrConnWriteOnly = errors.New("plc_conn: can't read, write only connection")
	ErrConnReadOnly  = errors.New("plc_conn: can't write, read only connection")
)

type ConnPool struct {
	plcURI       string
	plcDriver    plc4go.PlcDriverManager
	maxOpen      int
	connRequests map[uint64]chan connRequest
	nextRequest  uint64
	numOpen      int
	mu           sync.Mutex
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

func (c *ConnPool) newConn() (plc4go.PlcConnection, error) {
	plcConnChan := c.plcDriver.GetConnection(c.plcURI)
	plcConnResult := <-plcConnChan
	if plcConnResult.GetErr() != nil {
		return nil, plcConnResult.GetErr()
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
		reqKey := c.nextRequest
		c.nextRequest++
		c.connRequests[reqKey] = req
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
	c.numOpen++
	c.mu.Unlock()
	plcConn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	drvConn := &driverConn{
		connPool: c,
		conn:     plcConn,
	}
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
		var reqKey uint64
		for reqKey, req = range c.connRequests {
			break
		}
		delete(c.connRequests, reqKey)

		c.mu.Unlock()
		plcConn, err := c.newConn()
		c.mu.Lock()

		req <- connRequest{
			conn: &driverConn{
				connPool: c,
				conn:     plcConn,
			},
			err: err,
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

	for _, req := range c.connRequests {
		close(req)
	}
	c.mu.Unlock()
	return nil
}

func (c *ConnPool) readTagAddress(ctx context.Context, conn *driverConn, tagName string, tagAddress string) (model.PlcReadResponse, error) {
	if !conn.conn.GetMetadata().CanRead() {
		return nil, ErrConnWriteOnly
	}
	req, err := conn.conn.ReadRequestBuilder().
		AddTagAddress(tagName, tagAddress).
		Build()
	if err != nil {
		return nil, err
	}
	respChan := req.ExecuteWithContext(ctx)
	reqResult := <-respChan
	if reqResult.GetErr() != nil {
		return nil, err
	}
	return reqResult.GetResponse(), nil
}

func (c *ConnPool) ReadTagAddress(ctx context.Context, tagName string, tagAddress string) (model.PlcReadResponse, error) {
	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// DEBUG
	fmt.Printf("start conn %s, pool=%d, queue=%d \n", tagName, c.numOpen, len(c.connRequests))
	// DEBUG

	conn, err := c.conn(ctx)
	if err != nil {
		return nil, err
	}
	response, err := c.readTagAddress(ctx, conn, tagName, tagAddress)
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

	// DEBUG
	fmt.Printf("stop conn %s, pool=%d, queue=%d \n", tagName, c.numOpen, len(c.connRequests))
	// DEBUG

	return response, nil
}

func (c *ConnPool) writeTagAddress(ctx context.Context, conn *driverConn, tagName string, tagAddress string, value any) (model.PlcWriteResponse, error) {
	if !conn.conn.GetMetadata().CanWrite() {
		return nil, ErrConnReadOnly
	}
	req, err := conn.conn.WriteRequestBuilder().
		AddTagAddress(tagName, tagAddress, value).
		Build()
	if err != nil {
		return nil, err
	}
	respChan := req.ExecuteWithContext(ctx)
	reqResult := <-respChan
	if reqResult.GetErr() != nil {
		return nil, err
	}
	return reqResult.GetResponse(), nil
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
	response, err := c.writeTagAddress(ctx, conn, tagName, tagAddress, value)
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

type driverConn struct {
	connPool *ConnPool
	conn     plc4go.PlcConnection
	closed   bool
	mu       sync.Mutex
}

func (dc *driverConn) closeConn() error {
	dc.mu.Lock()
	dc.closed = true
	dc.mu.Unlock()

	closeResult := <-dc.conn.Close()
	if closeResult.GetErr() != nil {
		return ErrClosing
	}
	return nil
}

type connRequest struct {
	conn *driverConn
	err  error
}

func NewConnPool(driver plc4go.PlcDriverManager, plcURI string) *ConnPool {
	return &ConnPool{
		plcURI:       plcURI,
		plcDriver:    driver,
		connRequests: make(map[uint64]chan connRequest),
	}
}
