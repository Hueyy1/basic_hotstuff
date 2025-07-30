package service

import (
	"errors"
	"fmt"
	"github.com/relab/gorums"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/clientpb"
	"hxy352/src/types"
	"net"
	"sync"
)

type NodeManager struct {
	gConf *model.Config

	// calculate quorum size
	F                 int
	QuorumSize        int // 2 * F + 1
	TimeoutQuorumSize int // F + 1, used for cogsworth
	TotalNodesNumber  int
	ActualFaultNumber int
	FaultNodesMap     map[types.ID]bool

	// init nodes conn
	onceReplica sync.Once
	onceClient  sync.Once
	Nodes       []*basichotstuffpb.Node
	NodesCfg    *basichotstuffpb.Configuration
	Client      *clientpb.Node
}

func NewNodeManager(gConf *model.Config) *NodeManager {
	nm := &NodeManager{
		gConf:         gConf,
		FaultNodesMap: make(map[types.ID]bool),

		onceReplica: sync.Once{},
		onceClient:  sync.Once{},
	}
	nm.Init()
	return nm
}

func (nm *NodeManager) Init() {
	nm.TotalNodesNumber = nm.gConf.TotalNumber
	nm.ActualFaultNumber = nm.gConf.FaultNumber
	nm.CalQuorumSize()
	nm.SetFaultNodes()
}

func (nm *NodeManager) CalQuorumSize() {
	if nm.gConf.TotalNumber < 4 {
		log.Panic(errors.New("total node number must gte 4"))
	}

	if nm.gConf.TotalNumber > 16 {
		log.Panic(errors.New("total node number must lte 16"))
	}

	nm.F = (nm.gConf.TotalNumber - 1) / 3
	nm.QuorumSize = 2*nm.F + 1
	nm.TimeoutQuorumSize = nm.F + 1
}

func (nm *NodeManager) IsFaultNode() bool {
	return nm.FaultNodesMap[nm.gConf.Id]
}

func (nm *NodeManager) SetFaultNodes() {
	if nm.ActualFaultNumber == 0 {
		return
	}

	// F    1/2/3/ 4/ 5
	// node 3/6/9/12/15
	for i := 0; i < nm.ActualFaultNumber; i++ {
		tmp := types.ID((i + 1) * 3)
		nm.FaultNodesMap[tmp] = true
	}
}

func (nm *NodeManager) GetNodes() []*basichotstuffpb.Node {
	nm.onceReplica.Do(nm.InitAllReplicaClients)
	return nm.Nodes
}

func (nm *NodeManager) InitAllReplicaClients() {

	// todo: bug, grpc retry

	mgr := basichotstuffpb.NewManager(
		gorums.WithGrpcDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	var adds []string
	for _, config := range nm.gConf.Replica[0:nm.TotalNodesNumber] {
		if types.ID(config.Id) == nm.gConf.Id {
			// Skip myself
			continue
		} else {
			adds = append(adds, fmt.Sprintf("%s:%d", config.Host, config.Port))
		}
	}

	allNodesConfig, err := mgr.NewConfiguration(gorums.WithNodeList(adds))
	if err != nil {
		log.Panic(err)
	}

	nm.Nodes = make([]*basichotstuffpb.Node, len(adds))
	nm.Nodes = allNodesConfig.Nodes()
	nm.NodesCfg = allNodesConfig

	log.Infof("InitAllReplicaClients finished")
}

func (nm *NodeManager) InitClient() {
	mgr := clientpb.NewManager(
		gorums.WithGrpcDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	var adds []string
	adds = append(adds, fmt.Sprintf("%s:%d", nm.gConf.Client.Host, nm.gConf.Client.Port))

	allNodesConfig, err := mgr.NewConfiguration(gorums.WithNodeList(adds))
	if err != nil {
		log.Panic(err)
	}

	nm.Client = allNodesConfig.Nodes()[0]

	log.Infof("InitClient finished")
}

func (nm *NodeManager) GetClient() *clientpb.Node {
	nm.onceClient.Do(nm.InitClient)
	return nm.Client
}

func (nm *NodeManager) GetNodesCfg() *basichotstuffpb.Configuration {
	nm.onceReplica.Do(nm.InitAllReplicaClients)
	return nm.NodesCfg
}

func (nm *NodeManager) GetLeaderByView(cView types.View) types.ID {
	return types.ID(int(cView) % nm.TotalNodesNumber)
}

// GetLeaderAddress get leader address
func (nm *NodeManager) GetLeaderAddress(cView types.View) string {
	leaderIdx := int(nm.GetLeaderByView(cView))
	return fmt.Sprintf("%s:%d", nm.gConf.Replica[leaderIdx].Host, nm.gConf.Replica[leaderIdx].Port)
}

// GetLeaderNode get leader node
func (nm *NodeManager) GetLeaderNode(cView types.View) *basichotstuffpb.Node {
	leaderAddress := nm.GetLeaderAddress(cView)
	a1, _ := normalizeAddr(leaderAddress)

	for _, node := range nm.GetNodes() {
		//log.Debugf("node.Address: %s, leaderAddress: %s", node.Address(), leaderAddress)
		a2, _ := normalizeAddr(node.Address())
		if a1 == a2 {
			return node
		}
	}
	return nil
}

func normalizeAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(ipAddr.IP.String(), port), nil
}
