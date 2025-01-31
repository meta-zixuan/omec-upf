// SPDX-License-Identifier: Apache-2.0
// Copyright 2020 Intel Corporation

package pfcpiface

import (
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type EndMarker struct {
	TEID     uint32
	PeerIP   net.IP
	PeerPort uint16
}

// CreateFAR appends far to existing list of FARs in the session.
func (s *PFCPSession) CreateFAR(f far) {
	s.Fars = append(s.Fars, f)
}

func addEndMarkerForGtp(farItem far, endMarkerList *[]EndMarker) {
	newEndMarker := EndMarker{
		TEID:     farItem.tunnelTEID,
		PeerIP:   int2ip(farItem.tunnelIP4Dst),
		PeerPort: farItem.tunnelPort,
	}
	*endMarkerList = append(*endMarkerList, newEndMarker)
}

func addEndMarker(farItem far, endMarkerList *[][]byte) {
	// This time lets fill out some information
	log.Info("Adding end Marker for farID : ", farItem.farID)

	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}
	buffer := gopacket.NewSerializeBuffer()
	ipLayer := &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    int2ip(farItem.tunnelIP4Src),
		DstIP:    int2ip(farItem.tunnelIP4Dst),
		Protocol: layers.IPProtocolUDP,
	}
	ethernetLayer := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xFF, 0xAA, 0xFA, 0xAA, 0xFF, 0xAA},
		DstMAC:       net.HardwareAddr{0xBD, 0xBD, 0xBD, 0xBD, 0xBD, 0xBD},
		EthernetType: layers.EthernetTypeIPv4,
	}
	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(2152),
		DstPort: layers.UDPPort(2152),
	}

	err := udpLayer.SetNetworkLayerForChecksum(ipLayer)
	if err != nil {
		log.Info("set checksum for UDP layer in endmarker failed")
		return
	}

	gtpLayer := &layers.GTPv1U{
		Version:      1,
		MessageType:  254,
		ProtocolType: farItem.tunnelType,
		TEID:         farItem.tunnelTEID,
	}
	// And create the packet with the layers
	err = gopacket.SerializeLayers(buffer, options,
		ethernetLayer,
		ipLayer,
		udpLayer,
		gtpLayer,
	)

	if err == nil {
		outgoingPacket := buffer.Bytes()
		*endMarkerList = append(*endMarkerList, outgoingPacket)
	} else {
		log.Info("go packet serialize failed : ", err)
	}
}

// UpdateFAR updates existing far in the session.
func (s *PFCPSession) UpdateFAR(f *far, endMarkerList *[]EndMarker) error {
	for idx, v := range s.Fars {
		if v.farID == f.farID {
			if f.sendEndMarker {
				addEndMarkerForGtp(v, endMarkerList)
			}

			s.Fars[idx] = *f

			return nil
		}
	}

	return ErrNotFound("FAR")
}

// RemoveFAR removes far from existing list of FARs in the session.
func (s *PFCPSession) RemoveFAR(id uint32) (*far, error) {
	for idx, v := range s.Fars {
		if v.farID == id {
			s.Fars = append(s.Fars[:idx], s.Fars[idx+1:]...)
			return &v, nil
		}
	}

	return nil, ErrNotFound("FAR")
}
