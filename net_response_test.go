package main

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/morfien101/influxLineProtocolOutput"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helpers
func UDPServer(t *testing.T, wg *sync.WaitGroup) {
	udpAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:2004")
	conn, _ := net.ListenUDP("udp", udpAddr)
	wg.Done()
	buf := make([]byte, 1024)
	_, remoteaddr, _ := conn.ReadFromUDP(buf)
	conn.WriteToUDP(buf, remoteaddr)
	conn.Close()
	wg.Done()
}

func TCPServer(t *testing.T, wg *sync.WaitGroup) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:2004")
	tcpServer, _ := net.ListenTCP("tcp", tcpAddr)
	wg.Done()
	conn, _ := tcpServer.AcceptTCP()
	buf := make([]byte, 1024)
	conn.Read(buf)
	conn.Write(buf)
	conn.CloseWrite()
	tcpServer.Close()
	wg.Done()
}

func TestBadProtocol(t *testing.T) {
	// Init plugin
	c := NetResponse{
		Protocol: "unknownprotocol",
		Address:  ":9999",
	}
	// Error
	err1 := c.gather(influxLineProtocolOutput.New(c.Name))
	require.Error(t, err1)
	assert.Equal(t, "Bad protocol", err1.Error())
}

func TestNoPort(t *testing.T) {
	c := NetResponse{
		Protocol: "tcp",
		Address:  ":",
	}
	err1 := c.gather(influxLineProtocolOutput.New(c.Name))
	require.Error(t, err1)
	assert.Equal(t, "Bad port", err1.Error())
}

func TestAddressOnly(t *testing.T) {
	c := NetResponse{
		Protocol: "tcp",
		Address:  "127.0.0.1",
	}
	err1 := c.gather(influxLineProtocolOutput.New(c.Name))
	require.Error(t, err1)
	assert.Equal(t, "address 127.0.0.1: missing port in address", err1.Error())
}

func TestSendExpectStrings(t *testing.T) {
	tc := NetResponse{
		Name:     "tcp",
		Protocol: "udp",
		Address:  "127.0.0.1:7",
		Send:     "",
		Expect:   "toast",
	}
	uc := NetResponse{
		Name:     "udp",
		Protocol: "udp",
		Address:  "127.0.0.1:7",
		Send:     "toast",
		Expect:   "",
	}
	err1 := tc.gather(influxLineProtocolOutput.New(tc.Name))
	require.Error(t, err1)
	assert.Equal(t, "Send string cannot be empty", err1.Error())
	err2 := uc.gather(influxLineProtocolOutput.New(uc.Name))
	require.Error(t, err2)
	assert.Equal(t, "Expected string cannot be empty", err2.Error())
}

func TestTCPError(t *testing.T) {
	// Init plugin
	c := NetResponse{
		Name:     "net_response",
		Protocol: "tcp",
		Address:  ":9999",
	}
	// Error
	container := influxLineProtocolOutput.New(c.Name)
	err1 := c.gather(container)
	require.NoError(t, err1)

	expectedTags := map[string]string{
		"server":      "",
		"port":        "9999",
		"protocol":    "tcp",
		"result_text": "connection_failed",
	}
	expectedValues := map[string]interface{}{
		"result_code": 1,
		"result_type": "connection_failed",
	}

	if err := container.Contains(expectedTags, expectedValues); err != nil {
		t.Errorf("TestTCPError failed. Error: %s", err)
	}
}

func TestTCPOK1(t *testing.T) {
	var wg sync.WaitGroup
	// Init plugin
	c := NetResponse{
		Name:        "net_responce",
		Address:     "127.0.0.1:2004",
		Send:        "test",
		Expect:      "test",
		ReadTimeout: Duration{Duration: time.Second * 3},
		Timeout:     Duration{Duration: time.Second},
		Protocol:    "tcp",
	}
	// Start TCP server
	wg.Add(1)
	go TCPServer(t, &wg)
	wg.Wait()
	// Connect
	wg.Add(1)
	container := influxLineProtocolOutput.New("net_response")
	err1 := c.gather(container)
	wg.Wait()
	// Override response time, due to local testing we get unpredictable response times.
	container.Values["response_time"] = 1.0

	require.NoError(t, err1)

	expectedValues := map[string]interface{}{
		"result_code":   0,
		"result_type":   "success",
		"string_found":  true,
		"response_time": 1.0,
	}
	expectedTags := map[string]string{
		"result_text": "success",
		"server":      "127.0.0.1",
		"port":        "2004",
		"protocol":    "tcp",
	}
	if err := container.Contains(expectedTags, expectedValues); err != nil {
		t.Errorf("TCP success doesn't have to the correct values. Error: %s", err)
	}
	// Waiting TCPserver
	wg.Wait()
}

