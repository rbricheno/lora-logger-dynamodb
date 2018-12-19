package loralogger

import (
	"bytes"
	"encoding/base64"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/utahta/go-cronowriter"
)

type udpPacket struct {
	addr *net.UDPAddr
	data []byte
}

// LoraLogger logs UDP packets
type LoraLogger struct {
	sync.RWMutex
	wg sync.WaitGroup

	conn   *net.UDPConn
	config Config
	closed bool

	//backends map[string]map[string]*net.UDPConn // [backendHost][gatewayID]UDPConn
	gateways map[string]*net.UDPAddr // [gatewayID]UDPAddr
}

// New creates a new loralogger.
func New(c Config) (*LoraLogger, error) {
	m := LoraLogger{
		//backends: make(map[string]map[string]*net.UDPConn),
		gateways: make(map[string]*net.UDPAddr),
	}

	addr, err := net.ResolveUDPAddr("udp", c.Bind)
	if err != nil {
		return nil, errors.Wrap(err, "resolve udp addr error")
	}

	log.WithField("addr", addr).Info("starting listener")
	m.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "listen udp error")
	}

	go func() {
		m.wg.Add(1)
		err := m.readUplinkPackets()
		if !m.isClosed() {
			log.WithError(err).Error("read udp packets error")
		}
		m.wg.Done()
	}()

	return &m, nil
}

// Closes the loralogger.
func (m *LoraLogger) Close() error {
	m.Lock()
	m.closed = true

	log.Info("closing listener")
	if err := m.conn.Close(); err != nil {
		return errors.Wrap(err, "close udp listener error")
	}

	m.Unlock()
	m.wg.Wait()
	return nil
}

func (m *LoraLogger) isClosed() bool {
	m.RLock()
	defer m.RUnlock()
	return m.closed
}

func (m *LoraLogger) setGateway(gatewayID string, addr *net.UDPAddr) error {
	m.Lock()
	defer m.Unlock()
	m.gateways[gatewayID] = addr
	return nil
}

func (m *LoraLogger) getGateway(gatewayID string) (*net.UDPAddr, error) {
	m.RLock()
	defer m.RUnlock()

	addr, ok := m.gateways[gatewayID]
	if !ok {
		return nil, errors.New("gateway does not exist")
	}
	return addr, nil
}

func (m *LoraLogger) readUplinkPackets() error {
	buf := make([]byte, 65507) // max udp data size
	for {
		i, addr, err := m.conn.ReadFromUDP(buf)
		if err != nil {
			if m.isClosed() {
				return nil
			}

			log.WithError(err).Error("read from udp error")
			continue
		}

		data := make([]byte, i)
		copy(data, buf[:i])
		up := udpPacket{data: data, addr: addr}

		// handle packet async
		go func(up udpPacket) {
			if err := m.handleUplinkPacket(up); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"data_base64": base64.StdEncoding.EncodeToString(up.data),
					"addr":        up.addr,
				}).Error("could not handle packet")
			}
		}(up)
	}
}

func (m *LoraLogger) handleUplinkPacket(up udpPacket) error {
	pt, err := GetPacketType(up.data)
	if err != nil {
		return errors.Wrap(err, "get packet-type error")
	}

	gatewayID, err := GetGatewayID(up.data)
	if err != nil {
		return errors.Wrap(err, "get gateway id error")
	}

	log.WithFields(log.Fields{
		"from_addr":   up.addr,
		"gateway_id":  gatewayID,
		"packet_type": pt,
	}).Info("Logging packet to text file")

	w := cronowriter.MustNew("/var/log/loralogger/%Y/%m/%d/lora.log")

	buffer := bytes.Buffer{}
	buffer.WriteString(time.Now().Format(time.RFC3339))
	buffer.WriteString(", ")
	buffer.WriteString(gatewayID)
	buffer.WriteString(", ")
	buffer.WriteString(base64.StdEncoding.EncodeToString(up.data))
	buffer.WriteString("\n")

	w.Write([]byte(buffer.String()))

	return nil
}
