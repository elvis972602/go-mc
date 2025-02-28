package bot

import (
	"errors"
	"fmt"

	"github.com/Tnze/go-mc/data/packetid"
	pk "github.com/Tnze/go-mc/net/packet"
)

// HandleGame receive server packet and response them correctly.
// Note that HandleGame will block if you don't receive from Events.
func (c *Client) HandleGame() error {
	var p pk.Packet
	for {
		// Read packets
		if err := c.Conn.ReadPacket(&p); err != nil {
			return err
		}

		if p.ID == int32(packetid.BundleDelimiter) {
			err := c.handleBundlePackets()
			if err != nil {
				return err
			}
		} else {
			// handle packets
			err := c.handlePacket(p)
			if err != nil {
				return err
			}
		}
	}
}

func (c *Client) handleBundlePackets() (err error) {
	var packets []pk.Packet
	var packetNames []string
	for i := 0; i < 4096; i++ {
		var p pk.Packet
		// Read packets
		if err := c.Conn.ReadPacket(&p); err != nil {
			return err
		}

		if p.ID == int32(packetid.BundleDelimiter) {
			// bundle finished
			goto handlePackets
		}

		packets = append(packets, p)
		packetNames = append(packetNames, packetid.ClientboundPacketID(p.ID).String())
	}
	return errors.New("packet number of a bundle out of limit")

handlePackets:
	for i := range packets {
		if err := c.handlePacket(packets[i]); err != nil {
			return err
		}
	}
	return nil
}

type PacketHandlerError struct {
	ID  packetid.ClientboundPacketID
	Err error
}

func (d PacketHandlerError) Error() string {
	return fmt.Sprintf("handle packet %v error: %v", d.ID, d.Err)
}

func (d PacketHandlerError) Unwrap() error {
	return d.Err
}

func (c *Client) handlePacket(p pk.Packet) (err error) {
	packetID := packetid.ClientboundPacketID(p.ID)
	for _, handler := range c.Events.generic {
		if err = handler.F(p); err != nil {
			return PacketHandlerError{ID: packetID, Err: err}
		}
	}
	for _, handler := range c.Events.handlers[packetID] {
		err = handler.F(p)
		if err != nil {
			return PacketHandlerError{ID: packetID, Err: err}
		}
	}
	return
}
