package loralogger

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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
		config:   c,
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
	}).Info("Logging packet to DynamoDB")

	creds := credentials.NewSharedCredentials(m.config.CredentialsPath, m.config.CredentialsProfile)

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(m.config.Region),
		Credentials: creds},
	)

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	currentTime := time.Now()

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":g": {
				S: aws.String(gatewayID),
			},
			":p": {
				S: aws.String(base64.StdEncoding.EncodeToString(up.data)),
			},
			":e": {
				N: aws.Int(currentTime.AddDate(0, 0, 14).Unix()),
			},
		},
		TableName: aws.String(m.config.Table),
		Key: map[string]*dynamodb.AttributeValue{
			"item": {
				S: aws.String("raw#" + currentTime.Format("2006-01-02")),
			},
			"date_or_time": {
				S: aws.String(currentTime.Format("15:04:05.000000")),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set gateway_id = :g, packet = :p, expires = :e"),
	}

	_, err = svc.UpdateItem(input)

	if err != nil {
		fmt.Println(err.Error())
		return errors.Wrap(err, "DynamoDB error")
	}

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
