package tcpclient

import (
	"sync"
	"testing"
	"time"

	"github.com/dachad/tcpgoon/tcpserver"
)

// We really need to refactor this test. We should verify connections do become established,
// rather than just waiting for a second and finish
// We should also test "failing" connections, and ensure their status is reported properly
func TestTCPConnectEstablished(t *testing.T) {
	var host = "127.0.0.1"
	var port = 55555

	dispatcher := &tcpserver.Dispatcher{
		Handlers: make(map[string]*tcpserver.Handler),
		Lock:     sync.RWMutex{},
	}

	runTCPServer := func() {
		t.Log("Starting TCP server...")
		if err := dispatcher.ListenHandlers(port); err != nil {
			t.Fatal("Could not start the TCP server", err)
			return
		}
	}
	go runTCPServer()
	time.Sleep(500 * time.Millisecond)

	defer func() {
		err := recover()
		if err != "sync: negative WaitGroup counter" {
			t.Fatalf("Unexpected panic: %#v", err)
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)

	var statusChannel = make(chan Connection, 2)
	var closeRequest = make(chan bool)

	// We use a different subroutine to be able to reach the closeRequest command
	t.Log("Initiating TCP Connect")
	go TCPConnect(1, host, port, &wg, statusChannel, closeRequest)
	if (<-statusChannel).GetConnectionStatus() == ConnectionDialing {
		t.Log("Connection Dialing")
	} else {
		t.Error("Connection failed to dial")
	}
	var connectionEstablished = <-statusChannel
	if connectionEstablished.GetConnectionStatus() == ConnectionEstablished {
		t.Log("Connection Established")
	} else {
		t.Error("Connection failed to establish")
	}
	if connectionEstablished.GetTCPProcessingDuration() != 0 {
		t.Log("Connection Estalished in ", connectionEstablished.GetTCPProcessingDuration())
	} else {
		t.Error("Connection TCP Processing Duration not consistent")
	}

	// We ask to close the TCP connection
	closeRequest <- true
	time.Sleep(100 * time.Millisecond)

	// Validates wg has been decreased to 0, and next one is making it negative
	wg.Done()
	t.Fatal("Should panic")
}

func TestTCPConnectErrored(t *testing.T) {
	var host = "127.0.0.1"
	var port = 55556

	defer func() {
		err := recover()
		if err != "sync: negative WaitGroup counter" {
			t.Fatalf("Unexpected panic: %#v", err)
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)

	var statusChannel = make(chan Connection, 2)
	var closeRequest = make(chan bool)

	t.Log("Initiating TCP Connect")
	TCPConnect(1, host, port, &wg, statusChannel, closeRequest)
	if (<-statusChannel).GetConnectionStatus() == ConnectionDialing {
		t.Log("Connection Dialing")
	} else {
		t.Error("Connection failed to dial")
	}

	var connectionErrored = <-statusChannel

	if connectionErrored.GetConnectionStatus() == ConnectionError {
		t.Log("Connection Errored")
	} else {
		t.Error("Connection not errored")
	}
	if connectionErrored.GetTCPProcessingDuration() != 0 {
		t.Log("Connection Errored in ", connectionErrored.GetTCPProcessingDuration())
	} else {
		t.Error("Connection TCP Processing Duration not consistent")
	}

	// Validates wg has been decreased to 0, and next one is making it negative
	wg.Done()
	t.Fatal("Should panic")
}

func TestReportConnectionStatus(t *testing.T) {
	connStatusCh := make(chan Connection, 1)
	connectionDescription := Connection{
		ID:      0,
		status:  ConnectionDialing,
		metrics: connectionMetrics{},
	}
	reportConnectionStatus(connStatusCh, connectionDescription)
	if <-connStatusCh != connectionDescription {
		t.Error("Not proper Connection reported: ", <-connStatusCh)
	}
}
