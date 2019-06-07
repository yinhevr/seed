package seed

import (
	"context"
	"github.com/godcong/go-ipfs-restapi"
	"github.com/yinhevr/seed/model"
	"golang.org/x/xerrors"
	"strings"
	"sync"
	"time"
)

var rest *shell.Shell

// Rest ...
func Rest() *shell.Shell {
	if rest == nil {
		rest = shell.NewShell("localhost:5001")
	}
	return rest
}

// InitShell ...
func InitShell(s string) {
	log.Info("ipfs shell:", s)
	rest = shell.NewShell(s)
}

// QuickConnect ...
func QuickConnect(addr string) {
	var e error
	go func() {
		for {
			e = SwarmConnect(addr)
			if e != nil {
				return
			}
			time.Sleep(30 * time.Second)
		}
	}()
}

var swarms = sync.Pool{}

// PoolSwarmConnect ...
func PoolSwarmConnect() {
	SwarmAdd(&model.SourcePeer{
		SourcePeerDetail: &model.SourcePeerDetail{
			Addr: "/ip4/47.101.169.94/tcp/4001",
			Peer: "QmeF1HVnBYTzFFLGm4VmAsHM4M7zZS3WUYx62PiKC2sqRq",
		},
	})
	log.Info("PoolSwarmConnect running")
	for {
		if s := swarms.Get(); s != nil {
			sp, b := s.(*model.SourcePeer)
			if b {
				e := SwarmConnect(swarmAddress(sp))
				log.Info(swarmAddress(sp))
				if e != nil {
					log.Error("swarm connect err:", e)
				}
				swarms.Put(sp)
			}
			time.Sleep(30 * time.Second)
			continue
		}
		time.Sleep(5 * time.Second)
	}
}

// SwarmAdd ...
func SwarmAdd(sp *model.SourcePeer) {
	swarms.Put(sp)
}

// SwarmAddAddress ...
func SwarmAddAddress(addr string) {
	swarms.Put(AddressSwarm(addr))
}

// SwarmAddList ...
func SwarmAddList(sps []*model.SourcePeer) {
	for _, v := range sps {
		SwarmAdd(v)
	}
}

// SwarmConnect ...
func SwarmConnect(addr string) (e error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	log.Info("connect to:", addr)
	if rest == nil {
		return xerrors.New("rest is not inited")
	}
	if err := rest.SwarmConnect(ctx, addr); err != nil {
		return err
	}
	return
}
func swarmAddress(peer *model.SourcePeer) string {
	if peer != nil {
		return peer.Addr + "/ipfs/" + peer.Peer
	}
	return ""
}

// AddressSwarm ...
func AddressSwarm(address string) (peer *model.SourcePeer) {
	ss := strings.Split(address, "/")
	size := len(ss)
	log.Info("address:", address)
	log.Info("size:", size)
	if size < 7 {
		return &model.SourcePeer{}
	}
	return &model.SourcePeer{
		SourcePeerDetail: &model.SourcePeerDetail{
			Addr: strings.Join(ss[:size-2], "/"),
			Peer: ss[size-1],
		},
	}
}

func swarmConnectTo(peer *model.SourcePeer) (e error) {
	address := swarmAddress(peer)
	if address == "" {
		return xerrors.New("null address")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	log.Info("connect to:", address)
	if err := rest.SwarmConnect(ctx, address); err != nil {
		return err
	}
	return
}

func swarmConnects(peers []*model.SourcePeer) {
	if peers == nil {
		return
	}

	var nextPeers []*model.SourcePeer
	for _, value := range peers {
		e := swarmConnectTo(value)
		if e != nil {
			//log.Error(e)
			time.Sleep(30 * time.Second)
			continue
		}
		//filter the error peers
		nextPeers = append(nextPeers, value)
		time.Sleep(30 * time.Second)
	}
	//rerun when connect is end
	swarmConnects(nextPeers)
}
