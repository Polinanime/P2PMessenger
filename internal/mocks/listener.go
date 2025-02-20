package mocks

import (
	"fmt"
	"net"
)

type MockListener struct {
	Conn   net.Conn
	Closed bool
}

func NewMockListener(conn net.Conn) *MockListener {
	return &MockListener{
		Conn:   conn,
		Closed: false,
	}
}

func (m *MockListener) Accept() (net.Conn, error) {
	if m.Closed {
		return nil, fmt.Errorf("listener is closed")
	}
	return m.Conn, nil
}

func (m *MockListener) Close() error {
	m.Closed = true
	return nil
}

func (m *MockListener) Addr() net.Addr {
	if m.Conn != nil {
		return m.Conn.LocalAddr()
	}
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
}
