package bridge

import (
	"context"

	"github.com/kleeedolinux/gorilix/cluster"
	"github.com/kleeedolinux/gorilix/system"
)


type ClusterAdapter struct {
	*cluster.Cluster
}


func (a *ClusterAdapter) Self() system.Node {
	return &NodeAdapter{a.Cluster.Self()}
}


func (a *ClusterAdapter) Members() []system.Node {
	clusterMembers := a.Cluster.Members()
	systemNodes := make([]system.Node, len(clusterMembers))

	for i, member := range clusterMembers {
		systemNodes[i] = &NodeAdapter{member}
	}

	return systemNodes
}


type NodeAdapter struct {
	*cluster.Node
}


func (n *NodeAdapter) GetName() string {
	return n.Node.Name
}


func (n *NodeAdapter) GetAddress() string {
	return n.Node.Addr.String()
}


func (n *NodeAdapter) GetPort() uint16 {
	return n.Node.Port
}


func (n *NodeAdapter) GetStatus() int {
	return int(n.Node.Status)
}


type ClusterProvider struct{}


func NewClusterProvider() *ClusterProvider {
	return &ClusterProvider{}
}


func (p *ClusterProvider) NewCluster(config *system.ClusterConfig, sys interface{}) (system.Cluster, error) {
	actorSystem, ok := sys.(*system.ActorSystem)
	if !ok {
		return nil, cluster.ErrInvalidSystemReference
	}

	
	clusterConfig := &cluster.ClusterConfig{
		NodeName:     config.NodeName,
		BindAddr:     config.BindAddr,
		BindPort:     config.BindPort,
		Seeds:        config.Seeds,
		GossipNodes:  3,
		PushInterval: cluster.DefaultConfig().PushInterval,
		PullInterval: cluster.DefaultConfig().PullInterval,
	}

	
	clusterInstance := cluster.NewCluster(clusterConfig, &systemAdapter{actorSystem})
	return &ClusterAdapter{clusterInstance}, nil
}


type systemAdapter struct {
	system *system.ActorSystem
}


func (a *systemAdapter) SendMessage(ctx context.Context, actorID string, message interface{}) error {
	return a.system.SendMessage(ctx, actorID, message)
}


func (a *systemAdapter) SendNamedMessage(ctx context.Context, name string, message interface{}) error {
	return a.system.SendNamedMessage(ctx, name, message)
}