func TestTCPOK2(t *testing.T) {
	var wg sync.WaitGroup
	// Init plugin
	c := NetResponse{
		Address:     "127.0.0.1:2004",
		Send:        "test",
		Expect:      "test2",
		ReadTimeout: Duration{Duration: time.Second * 3},
		Timeout:     Duration{Duration: time.Second},
		Protocol:    "tcp",
	}
	// Start TCP server
	wg.Add(1)
	go TCPServer(t, &wg)
	wg.Wait()
	// Connect
	wg.Add(1)
	container := influxLineProtocolOutput.New("net_response")
	err1 := c.gather(container)
	wg.Wait()
	// Override response time, due to local testing we get unpredictable response times.
	container.Values["response_time"] = 1.0

	require.NoError(t, err1)
	expectedValues := map[string]interface{}{
		"result_code":   1,
		"result_type":   "string_mismatch",
		"string_found":  false,
		"response_time": 1.0,
	}
	expectedTags := map[string]string{
		"result_text": "string_mismatch",
		"server":      "127.0.0.1",
		"port":        "2004",
		"protocol":    "tcp",
	}
	if err := container.Contains(expectedTags, expectedValues); err != nil {
		t.Errorf("TCP success doesn't have to the correct values. Error: %s", err)
	}
	// Waiting TCPserver
	wg.Wait()
}

func TestUDPError(t *testing.T) {
	// Init plugin
	c := NetResponse{
		Address:  ":9999",
		Send:     "test",
		Expect:   "test",
		Protocol: "udp",
	}
	// Gather
	container := influxLineProtocolOutput.New("net_response")
	err1 := c.gather(container)
	// Override response time, due to local testing we get unpredictable response times.
	container.Values["response_time"] = 1.0

	// Error
	require.NoError(t, err1)
	expectedValues := map[string]interface{}{
		"result_code":   1,
		"result_type":   "read_failed",
		"response_time": 1.0,
	}
	expectedTags := map[string]string{
		"result_text": "read_failed",
		"server":      "",
		"port":        "9999",
		"protocol":    "udp",
	}
	if err := container.Contains(expectedTags, expectedValues); err != nil {
		t.Errorf("UDP Error failed to set the correct values. Error: %s", err)
	}
}

func TestUDPOK1(t *testing.T) {
	var wg sync.WaitGroup
	// Init plugin
	c := NetResponse{
		Address:     "127.0.0.1:2004",
		Send:        "test",
		Expect:      "test",
		ReadTimeout: Duration{Duration: time.Second * 3},
		Timeout:     Duration{Duration: time.Second},
		Protocol:    "udp",
	}
	// Start UDP server
	wg.Add(1)
	go UDPServer(t, &wg)
	wg.Wait()
	// Connect
	wg.Add(1)
	container := influxLineProtocolOutput.New("net_response")
	err1 := c.gather(container)
	wg.Wait()
	// Override response time, due to local testing we get unpredictable response times.
	container.Values["response_time"] = 1.0

	require.NoError(t, err1)

	expectedValues := map[string]interface{}{
		"result_code":   0,
		"result_type":   "success",
		"string_found":  true,
		"response_time": 1.0,
	}
	expectedTags := map[string]string{
		"result_text": "success",
		"server":      "127.0.0.1",
		"port":        "2004",
		"protocol":    "udp",
	}
	// Waiting UDPserver
	wg.Wait()
	if err := container.Contains(expectedTags, expectedValues); err != nil {
		t.Errorf("UDP Success failed to set the correct values. Error: %s", err)
	}
}
